package handlers

import (
	"testing"

	"github.com/BreadBrand/breadmachine/models"
)

// makeIng is a test helper that builds a models.Ingredient with the fields
// relevant to unit conversion already populated (as normalizeIngredients would).
func makeIng(name, unit string, qty, grams float64) models.Ingredient {
	return models.Ingredient{
		IngredientName: name,
		Unit:           unit,
		Quantity:       qty,
		Grams:          grams,
	}
}

func TestNormalizeIngredients_TblsUnit_ConvertsAtTablespoonRate(t *testing.T) {
	// Regression: "Tbls" (the frontend dropdown spelling) used to miss both the
	// CanonicalUnit and unitToGrams lookups, silently falling back to a ×1
	// multiplier instead of the correct ×15 (tbsp) rate.
	ings := []models.Ingredient{
		makeIng("bread flour", "g", 500, 500),
		makeIng("butter", "Tbls", 2, 0),
	}
	_, result := normalizeIngredients(ings)
	butter := result[1]
	if butter.Unit != "tbsp" {
		t.Errorf("butter unit: want tbsp, got %q", butter.Unit)
	}
	if butter.Grams != 30 {
		t.Errorf("butter grams: want 30 (2 tbsp × 15), got %v", butter.Grams)
	}
}

func TestIsGramDominant_AnyMassUnit(t *testing.T) {
	ings := []models.Ingredient{
		makeIng("bread flour", "g", 500, 500),
		makeIng("salt", "tsp", 1.5, 9),
		makeIng("baking powder", "tsp", 1, 4),
		makeIng("milk", "cup", 1, 240),
	}
	if !isGramDominant(ings) {
		t.Error("any mass unit should make a recipe gram-dominant")
	}
}

func TestIsGramDominant_SingleMassOutnumberedByVolume(t *testing.T) {
	// The common bread pattern: big gram measurements + several small tsp/tbsp.
	// Should still be gram-dominant even though volume outnumbers mass.
	ings := []models.Ingredient{
		makeIng("flour", "g", 500, 500),
		makeIng("salt", "tsp", 1.5, 9),
		makeIng("baking powder", "tsp", 1, 4),
		makeIng("sugar", "tbsp", 2, 25),
	}
	if !isGramDominant(ings) {
		t.Error("1 mass unit vs 3 volume units should still be gram-dominant")
	}
}

func TestIsGramDominant_AllVolume(t *testing.T) {
	ings := []models.Ingredient{
		makeIng("flour", "cup", 2, 240),
		makeIng("milk", "cup", 1, 240),
		makeIng("salt", "tsp", 1, 6),
	}
	if isGramDominant(ings) {
		t.Error("all-volume recipe should not be gram-dominant")
	}
}

func TestIsGramDominant_CountIngredientsIgnored(t *testing.T) {
	// "count" unit (eggs, etc.) should not count toward either tally
	ings := []models.Ingredient{
		makeIng("bread flour", "g", 500, 500),
		makeIng("egg", "count", 2, 0),
		makeIng("salt", "tsp", 1, 6),
	}
	if !isGramDominant(ings) {
		t.Error("count-unit ingredients should be ignored; 1 mass vs 1 volume should still be gram-dominant")
	}
}

func TestConvertToGrams_ConvertsVolumeInGramDominantRecipe(t *testing.T) {
	ings := []models.Ingredient{
		makeIng("bread flour", "g", 500, 500),
		makeIng("water", "g", 350, 350),
		makeIng("salt", "tsp", 1.5, 9.2),
		makeIng("baking powder", "tsp", 1, 4.0),
	}
	result := convertToGrams(ings)

	salt := result[2]
	if salt.Unit != "g" {
		t.Errorf("salt unit: want g, got %q", salt.Unit)
	}
	if salt.Quantity != 9.2 {
		t.Errorf("salt quantity: want 9.2, got %v", salt.Quantity)
	}

	bp := result[3]
	if bp.Unit != "g" {
		t.Errorf("baking powder unit: want g, got %q", bp.Unit)
	}
	if bp.Quantity != 4.0 {
		t.Errorf("baking powder quantity: want 4.0, got %v", bp.Quantity)
	}
}

func TestConvertToGrams_LeavesGramsIngredientUntouched(t *testing.T) {
	ings := []models.Ingredient{
		makeIng("bread flour", "g", 500, 500),
		makeIng("water", "g", 350, 350),
		makeIng("salt", "tsp", 1.5, 9.2),
	}
	result := convertToGrams(ings)
	if result[0].Quantity != 500 || result[0].Unit != "g" {
		t.Error("mass-unit ingredient should be unchanged")
	}
}

func TestConvertToGrams_NoConversionWhenNotGramDominant(t *testing.T) {
	ings := []models.Ingredient{
		makeIng("flour", "cup", 2, 240),
		makeIng("milk", "cup", 1, 240),
		makeIng("salt", "tsp", 1, 6),
	}
	result := convertToGrams(ings)
	for _, ing := range result {
		if ing.Unit == "g" {
			t.Errorf("non-gram-dominant recipe should not be converted, but %q was changed to g", ing.IngredientName)
		}
	}
}

func TestConvertToGrams_SkipsIngredientWithUnknownDensity(t *testing.T) {
	ings := []models.Ingredient{
		makeIng("bread flour", "g", 500, 500),
		makeIng("water", "g", 350, 350),
		makeIng("mystery spice", "tsp", 1, 0), // Grams=0 means density unknown
	}
	result := convertToGrams(ings)
	mystery := result[2]
	if mystery.Unit != "tsp" {
		t.Errorf("ingredient with unknown density should keep original unit, got %q", mystery.Unit)
	}
}
