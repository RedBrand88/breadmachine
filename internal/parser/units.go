package parser

import "strings"

// KnownUnits is the canonical list of recognised unit strings.
// Matching is always case-insensitive. Multi-word units (e.g. "fl oz") must be
// checked before single-word units to prevent partial matches.
var KnownUnits = []string{
	// multi-word first
	"fl oz", "fluid ounce", "fluid ounces",
	// weight
	"g", "gram", "grams",
	"kg", "kilogram", "kilograms",
	"oz", "ounce", "ounces",
	"lb", "pound", "pounds",
	// volume
	"ml", "milliliter", "millilitre", "milliliters", "millilitres",
	"l", "liter", "litre", "liters", "litres",
	"tsp", "teaspoon", "teaspoons",
	"tbsp", "tablespoon", "tablespoons", "tbls",
	"cup", "cups",
	// approximate
	"pinch", "handful", "dash", "smidge", "sprig", "clove", "slice", "piece", "bunch",
}

// unitCanonicals maps long-form and plural unit variants to their canonical abbreviation.
var unitCanonicals = map[string]string{
	"fluid ounce":  "fl oz",
	"fluid ounces": "fl oz",
	"gram":         "g",
	"grams":        "g",
	"kilogram":     "kg",
	"kilograms":    "kg",
	"ounce":        "oz",
	"ounces":       "oz",
	"pound":        "lb",
	"pounds":       "lb",
	"milliliter":   "ml",
	"millilitre":   "ml",
	"milliliters":  "ml",
	"millilitres":  "ml",
	"liter":        "l",
	"litre":        "l",
	"liters":       "l",
	"litres":       "l",
	"teaspoon":     "tsp",
	"teaspoons":    "tsp",
	"tablespoon":   "tbsp",
	"tablespoons":  "tbsp",
	"cups":         "cup",
	// legacy variant never in KnownUnits but present in some DB records
	"tbs": "tbsp",
	// frontend dropdown offers this spelling; never in KnownUnits
	"tbls": "tbsp",
}

// CanonicalUnit returns the canonical abbreviation for a matched unit string.
// If no mapping exists the input is returned unchanged.
func CanonicalUnit(u string) string {
	lower := strings.ToLower(u)
	if canon, ok := unitCanonicals[lower]; ok {
		return canon
	}
	return lower
}

// MatchUnit returns the canonical unit string if the token matches a known unit,
// or empty string if no match. Matching is case-insensitive.
func MatchUnit(token string) string {
	lower := strings.ToLower(strings.TrimSpace(token))
	for _, u := range KnownUnits {
		if lower == u {
			return CanonicalUnit(u)
		}
	}
	return ""
}
