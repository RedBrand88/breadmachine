package parser

import (
	"strings"
	"testing"
)

func TestDetectSections_KeywordMidSentence_DoesNotTrigger(t *testing.T) {
	// "ingredients" mid-sentence must not trigger section detection
	input := "This recipe uses the best ingredients I have ever tasted in my life"
	sm := DetectSections(input)
	if len(sm.IngredientGroups) > 0 && len(sm.IngredientGroups[0].Lines) > 0 {
		t.Error("mid-sentence 'ingredients' should not trigger section detection")
	}
}

func TestDetectSections_KeywordOnOwnLine_Triggers(t *testing.T) {
	input := "My Great Bread\n\nIngredients\n500g flour\n200g water\n\nDirections\nMix together."
	sm := DetectSections(input)
	if len(sm.IngredientGroups) == 0 {
		t.Fatal("expected ingredient group, got none")
	}
	if len(sm.IngredientGroups[0].Lines) != 2 {
		t.Errorf("expected 2 ingredient lines, got %d", len(sm.IngredientGroups[0].Lines))
	}
	if len(sm.InstructionLines) == 0 {
		t.Error("expected instruction lines")
	}
}

func TestDetectSections_KeywordWithColon_Triggers(t *testing.T) {
	input := "My Bread\n\nIngredients:\n400g flour\n\nInstructions:\nMix well."
	sm := DetectSections(input)
	if len(sm.IngredientGroups) == 0 || len(sm.IngredientGroups[0].Lines) == 0 {
		t.Error("keyword with colon should trigger section detection")
	}
}

func TestDetectSections_TitleFromHeader(t *testing.T) {
	input := "Roasted Garlic Focaccia\n\nIngredients\n500g flour"
	sm := DetectSections(input)
	if sm.Title != "Roasted Garlic Focaccia" {
		t.Errorf("expected title 'Roasted Garlic Focaccia', got %q", sm.Title)
	}
	if sm.TitleDetectionMethod != TitleHeuristic {
		t.Errorf("expected TitleHeuristic, got %v", sm.TitleDetectionMethod)
	}
}

func TestDetectSections_TitleEmpty_WhenFirstLineIsIngredient(t *testing.T) {
	input := "500g flour\n200g water\n\nInstructions\nMix."
	sm := DetectSections(input)
	if sm.TitleDetectionMethod != TitleEmpty {
		t.Errorf("first line starting with digit should not be title; got method %v, title %q",
			sm.TitleDetectionMethod, sm.Title)
	}
}

func TestDetectSections_SubsectionHeaders(t *testing.T) {
	input := "My Focaccia\n\nIngredients\nFor the dough –\n200g flour\n50g water\n\nFor the pesto –\n30g olive oil\n20g parmesan\n\nDirections\nMix."
	sm := DetectSections(input)
	if len(sm.IngredientGroups) < 2 {
		t.Fatalf("expected 2 ingredient groups, got %d", len(sm.IngredientGroups))
	}
	if sm.IngredientGroups[0].Phase != "dough" {
		t.Errorf("expected phase 'dough', got %q", sm.IngredientGroups[0].Phase)
	}
	if sm.IngredientGroups[1].Phase != "pesto" {
		t.Errorf("expected phase 'pesto', got %q", sm.IngredientGroups[1].Phase)
	}
}

func TestDetectSections_EmptyGroupDiscarded(t *testing.T) {
	// Subsection header with no lines before next header → discarded
	input := "Bread\n\nIngredients\nFor the dough –\nFor the topping –\n20g cheese\n\nDirections\nMix."
	sm := DetectSections(input)
	for _, g := range sm.IngredientGroups {
		if len(g.Lines) == 0 {
			t.Errorf("empty group with phase %q should have been discarded", g.Phase)
		}
	}
}

func TestDetectSections_BakersPercentageBlockSkipped(t *testing.T) {
	input := "Bread\n\nIngredients\n500g flour\n300g water\n\nFinal Dough Baker's Percentages\n100% all purpose flour\n65% liquid\n\nInstructions\nMix."
	sm := DetectSections(input)
	for _, g := range sm.IngredientGroups {
		for _, line := range g.Lines {
			if strings.Contains(line, "%") && strings.Contains(strings.ToLower(line), "flour") {
				t.Errorf("baker's percentage line should be skipped: %q", line)
			}
		}
	}
}

