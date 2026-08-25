package parser

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	reQuantityStart             = regexp.MustCompile(`^\d`)
	reInstructionNum            = regexp.MustCompile(`^\d+[\.\)]\s`)
	reMetaField                 = regexp.MustCompile(`(?i)^(Prep Time|Cook Time|Total Time|Additional Time|Servings|Yield|Category|Cuisine|Diet|Author|Baking Time|Bake Time|Rise Time|Rest Time|Chill Time|Preparation Time)[\s:]+`)
	reTrailingIngredientsSuffix = regexp.MustCompile(`(?i)\s+ingredients\s*$`)
	reHasDigit                  = regexp.MustCompile(`\d`)
)

// sectionKeywords are only matched when they are the SOLE content of a line.
var sectionKeywords = map[string]string{
	"ingredients":     "ingredients",
	"ingredient list": "ingredients",
	"directions":      "instructions",
	"instructions":    "instructions",
	"method":          "instructions",
	"steps":           "instructions",
	"preparation":     "instructions",
	"how to make":     "instructions",
}

// ingredientSubsectionPatterns match subsection headers within ingredients.
var ingredientSubsectionPatterns = []struct {
	pattern *regexp.Regexp
	phase   string // empty means derive from capture group
}{
	{regexp.MustCompile(`(?i)^starter\s+build\s*[-–]?$`), "starter build"},
	{regexp.MustCompile(`(?i)^levain\s*[-–]?$`), "levain"},
	{regexp.MustCompile(`(?i)^final\s+dough(\s+ingredients)?\s*[-–]?$`), "final dough"},
	{regexp.MustCompile(`(?i)^(dough)\s*[-–]?$`), "dough"},
	{regexp.MustCompile(`(?i)^scald\s*[-–]?$`), "scald"},
	{regexp.MustCompile(`(?i)^tangzhong\s*[-–]?$`), "tangzhong"},
	{regexp.MustCompile(`(?i)^yudane\s*[-–]?$`), "yudane"},
	{regexp.MustCompile(`(?i)^for\s+the\s+(.+?)\s*[-–]?$`), ""}, // derive from capture
	{regexp.MustCompile(`(?i)^(.+?)\s+ingredients\s*[-–]?$`), ""},
	{regexp.MustCompile(`(?i)^(topping|filling|sauce|pesto)\s*[-–]?$`), ""},
}

var bakersPctPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)baker'?s\s+percentages?`),
	regexp.MustCompile(`(?i)baker'?s\s+%`),
}

// lowerConnectors are short words skipped when checking title case, so
// "Main Dough" and "For the Pesto" style headers don't need every word capitalized.
var lowerConnectors = map[string]bool{
	"of": true, "the": true, "and": true, "a": true, "an": true,
	"in": true, "on": true, "to": true, "or": true, "for": true, "&": true,
}

// looksLikeTitleCase reports whether every significant word (i.e. not a short
// connector) starts with an uppercase letter. Real subsection headers in these
// recipes are reliably Title Case ("Main Dough", "Stiff Sweet Starter"); bare
// ingredient references are sentence-case (only the line's first word
// capitalized, e.g. "Pinch of salt", "Olive oil") — this is what lets the
// generic fallback tell them apart.
func looksLikeTitleCase(line string) bool {
	words := strings.Fields(line)
	significant, capped := 0, 0
	for _, w := range words {
		if lowerConnectors[strings.ToLower(w)] {
			continue
		}
		significant++
		r, _ := utf8.DecodeRuneInString(w)
		if unicode.IsUpper(r) {
			capped++
		}
	}
	return significant > 0 && capped == significant
}

