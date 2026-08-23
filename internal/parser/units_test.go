package parser

import "testing"

func TestMatchUnit_ReturnsCanonical_Grams(t *testing.T) {
	for _, input := range []string{"gram", "grams"} {
		if got := MatchUnit(input); got != "g" {
			t.Errorf("MatchUnit(%q) = %q, want %q", input, got, "g")
		}
	}
}

func TestMatchUnit_ReturnsCanonical_Teaspoon(t *testing.T) {
	for _, input := range []string{"teaspoon", "teaspoons", "Teaspoon", "TEASPOONS"} {
		if got := MatchUnit(input); got != "tsp" {
			t.Errorf("MatchUnit(%q) = %q, want %q", input, got, "tsp")
		}
	}
}

func TestMatchUnit_ReturnsCanonical_Tablespoon(t *testing.T) {
	for _, input := range []string{"tablespoon", "tablespoons"} {
		if got := MatchUnit(input); got != "tbsp" {
			t.Errorf("MatchUnit(%q) = %q, want %q", input, got, "tbsp")
		}
	}
}

func TestMatchUnit_ReturnsCanonical_Cups(t *testing.T) {
	if got := MatchUnit("cups"); got != "cup" {
		t.Errorf("MatchUnit(%q) = %q, want %q", "cups", got, "cup")
	}
}

func TestMatchUnit_ReturnsCanonical_Milliliter(t *testing.T) {
	for _, input := range []string{"milliliter", "millilitre", "milliliters", "millilitres"} {
		if got := MatchUnit(input); got != "ml" {
			t.Errorf("MatchUnit(%q) = %q, want %q", input, got, "ml")
		}
	}
}

func TestMatchUnit_ReturnsCanonical_Liter(t *testing.T) {
	for _, input := range []string{"liter", "litre", "liters", "litres"} {
		if got := MatchUnit(input); got != "l" {
			t.Errorf("MatchUnit(%q) = %q, want %q", input, got, "l")
		}
	}
}

func TestMatchUnit_ReturnsCanonical_Kilogram(t *testing.T) {
	for _, input := range []string{"kilogram", "kilograms"} {
		if got := MatchUnit(input); got != "kg" {
			t.Errorf("MatchUnit(%q) = %q, want %q", input, got, "kg")
		}
	}
}

func TestMatchUnit_ReturnsCanonical_Ounce(t *testing.T) {
	for _, input := range []string{"ounce", "ounces"} {
		if got := MatchUnit(input); got != "oz" {
			t.Errorf("MatchUnit(%q) = %q, want %q", input, got, "oz")
		}
	}
}

func TestMatchUnit_ReturnsCanonical_Pound(t *testing.T) {
	for _, input := range []string{"pound", "pounds"} {
		if got := MatchUnit(input); got != "lb" {
			t.Errorf("MatchUnit(%q) = %q, want %q", input, got, "lb")
		}
	}
}

func TestMatchUnit_ReturnsCanonical_FluidOunce(t *testing.T) {
	for _, input := range []string{"fluid ounce", "fluid ounces"} {
		if got := MatchUnit(input); got != "fl oz" {
			t.Errorf("MatchUnit(%q) = %q, want %q", input, got, "fl oz")
		}
	}
}

func TestMatchUnit_AlreadyCanonical_Unchanged(t *testing.T) {
	canonical := []string{"g", "kg", "oz", "lb", "fl oz", "ml", "l", "tsp", "tbsp", "cup"}
	for _, u := range canonical {
		if got := MatchUnit(u); got != u {
			t.Errorf("MatchUnit(%q) = %q, want unchanged %q", u, got, u)
		}
	}
}

func TestMatchUnit_Bunch_Recognised(t *testing.T) {
	if got := MatchUnit("bunch"); got != "bunch" {
		t.Errorf("MatchUnit(%q) = %q, want %q", "bunch", got, "bunch")
	}
}

func TestMatchUnit_Unknown_ReturnsEmpty(t *testing.T) {
	if got := MatchUnit("foobar"); got != "" {
		t.Errorf("MatchUnit(%q) = %q, want empty", "foobar", got)
	}
}

func TestCanonicalUnit_LegacyTbs_MapsToTbsp(t *testing.T) {
	if got := CanonicalUnit("tbs"); got != "tbsp" {
		t.Errorf("CanonicalUnit(%q) = %q, want %q", "tbs", got, "tbsp")
	}
}

func TestCanonicalUnit_LegacyTbls_MapsToTbsp(t *testing.T) {
	if got := CanonicalUnit("tbls"); got != "tbsp" {
		t.Errorf("CanonicalUnit(%q) = %q, want %q", "tbls", got, "tbsp")
	}
	if got := CanonicalUnit("Tbls"); got != "tbsp" {
		t.Errorf("CanonicalUnit(%q) = %q, want %q", "Tbls", got, "tbsp")
	}
}

func TestCanonicalUnit_Unknown_PassesThrough(t *testing.T) {
	if got := CanonicalUnit("foobar"); got != "foobar" {
		t.Errorf("CanonicalUnit(%q) = %q, want input unchanged", "foobar", got)
	}
}

func TestCanonicalUnit_MixedCase_NormalisedToLower(t *testing.T) {
	if got := CanonicalUnit("Gram"); got != "g" {
		t.Errorf("CanonicalUnit(%q) = %q, want %q", "Gram", got, "g")
	}
	if got := CanonicalUnit("G"); got != "g" {
		t.Errorf("CanonicalUnit(%q) = %q, want %q", "G", got, "g")
	}
}
