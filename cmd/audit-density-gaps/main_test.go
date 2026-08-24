package main

import (
	"testing"

	"github.com/BreadBrand/breadmachine/models"
)

func TestCheckIngredients_FlagsUnknownDensityAndCountWeight(t *testing.T) {
	densityGaps := gapReport{}
	countWeightGaps := gapReport{}

	ings := []models.Ingredient{
		{IngredientName: "bread flour", Unit: "g", Quantity: 500}, // known density
		{IngredientName: "dragonfruit", Unit: "g", Quantity: 100}, // unknown density
		{IngredientName: "egg", Unit: "count", Quantity: 2},       // known count weight
		{IngredientName: "banana", Unit: "count", Quantity: 1},    // unknown count weight
	}

	checkIngredients(ings, "Test Recipe", densityGaps, countWeightGaps)

	if _, ok := densityGaps["dragonfruit"]; !ok {
		t.Error("expected dragonfruit to be flagged as a density gap")
	}
	if _, ok := densityGaps["bread flour"]; ok {
		t.Error("bread flour has a known density, should not be flagged")
	}
	if _, ok := countWeightGaps["banana"]; !ok {
		t.Error("expected banana to be flagged as a count-weight gap")
	}
	if _, ok := countWeightGaps["egg"]; ok {
		t.Error("egg has a known count weight, should not be flagged")
	}
}

func TestCheckIngredients_DeduplicatesRecipeTitlesPerIngredient(t *testing.T) {
	densityGaps := gapReport{}
	countWeightGaps := gapReport{}

	ings := []models.Ingredient{{IngredientName: "dragonfruit", Unit: "g", Quantity: 100}}
	checkIngredients(ings, "Recipe A", densityGaps, countWeightGaps)
	checkIngredients(ings, "Recipe A", densityGaps, countWeightGaps)
	checkIngredients(ings, "Recipe B", densityGaps, countWeightGaps)

	if len(densityGaps["dragonfruit"]) != 2 {
		t.Errorf("expected 2 distinct recipe titles, got %d", len(densityGaps["dragonfruit"]))
	}
}

func TestGapReport_NoGapsFound(t *testing.T) {
	r := gapReport{}
	if len(r) != 0 {
		t.Fatal("expected empty report")
	}
	// print() with no gaps should not panic; behavior is exercised via
	// the exit-code check in checkIngredients-based integration above.
	r.print("test")
}
