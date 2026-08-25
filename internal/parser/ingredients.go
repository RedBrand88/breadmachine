package parser

import (
	"regexp"
	"strings"
)

var (
	reQuantityAnchor = regexp.MustCompile(`^(\d+\s+\d+/\d+|\d+(?:\.\d*)?(?:/\d+)?)\s*`)
	reLeadingParen   = regexp.MustCompile(`^\([^)]*\)\s*`)
	reBulletPrefix   = regexp.MustCompile(`^[-*•—–]\s+`)
	reCheckboxPrefix = regexp.MustCompile(`^[▢□☐]\s*`)
	reCrossRef       = regexp.MustCompile(`(?i)\s*(from the build above|see note|recipe follows|from above)\s*`)
	reToppingLine    = regexp.MustCompile(`(?i)\btopping\b`)
)

var noQtyPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)^(.+?)\s+to\s+taste$`),
	regexp.MustCompile(`(?i)^(.+?)\s+as\s+needed$`),
	regexp.MustCompile(`(?i)^(.+?)\s+for\s+dusting$`),
	regexp.MustCompile(`(?i)^(.+?)\s+for\s+greasing$`),
	regexp.MustCompile(`(?i)^(.+?)\s+to\s+serve$`),
	regexp.MustCompile(`(?i)^(.+?),?\s+for\s+topping$`),
}

var adjectivePrefixes = []string{"bubbly", "active", "ripe", "fresh", "large", "medium", "small"}

// explicitFlourPhases are phase names known to always contribute to baker's-% flour totals,
// regardless of their ingredient content. Unchanged from the old doughPhases allow-list.
var explicitFlourPhases = map[string]bool{
	"dough":         true,
	"":              true,
	"starter build": true,
	"levain":        true,
	"starter":       true,
	"pre-ferment":   true,
	"final dough":   true,
	"scald":         true,
	"tangzhong":     true,
	"yudane":        true,
}

// reExplicitNonFlourPhase matches phase names known to never contribute to baker's-% flour
// totals as a whole word within the phase name (e.g. "Streusel Topping" matches "topping"),
// not just when the phase name equals one of these words exactly.
var reExplicitNonFlourPhase = regexp.MustCompile(`(?i)\b(topping|filling|sauce|pesto)\b`)

// baseIngredientKeywords mirrors models.baseIngredientKeywords (models/recipe.go). Kept as a
// small local copy since internal/parser deliberately doesn't import models (pure text→DTO layer,
// no persistence-model dependency) — see Global Constraints in the plan this task came from.
var baseIngredientKeywords = []string{"flour", "lentil", "oat", "cauliflower", "chickpea", "tapioca"}

// classifyPhase decides whether a group counts toward baker's-% flour totals. Tier 1/2 are
// exact, pre-existing, hardcoded phase names (case-insensitive). Tier 3 — anything else, e.g. an
// arbitrary header like "Stiff Sweet Starter" or "Roasted Pepper" — is decided by content: does
// any ingredient in the group actually look like flour, not by guessing from the header text.
func classifyPhase(phase string, dtos []IngredientDTO) bool {
	lower := strings.ToLower(phase)
	if explicitFlourPhases[lower] {
		return true
	}
	if reExplicitNonFlourPhase.MatchString(phase) {
		return false
	}
	for _, d := range dtos {
		name := strings.ToLower(d.IngredientName)
		for _, kw := range baseIngredientKeywords {
			if strings.Contains(name, kw) {
				return true
			}
		}
	}
	return false
}

// ParseIngredients parses all IngredientGroups and routes each line to either the dough slice
// or the other slice, based on classifyPhase. Individual lines containing "topping" are routed
// to other regardless of group classification (unchanged special case).
func ParseIngredients(groups []IngredientGroup) (dough []IngredientDTO, other []IngredientDTO) {
	for _, group := range groups {
		dtos := make([]IngredientDTO, len(group.Lines))
		for i, line := range group.Lines {
			dtos[i] = parseIngredientLine(line)
		}
		groupIsFlour := classifyPhase(group.Phase, dtos)
		for i, line := range group.Lines {
			dto := dtos[i]
			phase := group.Phase
			isFlour := groupIsFlour
			if isFlour && reToppingLine.MatchString(line) {
				phase = "topping"
				isFlour = false
			}
			dto.Phase = phase
			if isFlour {
				dough = append(dough, dto)
			} else {
				other = append(other, dto)
			}
		}
	}
	return
}

func parseIngredientLine(raw string) IngredientDTO {
	dto := IngredientDTO{RawLine: raw}
	line := raw

	// 1. Strip bullet and checkbox prefixes
	line = reBulletPrefix.ReplaceAllString(line, "")
	line = reCheckboxPrefix.ReplaceAllString(line, "")
	line = strings.TrimSpace(line)

	// 2. Yeast alternatives — take first option before "or", keep full RawLine
	if idx := strings.Index(strings.ToLower(line), " or "); idx != -1 {
		before := strings.TrimSpace(line[:idx])
		after := strings.TrimSpace(line[idx+4:])
		if reQuantityAnchor.MatchString(before) && reQuantityAnchor.MatchString(after) {
			line = before
		}
	}

	// 3. No-quantity patterns (e.g. "salt to taste")
	for _, re := range noQtyPatterns {
		m := re.FindStringSubmatch(line)
		if m != nil {
			dto.IngredientName = strings.ToLower(strings.TrimSpace(m[1]))
			dto.ParseOK = dto.IngredientName != ""
			return dto
		}
	}

	// 4. Negative quantity → ParseOK=false
	if strings.HasPrefix(line, "-") && len(line) > 1 && line[1] >= '0' && line[1] <= '9' {
		dto.ParseOK = false
		return dto
	}

	// 5. Extract quantity
	if m := reQuantityAnchor.FindStringSubmatch(line); m != nil {
		dto.Quantity = m[1]
		line = strings.TrimSpace(line[len(m[0]):])
	}

	// 6. Extract unit (KnownUnits is ordered multi-word first)
	lower := strings.ToLower(line)
	for _, u := range KnownUnits {
		if lower == u {
			dto.Unit = CanonicalUnit(u)
			line = ""
			break
		}
		if strings.HasPrefix(lower, u+" ") || strings.HasPrefix(lower, u+",") {
			dto.Unit = CanonicalUnit(u)
			line = strings.TrimSpace(line[len(u):])
			break
		}
	}

	// Strip leading comma left by "u," prefix match
	line = strings.TrimLeft(line, ", \t")

	// 6.5. Strip inline alternate measurement expressed as "/ N unit"
	// e.g. "/ 2 tbsp" from "30g / 2 tbsp ghee". Only attempt when a primary
	// unit was already matched, so we don't accidentally consume ingredient names.
	if dto.Unit != "" && strings.HasPrefix(line, "/") {
		tail := strings.TrimSpace(line[1:])
		if m2 := reQuantityAnchor.FindString(tail); m2 != "" {
			rest := strings.TrimSpace(tail[len(m2):])
			lower2 := strings.ToLower(rest)
			// The alternate unit is intentionally discarded — only advance line past it.
			for _, u := range KnownUnits {
				if lower2 == u || strings.HasPrefix(lower2, u+" ") || strings.HasPrefix(lower2, u+",") {
					line = strings.TrimLeft(strings.TrimSpace(rest[len(u):]), ", \t")
					break
				}
			}
		}
	}

	// 7. Strip leading parenthetical (alternate measurement before ingredient name,
	// e.g. "(4 cups)" in "500 g (4 cups) all purpose flour"). Trailing parens like
	// "(100% hydration)" or "(2 large eggs)" are kept in the name.
	line = reLeadingParen.ReplaceAllString(line, "")
	line = strings.TrimSpace(line)

	// 8. Strip adjective prefixes (loop; handle "adj " and "adj," forms)
	for {
		stripped := false
		for _, adj := range adjectivePrefixes {
			lower2 := strings.ToLower(line)
			prefix1 := adj + " "
			prefix2 := adj + ","
			var candidate string
			if strings.HasPrefix(lower2, prefix1) {
				candidate = strings.TrimSpace(line[len(prefix1):])
			} else if strings.HasPrefix(lower2, prefix2) {
				candidate = strings.TrimSpace(line[len(prefix2):])
			} else {
				continue
			}
			// Guard: don't strip if the current line is an unsupported yeast name
			if !IsUnsupportedYeast(line) {
				line = candidate
				stripped = true
			}
			break
		}
		if !stripped {
			break
		}
	}

	// 9. Strip comma-separated notes (rest after first comma has no quantity).
	// Guard: skip when the pre-comma text is a single word ending in a
	// past-participle suffix (-ed, -en) — it's an adjective and the noun
	// likely follows the comma (e.g. "unsalted, frozen butter").
	if idx := strings.Index(line, ","); idx != -1 {
		before := strings.TrimSpace(line[:idx])
		rest := strings.TrimSpace(line[idx+1:])
		beforeWords := strings.Fields(before)
		isAdjectiveOnly := len(beforeWords) == 1 &&
			(strings.HasSuffix(strings.ToLower(beforeWords[0]), "ed") ||
				strings.HasSuffix(strings.ToLower(beforeWords[0]), "en"))
		if !reQuantityAnchor.MatchString(rest) && !isAdjectiveOnly {
			line = before
		}
	}

	// 10. Strip cross-reference phrases
	line = reCrossRef.ReplaceAllString(line, "")
	line = strings.TrimSpace(line)

	// 11. Countable items: quantity present but no unit matched → assign "count"
	if dto.Quantity != "" && dto.Unit == "" {
		dto.Unit = "count"
	}

	dto.IngredientName = line

	// 11. Determine ParseOK: name must be non-empty; zero qty+empty unit only ok
	// for lines matching a no-qty pattern (handled above and returned early).
	dto.ParseOK = dto.IngredientName != "" &&
		!(dto.Quantity == "" && dto.Unit == "" && !isNoQtyLine(raw))

	return dto
}

func isNoQtyLine(raw string) bool {
	for _, re := range noQtyPatterns {
		if re.MatchString(raw) {
			return true
		}
	}
	return false
}