func TestDetectSections_NoLineBreaks(t *testing.T) {
	// Fewer than 3 newlines → NoLineBreaks = true
	input := "title\ningredients\n500g flour"
	sm := DetectSections(input)
	if !sm.NoLineBreaks {
		t.Error("expected NoLineBreaks=true for input with < 3 newlines")
	}
}

func TestDetectSections_InstructionSubHeaderPrepended(t *testing.T) {
	input := "Bread\n\nIngredients\n500g flour\n\nInstructions\nMix the Dough\nCombine flour and water in a bowl."
	sm := DetectSections(input)
	if len(sm.InstructionLines) == 0 {
		t.Fatal("expected instruction lines")
	}
	if !strings.HasPrefix(sm.InstructionLines[0], "Mix the Dough:") {
		t.Errorf("sub-header not prepended; got %q", sm.InstructionLines[0])
	}
}

func TestDetectSections_ColonSubsection_Finishes_RoutesToOther(t *testing.T) {
	input := "Naan\n\nIngredients\n270g bread flour\nFinishes:\n30g ghee\n1 garlic clove\n\nInstructions\nMix."
	sm := DetectSections(input)
	if len(sm.IngredientGroups) != 2 {
		t.Fatalf("expected 2 ingredient groups, got %d", len(sm.IngredientGroups))
	}
	if sm.IngredientGroups[0].Phase != "dough" || len(sm.IngredientGroups[0].Lines) != 1 {
		t.Errorf("group 0: got phase=%q lines=%d", sm.IngredientGroups[0].Phase, len(sm.IngredientGroups[0].Lines))
	}
	if sm.IngredientGroups[1].Phase != "Finishes" || len(sm.IngredientGroups[1].Lines) != 2 {
		t.Errorf("group 1: got phase=%q lines=%d", sm.IngredientGroups[1].Phase, len(sm.IngredientGroups[1].Lines))
	}
	_, other := ParseIngredients(sm.IngredientGroups)
	if len(other) != 2 {
		t.Errorf("finishes ingredients should route to other, got %d", len(other))
	}
}

func TestDetectSections_ColonSubsection_MultiWord_RoutesToOther(t *testing.T) {
	input := "Naan\n\nIngredients\n270g bread flour\nCheese Naan:\n100g cheddar\n\nInstructions\nMix."
	sm := DetectSections(input)
	if len(sm.IngredientGroups) != 2 {
		t.Fatalf("expected 2 ingredient groups, got %d", len(sm.IngredientGroups))
	}
	if sm.IngredientGroups[1].Phase != "Cheese Naan" {
		t.Errorf("expected phase 'Cheese Naan', got %q", sm.IngredientGroups[1].Phase)
	}
	_, other := ParseIngredients(sm.IngredientGroups)
	if len(other) != 1 {
		t.Errorf("cheese naan ingredient should route to other, got %d", len(other))
	}
}

func TestDetectSections_ColonSubsection_WithDigit_NotMatched(t *testing.T) {
	// A line with a digit (like a step label) should not be treated as a subsection.
	input := "Bread\n\nIngredients\n500g flour\nStep 1:\n200g water\n\nInstructions\nMix."
	sm := DetectSections(input)
	// "Step 1:" has a digit — should fall through as a plain ingredient line, not create a new group
	if len(sm.IngredientGroups) != 1 {
		t.Errorf("line with digit should not create subsection, got %d groups", len(sm.IngredientGroups))
	}
}

func TestDetectSections_DescriptionCappedAt2000(t *testing.T) {
	longDesc := strings.Repeat("This is a description sentence. ", 80) // >2000 chars
	input := "My Bread\n\n" + longDesc + "\n\nIngredients\n500g flour"
	sm := DetectSections(input)
	if len(sm.Description) > 2000 {
		t.Errorf("description exceeds 2000 chars: %d", len(sm.Description))
	}
}

func TestDetectSections_Scald_NoColon_RecognisedAsSubsection(t *testing.T) {
	input := "Burger Buns\n\nIngredients\n300g bread flour\n150g water\nScald\n60g bread flour\n180ml boiling water\n\nDirections\nMix."
	sm := DetectSections(input)
	if len(sm.IngredientGroups) != 2 {
		t.Fatalf("expected 2 ingredient groups, got %d", len(sm.IngredientGroups))
	}
	found := false
	for _, g := range sm.IngredientGroups {
		if g.Phase == "Scald" {
			found = true
		}
	}
	if !found {
		t.Error("expected a group with phase 'Scald'")
	}
}