// DetectSections parses a normalised recipe string into a SectionMap.
func DetectSections(cleaned string) SectionMap {
	sm := SectionMap{}

	lines := strings.Split(cleaned, "\n")

	// NoLineBreaks detection
	newlineCount := strings.Count(cleaned, "\n")
	if newlineCount < 3 {
		sm.NoLineBreaks = true
	}

	type section int
	const (
		secDescription section = iota
		secIngredients
		secInstructions
	)

	current := secDescription
	var currentGroup *IngredientGroup
	skipBakersPct := false

	var pendingPhase string
	var pendingLines []string
	hasPending := false

	flushPending := func() {
		if !hasPending {
			return
		}
		confirmed := false
		for _, l := range pendingLines {
			if reHasDigit.MatchString(l) {
				confirmed = true
				break
			}
		}
		if confirmed {
			sm.IngredientGroups = discardEmptyGroups(sm.IngredientGroups)
			g := IngredientGroup{Phase: pendingPhase, Lines: pendingLines}
			sm.IngredientGroups = append(sm.IngredientGroups, g)
			currentGroup = &sm.IngredientGroups[len(sm.IngredientGroups)-1]
		} else {
			if currentGroup == nil {
				g := IngredientGroup{Phase: "dough"}
				sm.IngredientGroups = append(sm.IngredientGroups, g)
				currentGroup = &sm.IngredientGroups[len(sm.IngredientGroups)-1]
			}
			currentGroup.Lines = append(currentGroup.Lines, pendingPhase)
			currentGroup.Lines = append(currentGroup.Lines, pendingLines...)
			currentGroup = &sm.IngredientGroups[len(sm.IngredientGroups)-1]
		}
		pendingPhase = ""
		pendingLines = nil
		hasPending = false
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		lowerTrimmed := strings.ToLower(strings.Trim(trimmed, ":– \t"))

		// Baker's percentage block detection — skip until next known section
		if isBakersPctHeader(trimmed) {
			skipBakersPct = true
			continue
		}
		if skipBakersPct {
			if dest, ok := sectionKeywords[lowerTrimmed]; ok {
				skipBakersPct = false
				switch dest {
				case "ingredients":
					current = secIngredients
					if currentGroup == nil {
						g := IngredientGroup{Phase: "dough"}
						sm.IngredientGroups = append(sm.IngredientGroups, g)
						currentGroup = &sm.IngredientGroups[len(sm.IngredientGroups)-1]
					}
				case "instructions":
					flushPending()
					current = secInstructions
					currentGroup = nil
					sm.IngredientGroups = discardEmptyGroups(sm.IngredientGroups)
				}
			}
			continue
		}

		// Section keyword matching — LINE-EXCLUSIVE (keyword must be sole content)
		if dest, ok := sectionKeywords[lowerTrimmed]; ok {
			switch dest {
			case "ingredients":
				current = secIngredients
				if currentGroup == nil {
					g := IngredientGroup{Phase: "dough"}
					sm.IngredientGroups = append(sm.IngredientGroups, g)
					currentGroup = &sm.IngredientGroups[len(sm.IngredientGroups)-1]
				}
			case "instructions":
				flushPending()
				current = secInstructions
				currentGroup = nil
				sm.IngredientGroups = discardEmptyGroups(sm.IngredientGroups)
			}
			continue
		}

		switch current {
		case secDescription:
			if reMetaField.MatchString(trimmed) {
				sm.MetadataLines = append(sm.MetadataLines, trimmed)
				continue
			}
			// Title: first non-empty line that doesn't start with a digit
			if sm.Title == "" && trimmed != "" {
				if reQuantityStart.MatchString(trimmed) {
					sm.TitleDetectionMethod = TitleEmpty
				} else {
					sm.Title = trimmed
					sm.TitleDetectionMethod = TitleHeuristic
					continue
				}
			}
			if trimmed != "" {
				if sm.Description != "" {
					sm.Description += "\n"
				}
				sm.Description += trimmed
			}

		case secIngredients:
			if trimmed == "" {
				continue
			}
			if reMetaField.MatchString(trimmed) {
				sm.MetadataLines = append(sm.MetadataLines, trimmed)
				continue
			}
			// Check for subsection header
			if phase, ok, deferable := matchIngredientSubsection(trimmed); ok {
				flushPending()
				if deferable {
					pendingPhase = phase
					pendingLines = nil
					hasPending = true
					continue
				}
				sm.IngredientGroups = discardEmptyGroups(sm.IngredientGroups)
				g := IngredientGroup{Phase: phase}
				sm.IngredientGroups = append(sm.IngredientGroups, g)
				currentGroup = &sm.IngredientGroups[len(sm.IngredientGroups)-1]
				continue
			}
			if hasPending {
				pendingLines = append(pendingLines, trimmed)
				continue
			}
			if currentGroup == nil {
				g := IngredientGroup{Phase: "dough"}
				sm.IngredientGroups = append(sm.IngredientGroups, g)
				currentGroup = &sm.IngredientGroups[len(sm.IngredientGroups)-1]
			}
			currentGroup.Lines = append(currentGroup.Lines, trimmed)
			// Update pointer (slice may have reallocated)
			currentGroup = &sm.IngredientGroups[len(sm.IngredientGroups)-1]

		case secInstructions:
			if trimmed == "" {
				continue
			}
			if reMetaField.MatchString(trimmed) {
				sm.MetadataLines = append(sm.MetadataLines, trimmed)
				continue
			}
			// Instruction sub-header: short capitalised line, no quantity, no terminal punctuation
			if isInstructionSubHeader(trimmed) {
				sm.InstructionLines = append(sm.InstructionLines, trimmed+":")
				continue
			}
			sm.InstructionLines = append(sm.InstructionLines, trimmed)
		}
	}

	// Final cleanup
	flushPending()
	sm.IngredientGroups = discardEmptyGroups(sm.IngredientGroups)

	// Cap description at 2000 chars at last sentence boundary
	sm.Description = capDescription(sm.Description, 2000)

	return sm
}

