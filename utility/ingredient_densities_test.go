package utility

import "testing"

func TestLookupDensity_BakingPowder(t *testing.T) {
	d := LookupDensity("baking powder")
	if d == 0 {
		t.Error("baking powder density should be non-zero")
	}
}

func TestLookupDensity_BakingPowder_OneTeaspoonIsFourGrams(t *testing.T) {
	// 1 tsp = 5 mL; 1 tsp baking powder is conventionally 4 g
	// so density should be 4/5 = 0.80 g/mL
	d := LookupDensity("baking powder")
	tspML := 5.0
	grams := tspML * d
	if grams < 3.8 || grams > 4.2 {
		t.Errorf("1 tsp baking powder should be ~4 g, got %.2f g (density=%.3f)", grams, d)
	}
}

func TestLookupDensity_AllFloursFallThroughToCatchAll(t *testing.T) {
	flours := []string{
		"all purpose flour",
		"bread flour",
		"whole wheat flour",
		"white flour",
		"rye flour",
		"spelt flour",
		"oat flour",
		"ap flour",
		"strong flour",
		"barley flour",
		"semolina flour",
		"unknown heritage grain flour",
	}
	for _, name := range flours {
		d := LookupDensity(name)
		if d == 0 {
			t.Errorf("LookupDensity(%q) = 0, want flour catch-all density", name)
		}
	}
}

func TestLookupDensity_Blueberries_OneCupIsAboutOneFiftyGrams(t *testing.T) {
	// Regression: blueberries had no density entry at all, so a recipe
	// storing them in grams (e.g. "270 g: blueberries") couldn't be
	// converted to cups for display — see formatIngredientDisplay's
	// muffin-recipe repro on the frontend.
	// 1 cup = 240 mL; 1 cup fresh blueberries is conventionally ~148 g,
	// so density should be ~148/240 = 0.62 g/mL.
	d := LookupDensity("blueberries")
	if d == 0 {
		t.Fatal("blueberries density should be non-zero")
	}
	cupML := 240.0
	grams := cupML * d
	if grams < 135 || grams > 165 {
		t.Errorf("1 cup blueberries should be ~148 g, got %.1f g (density=%.3f)", grams, d)
	}
}

func TestLookupDensity_NewIngredients(t *testing.T) {
	cases := []struct {
		name    string
		minGram float64 // minimum expected grams per tsp (5 mL)
		maxGram float64 // maximum expected grams per tsp
	}{
		// liquid sweeteners — dense, commonly measured in tbsp/cups
		{"honey", 6.5, 7.5},          // ~7.1 g/tsp
		{"maple syrup", 6.0, 7.5},    // ~6.8 g/tsp
		{"molasses", 6.5, 7.5},       // ~7.0 g/tsp
		// dairy
		{"buttermilk", 4.8, 5.5},     // ~5.1 g/tsp
		{"heavy cream", 4.8, 5.3},    // ~5.0 g/tsp
		{"yogurt", 4.8, 5.5},         // ~5.2 g/tsp
		// dry goods
		{"cocoa powder", 2.0, 3.0},   // ~2.4 g/tsp
		{"rolled oats", 1.5, 2.2},    // ~1.8 g/tsp
		{"vital wheat gluten", 2.5, 3.2}, // ~2.8 g/tsp
		// fats
		{"lard", 4.3, 4.8},           // ~4.6 g/tsp
		// leavener helper
		{"cream of tartar", 4.5, 5.5}, // ~5.0 g/tsp
		// grain
		{"cornmeal", 2.5, 3.5}, // ~3.0 g/tsp
		// hard cheese
		{"pecorino", 1.6, 2.0}, // ~1.85 g/tsp, matches parmesan
		// sauces
		{"pizza sauce", 4.8, 5.6},  // ~5.25 g/tsp
		{"tomato sauce", 4.8, 5.6}, // ~5.25 g/tsp
	}
	for _, c := range cases {
		d := LookupDensity(c.name)
		grams := 5.0 * d // 1 tsp = 5 mL
		if grams < c.minGram || grams > c.maxGram {
			t.Errorf("LookupDensity(%q): 1 tsp = %.2f g, want %.1f–%.1f g (density=%.3f)",
				c.name, grams, c.minGram, c.maxGram, d)
		}
	}
}