func TestDetectSections_Tangzhong_NoColon_RecognisedAsSubsection(t *testing.T) {
	input := "Milk Bread\n\nIngredients\n300g bread flour\nTangzhong\n30g bread flour\n150ml milk\n\nDirections\nMix."
	sm := DetectSections(input)
	found := false
	for _, g := range sm.IngredientGroups {
		if g.Phase == "Tangzhong" {
			found = true
		}
	}
	if !found {
		t.Error("expected a group with phase 'Tangzhong'")
	}
}

func TestDetectSections_Yudane_NoColon_RecognisedAsSubsection(t *testing.T) {
	input := "Japanese Milk Bread\n\nIngredients\n300g bread flour\nYudane\n50g bread flour\n50ml boiling water\n\nDirections\nMix."
	sm := DetectSections(input)
	found := false
	for _, g := range sm.IngredientGroups {
		if g.Phase == "Yudane" {
			found = true
		}
	}
	if !found {
		t.Error("expected a group with phase 'Yudane'")
	}
}

func TestDetectSections_GenericHeader_NoColonRequired(t *testing.T) {
	// "Main Dough" and "Stiff Sweet Starter" have no colon and don't match any of the
	// hand-listed named patterns — they must still be detected as headers.
	input := "Wonder Bread\n\nIngredients\nStiff Sweet Starter\n15 g sourdough starter\n15 g honey\n\nMain Dough\n575 g bread flour\n\nInstructions\nMix."
	sm := DetectSections(input)
	if len(sm.IngredientGroups) != 2 {
		t.Fatalf("expected 2 ingredient groups, got %d", len(sm.IngredientGroups))
	}
	if sm.IngredientGroups[0].Phase != "Stiff Sweet Starter" {
		t.Errorf("expected verbatim phase 'Stiff Sweet Starter', got %q", sm.IngredientGroups[0].Phase)
	}
	if sm.IngredientGroups[1].Phase != "Main Dough" {
		t.Errorf("expected verbatim phase 'Main Dough', got %q", sm.IngredientGroups[1].Phase)
	}
}

func TestDetectSections_NamedPattern_VerbatimCasePreserved(t *testing.T) {
	// Named patterns (e.g. "tangzhong") still recognize case-insensitively, but the returned
	// phase text keeps the original casing from the source instead of a lowercased constant.
	input := "Bread\n\nIngredients\nTangzhong\n35 g flour\n150 ml milk\n\nInstructions\nMix."
	sm := DetectSections(input)
	if len(sm.IngredientGroups) != 1 {
		t.Fatalf("expected 1 ingredient group, got %d", len(sm.IngredientGroups))
	}
	if sm.IngredientGroups[0].Phase != "Tangzhong" {
		t.Errorf("expected verbatim phase 'Tangzhong', got %q", sm.IngredientGroups[0].Phase)
	}
}

func TestDetectSections_BackReferenceLine_NotMisdetectedAsHeader(t *testing.T) {
	// "All of the stiff sweet starter from above" has no quantity either, but at 6 words it's
	// over the header-detection word cap — it must stay an ingredient line, not become a new
	// (spurious) subsection.
	input := "Bread\n\nIngredients\nMain Dough\nAll of the stiff sweet starter from above\n225 g water\n\nInstructions\nMix."
	sm := DetectSections(input)
	if len(sm.IngredientGroups) != 1 {
		t.Fatalf("expected 1 ingredient group (no spurious header split), got %d", len(sm.IngredientGroups))
	}
	if len(sm.IngredientGroups[0].Lines) != 2 {
		t.Errorf("expected 2 lines in the single group, got %d: %v", len(sm.IngredientGroups[0].Lines), sm.IngredientGroups[0].Lines)
	}
}

func TestDetectSections_NoQtyIngredientLine_NotMisdetectedAsHeader(t *testing.T) {
	// "Salt to taste" is short (3 words) and has no digits — exactly the shape a naive
	// word-count-only heuristic would misdetect as a header. It's a real quantity-less
	// ingredient line (noQtyPatterns) and must not split the group.
	input := "Bread\n\nIngredients\n200 g flour\nSalt to taste\n\nInstructions\nMix."
	sm := DetectSections(input)
	if len(sm.IngredientGroups) != 1 {
		t.Fatalf("expected 1 ingredient group, got %d", len(sm.IngredientGroups))
	}
	if len(sm.IngredientGroups[0].Lines) != 2 {
		t.Errorf("expected 2 lines (flour + salt to taste), got %d: %v", len(sm.IngredientGroups[0].Lines), sm.IngredientGroups[0].Lines)
	}
}