func isBakersPctHeader(line string) bool {
	for _, re := range bakersPctPatterns {
		if re.MatchString(line) {
			return true
		}
	}
	return false
}

// matchIngredientSubsection checks whether line is a subsection header. The third return
// value, deferable, is true only for a no-colon generic-fallback match (title case, no
// hand-listed pattern) — these are buffered by the caller rather than trusted immediately,
// because a bare title-case ingredient line looks identical to a real header until we see
// whether real content follows it. Colon-terminated and named/capture-derived matches are
// never deferable — a trailing colon or a recognized pattern is strong enough evidence to
// trust immediately, exactly as before this feature existed.
func matchIngredientSubsection(line string) (phase string, ok bool, deferable bool) {
	for _, p := range ingredientSubsectionPatterns {
		m := p.pattern.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		if p.phase != "" {
			// Named pattern matched (e.g. tangzhong, levain) — return the verbatim source
			// text (trailing dash/colon stripped), not the lowercased constant, so casing
			// like "Tangzhong" or "TANGZHONG" survives to the display layer. Strip an
			// optional trailing "Ingredients" suffix (only the "final dough (ingredients)?"
			// pattern allows this) so the label still equals its tier-1/tier-2 canonical
			// form for classification purposes.
			label := strings.TrimRight(strings.TrimSpace(line), "-–: \t")
			label = reTrailingIngredientsSuffix.ReplaceAllString(label, "")
			return label, true, false
		}
		// Derive phase from capture group, preserving original case.
		if len(m) > 1 {
			phase := strings.TrimSpace(m[1])
			phase = strings.TrimRight(phase, "– \t")
			return phase, true, false
		}
	}

	// Generic fallback: a short (1-4 word) line with no digits, that isn't a known
	// quantity-less ingredient phrasing (noQtyPatterns/isNoQtyLine in ingredients.go, same
	// package). A trailing colon is trusted immediately (strong, unambiguous signal — this
	// was the entire detection mechanism before this feature). Without a colon, the line
	// must also be Title Case, AND is only tentatively accepted (deferable=true) — bare
	// ingredient references are sentence-case ("Pinch of salt") but some are genuinely Title
	// Case too ("Olive Oil"), so the caller must see whether real content follows before
	// trusting it.
	if !strings.ContainsAny(line, "0123456789") && !isNoQtyLine(line) {
		trimmedLine := strings.TrimSpace(line)
		hasColon := strings.HasSuffix(trimmedLine, ":")
		name := strings.TrimRight(trimmedLine, "-–: \t")
		if words := strings.Fields(name); len(words) >= 1 && len(words) <= 4 {
			if hasColon {
				return name, true, false
			}
			if looksLikeTitleCase(name) {
				return name, true, true
			}
		}
	}

	return "", false, false
}

func isInstructionSubHeader(line string) bool {
	words := strings.Fields(line)
	if len(words) == 0 || len(words) > 6 {
		return false
	}
	if reQuantityStart.MatchString(line) {
		return false
	}
	if reInstructionNum.MatchString(line) {
		return false
	}
	// Must not end with terminal punctuation
	lastRune, _ := utf8.DecodeLastRuneInString(line)
	if lastRune == '.' || lastRune == '!' || lastRune == '?' {
		return false
	}
	// Must start with uppercase
	firstRune, _ := utf8.DecodeRuneInString(line)
	if !unicode.IsUpper(firstRune) {
		return false
	}
	return true
}

func discardEmptyGroups(groups []IngredientGroup) []IngredientGroup {
	var result []IngredientGroup
	for _, g := range groups {
		if len(g.Lines) > 0 {
			result = append(result, g)
		}
	}
	return result
}

func capDescription(desc string, limit int) string {
	runes := []rune(desc)
	if len(runes) <= limit {
		return desc
	}
	truncated := string(runes[:limit])
	lastPeriod := strings.LastIndex(truncated, ". ")
	if lastPeriod > 0 && lastPeriod > len(truncated)-200 {
		return truncated[:lastPeriod+1]
	}
	return truncated
}
