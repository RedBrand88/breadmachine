package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func loadFixture(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("could not load fixture %s: %v", name, err)
	}
	return string(data)
}

func TestParse_ChainBakerFocaccia(t *testing.T) {
	dto, err := Parse(loadFixture(t, "chainbaker_focaccia.txt"))
	if err != nil {
		t.Fatal(err)
	}

	if len(dto.DoughIngredients) != 6 {
		t.Errorf("doughIngredients: expected 6, got %d", len(dto.DoughIngredients))
	}

	if len(dto.OtherIngredients) == 0 {
		t.Error("expected otherIngredients to be populated")
	}

	var yeastFound bool
	for _, ing := range dto.DoughIngredients {
		if ing.IngredientName == "instant dry yeast" {
			yeastFound = true
			if ing.Quantity != "2.5" {
				t.Errorf("yeast quantity: expected '2.5', got %q", ing.Quantity)
			}
		}
	}
	if !yeastFound {
		t.Error("expected 'instant dry yeast' in doughIngredients")
	}

	for _, step := range dto.Instructions {
		if strings.Contains(step, "](") {
			t.Errorf("markdown link not stripped from instruction: %q", step)
		}
	}

	if len(dto.Instructions) < 10 {
		t.Errorf("expected >=10 instructions, got %d", len(dto.Instructions))
	}

	for _, step := range dto.Instructions {
		if strings.Contains(step, "Keep in mind") {
			t.Errorf("disclaimer should be excluded: %q", step)
		}
	}

	phases := map[string]bool{}
	for _, ing := range dto.OtherIngredients {
		phases[ing.Phase] = true
	}
	if !phases["roasted pepper"] && !phases["pesto"] {
		t.Errorf("expected phase labels on otherIngredients, got phases: %v", phases)
	}
}

func TestParse_AllRecipesFocaccia(t *testing.T) {
	dto, err := Parse(loadFixture(t, "allrecipes_focaccia.txt"))
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(dto.Title, "Terri") || strings.Contains(dto.Description, "Submitted by") {
		t.Error("attribution lines should be stripped")
	}

	if dto.PrepTime != 20 {
		t.Errorf("prepTime: expected 20, got %d", dto.PrepTime)
	}
	if dto.CookTime != 15 {
		t.Errorf("cookTime: expected 15, got %d", dto.CookTime)
	}
	if dto.AdditionalTime != 25 {
		t.Errorf("additionalTime: expected 25, got %d", dto.AdditionalTime)
	}
	if dto.Servings != "12" {
		t.Errorf("servings: expected '12', got %q", dto.Servings)
	}

	if len(dto.DoughIngredients) != 14 {
		t.Errorf("doughIngredients: expected 14, got %d", len(dto.DoughIngredients))
	}

	for _, ing := range dto.DoughIngredients {
		if strings.Contains(ing.IngredientName, "Calories") || strings.Contains(ing.IngredientName, "Carbohydrate") {
			t.Errorf("nutrition content should be stripped: %q", ing.IngredientName)
		}
	}

	if len(dto.Instructions) != 7 {
		t.Errorf("instructions: expected 7, got %d", len(dto.Instructions))
	}

	if dto.Confidence.Ingredients != 0.4 {
		t.Errorf("confidence.ingredients: expected 0.4, got %v", dto.Confidence.Ingredients)
	}
}

func TestParse_CleverCarrotSourdough(t *testing.T) {
	dto, err := Parse(loadFixture(t, "clevercarrot_sourdough_sandwich.txt"))
	if err != nil {
		t.Fatal(err)
	}

	if dto.PrepTime == 0 {
		t.Error("prepTime should not be zero")
	}
	if dto.CookTime == 0 {
		t.Error("cookTime should not be zero")
	}
	if dto.Servings == "" {
		t.Error("servings should not be empty")
	}

	if strings.Contains(dto.Servings, "1x") {
		t.Errorf("scaling artifact '1x' should be stripped: got %q", dto.Servings)
	}

	var starterFound bool
	for _, ing := range dto.DoughIngredients {
		if strings.HasPrefix(ing.IngredientName, "sourdough starter") {
			starterFound = true
			if ing.Quantity != "50" {
				t.Errorf("starter quantity: expected '50', got %q", ing.Quantity)
			}
			if ing.Unit != "g" {
				t.Errorf("starter unit: expected 'g', got %q", ing.Unit)
			}
		}
	}
	if !starterFound {
		t.Error("expected 'sourdough starter' in doughIngredients")
	}

	var flourFound bool
	for _, ing := range dto.DoughIngredients {
		if strings.Contains(ing.IngredientName, "flour") {
			flourFound = true
			if ing.Quantity != "500" {
				t.Errorf("flour quantity: expected '500', got %q", ing.Quantity)
			}
		}
	}
	if !flourFound {
		t.Error("flour not found in doughIngredients")
	}

	for _, step := range dto.Instructions {
		if strings.Contains(step, "theclevercarrot.com") {
			t.Errorf("source URL should be stripped: %q", step)
		}
	}
}

func TestParse_AboutBlankSourdough(t *testing.T) {
	dto, err := Parse(loadFixture(t, "aboutblank_sourdough_sandwich.txt"))
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(dto.Title, "about:blank") || strings.Contains(dto.Description, "about:blank") {
		t.Error("about:blank artifact should be stripped")
	}

	for _, ing := range dto.DoughIngredients {
		if strings.Contains(ing.IngredientName, "from the build above") {
			t.Errorf("cross-reference not stripped: %q", ing.IngredientName)
		}
	}

	for _, ing := range dto.DoughIngredients {
		if strings.Contains(ing.IngredientName, "% all purpose") || strings.Contains(ing.IngredientName, "% liquid") {
			t.Errorf("baker's percentage line leaked into ingredients: %q", ing.IngredientName)
		}
	}

	if len(dto.Instructions) < 8 {
		t.Errorf("expected >=8 instructions, got %d", len(dto.Instructions))
	}
}

func TestParse_JustAPinchVolumeOnly(t *testing.T) {
	dto, err := Parse(loadFixture(t, "justapinch_volume_only.txt"))
	if err != nil {
		t.Fatal(err)
	}

	var waterFound bool
	for _, ing := range dto.DoughIngredients {
		if ing.IngredientName == "water" {
			waterFound = true
			if ing.Quantity != "1" {
				t.Errorf("water: expected qty='1', got %q", ing.Quantity)
			}
			if ing.Unit != "cup" {
				t.Errorf("water: expected unit='cup', got %q", ing.Unit)
			}
		}
	}
	if !waterFound {
		t.Error("water not found in doughIngredients")
	}

	for _, ing := range dto.DoughIngredients {
		if strings.Contains(ing.IngredientName, "discard") {
			if IsUnsupportedYeast(ing.IngredientName) {
				t.Errorf("sourdough discard should not be unsupported yeast: %q", ing.IngredientName)
			}
		}
	}

	if dto.Confidence.Ingredients != 0.4 {
		t.Errorf("confidence.ingredients: expected 0.4 (unsupported yeast), got %v",
			dto.Confidence.Ingredients)
	}

	if len(dto.OtherIngredients) != 0 {
		t.Errorf("expected 0 otherIngredients, got %d", len(dto.OtherIngredients))
	}
}
