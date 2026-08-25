package models

import (
	"math"
	"testing"
)

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < 0.0001
}

func ing(name string, grams float64, phase Phase) Ingredient {
	return Ingredient{IngredientName: name, Grams: grams, Phase: phase}
}

func TestCalculateBakerPercentages_Basic(t *testing.T) {
	ingredients := []Ingredient{
		ing("flour", 500, PhaseDough),
		ing("water", 350, PhaseDough),
		ing("salt", 10, PhaseDough),
	}

	CalculateBakerPercentages(ingredients)

	if !almostEqual(ingredients[0].BakerPercentage, 100) {
		t.Errorf("flour: expected 100%%, got %v", ingredients[0].BakerPercentage)
	}
	if !almostEqual(ingredients[1].BakerPercentage, 70) {
		t.Errorf("water: expected 70%%, got %v", ingredients[1].BakerPercentage)
	}
	if !almostEqual(ingredients[2].BakerPercentage, 2) {
		t.Errorf("salt: expected 2%%, got %v", ingredients[2].BakerPercentage)
	}
}

func TestCalculateBakerPercentages_MultipleBaseIngredientsCombineIntoBase(t *testing.T) {
	// flour and oat both match base keywords, so the base weight is their sum (500),
	// not just the first match.
	ingredients := []Ingredient{
		ing("flour", 400, PhaseDough),
		ing("rolled oat", 100, PhaseDough),
		ing("water", 300, PhaseDough),
	}

	CalculateBakerPercentages(ingredients)

	if !almostEqual(ingredients[0].BakerPercentage, 80) {
		t.Errorf("flour: expected 80%%, got %v", ingredients[0].BakerPercentage)
	}
	if !almostEqual(ingredients[1].BakerPercentage, 20) {
		t.Errorf("oat: expected 20%%, got %v", ingredients[1].BakerPercentage)
	}
	if !almostEqual(ingredients[2].BakerPercentage, 60) {
		t.Errorf("water: expected 60%%, got %v", ingredients[2].BakerPercentage)
	}
}

func TestCalculateBakerPercentages_NonFlourPhaseExcludedFromBaseAndZeroed(t *testing.T) {
	// A base-keyword ingredient sitting in a phase outside the known flour-phase
	// list (e.g. "filling") must not count toward the base, and must itself get 0%.
	ingredients := []Ingredient{
		ing("flour", 500, PhaseDough),
		ing("filling flour", 200, "filling"),
	}

	CalculateBakerPercentages(ingredients)

	if !almostEqual(ingredients[0].BakerPercentage, 100) {
		t.Errorf("dough flour: expected 100%% (filling flour must not inflate base), got %v", ingredients[0].BakerPercentage)
	}
	if !almostEqual(ingredients[1].BakerPercentage, 0) {
		t.Errorf("filling flour: expected 0%%, got %v", ingredients[1].BakerPercentage)
	}
}

func TestCalculateBakerPercentages_WordBoundaryNonFlourPhase_StreuselTopping_Excluded(t *testing.T) {
	ingredients := []Ingredient{
		ing("flour", 500, PhaseDough),
		ing("streusel flour", 60, "Streusel Topping"),
	}

	CalculateBakerPercentages(ingredients)

	if !almostEqual(ingredients[0].BakerPercentage, 100) {
		t.Errorf("dough flour: expected 100%% (streusel topping must not inflate base), got %v", ingredients[0].BakerPercentage)
	}
	if !almostEqual(ingredients[1].BakerPercentage, 0) {
		t.Errorf("streusel topping flour: expected 0%%, got %v", ingredients[1].BakerPercentage)
	}
}

func TestCalculateBakerPercentages_NoBaseIngredientYieldsAllZero(t *testing.T) {
	ingredients := []Ingredient{
		ing("water", 300, PhaseDough),
		ing("salt", 5, PhaseDough),
	}

	CalculateBakerPercentages(ingredients)

	for i, dough := range ingredients {
		if !almostEqual(dough.BakerPercentage, 0) {
			t.Errorf("ingredient %d: expected 0%% with no base ingredient present, got %v", i, dough.BakerPercentage)
		}
	}
}

func TestCalculateBakerPercentages_EmptyPhaseTreatedAsDough(t *testing.T) {
	ingredients := []Ingredient{
		ing("flour", 500, ""),
		ing("water", 300, ""),
	}

	CalculateBakerPercentages(ingredients)

	if !almostEqual(ingredients[0].BakerPercentage, 100) {
		t.Errorf("flour: expected 100%%, got %v", ingredients[0].BakerPercentage)
	}
	if !almostEqual(ingredients[1].BakerPercentage, 60) {
		t.Errorf("water: expected 60%%, got %v", ingredients[1].BakerPercentage)
	}
}

func TestCalculateBakerPercentages_PhaseMatchIsCaseInsensitive(t *testing.T) {
	ingredients := []Ingredient{
		ing("flour", 500, "Dough"),
		ing("water", 250, "SCALD"),
	}

	CalculateBakerPercentages(ingredients)

	if !almostEqual(ingredients[0].BakerPercentage, 100) {
		t.Errorf("flour: expected 100%%, got %v", ingredients[0].BakerPercentage)
	}
	if !almostEqual(ingredients[1].BakerPercentage, 50) {
		t.Errorf("water: expected 50%%, got %v", ingredients[1].BakerPercentage)
	}
}

func TestCalculateBakerPercentages_UnrecognizedPhase_WithFlour_Counted(t *testing.T) {
	// A phase not in the explicit lists (e.g. "Stiff Sweet Starter") must still count toward
	// the base when it actually contains a flour-like ingredient.
	ingredients := []Ingredient{
		ing("bread flour", 60, "Stiff Sweet Starter"),
		ing("bread flour", 575, PhaseDough),
	}

	CalculateBakerPercentages(ingredients)

	if !almostEqual(ingredients[0].BakerPercentage, (60.0/635.0)*100) {
		t.Errorf("stiff sweet starter flour: expected %.2f%%, got %v", (60.0/635.0)*100, ingredients[0].BakerPercentage)
	}
	if !almostEqual(ingredients[1].BakerPercentage, (575.0/635.0)*100) {
		t.Errorf("main dough flour: expected %.2f%%, got %v", (575.0/635.0)*100, ingredients[1].BakerPercentage)
	}
}

func TestCalculateBakerPercentages_UnrecognizedPhase_NoFlour_Zeroed(t *testing.T) {
	// A phase not in the explicit lists, with no flour-like ingredient, must not count toward
	// the base and must itself get 0% — regardless of what it's named. This is the case that
	// broke the earlier "default everything unrecognized to flour" approach.
	ingredients := []Ingredient{
		ing("flour", 500, PhaseDough),
		ing("bell pepper", 100, "Roasted Pepper"),
	}

	CalculateBakerPercentages(ingredients)

	if !almostEqual(ingredients[0].BakerPercentage, 100) {
		t.Errorf("dough flour: expected 100%% (roasted pepper must not inflate base), got %v", ingredients[0].BakerPercentage)
	}
	if !almostEqual(ingredients[1].BakerPercentage, 0) {
		t.Errorf("bell pepper: expected 0%%, got %v", ingredients[1].BakerPercentage)
	}
}
