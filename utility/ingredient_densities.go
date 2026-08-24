package utility

import (
	"regexp"
	"strings"
)

type densityEntry struct {
	keyword string
	density float64
}

// Order matters — more specific entries must come before shorter keywords
// that would otherwise match first (e.g. "sea salt" before "salt").
var ingredientDensities = []densityEntry{
	// sweeteners
	{"maple syrup", 1.37},
	{"cream of tartar", 0.99},
	{"powdered sugar", 0.56},
	{"brown sugar", 0.93},
	{"molasses", 1.40},
	{"honey", 1.42},
	// dairy
	{"heavy cream", 1.01},
	{"buttermilk", 1.03},
	{"yogurt", 1.04},
	// yeast — specific first, generic last
	{"active dry yeast", 0.43},
	{"instant yeast", 0.43},
	{"rapid rise yeast", 0.43},
	{"dried yeast", 0.43},
	// oils — specific first
	{"olive oil", 0.908},
	{"avocado oil", 0.913},
	{"vegetable oil", 0.92},
	{"coconut oil", 0.92},
	{"canola oil", 0.92},
	{"sunflower oil", 0.92},
	// salt — specific first
	{"sea salt", 1.18},
	// leaveners
	{"baking powder", 0.80},
	{"baking soda", 0.88},
	// dry goods
	{"vital wheat gluten", 0.55},
	{"cocoa powder", 0.48},
	{"rolled oats", 0.36},
	{"cornmeal", 0.60},
	{"sourdough", 1.0},
	// legumes — specific first
	{"red lentils", 0.79},
	{"green lentils", 0.79},
	{"black lentils", 0.79},
	// binding agents
	{"psyllium husk", 0.25},
	// cheese — specific first, generic last
	{"cream cheese", 0.98},
	{"mozzarella", 0.47},
	{"cheddar", 0.47},
	{"parmesan", 0.37},
	{"pecorino", 0.37},
	// sauces
	{"pizza sauce", 1.05},
	{"tomato sauce", 1.05},
	// fruit
	{"blueberries", 0.62},
	// seeds — specific first
	{"sesame seeds", 0.60},
	{"chia seeds", 0.72},
	{"flax seeds", 0.76},
	{"poppy seeds", 0.56},
	// herbs and aromatics — specific first
	{"dried garlic", 0.40},
	{"garlic powder", 0.40},
	{"onion powder", 0.37},
	{"dried dill", 0.35},
	{"dill", 0.09},
	// single-word / catch-all matches — must come after multi-word entries
	{"lentil", 0.79},
	{"cheese", 0.47},
	{"seed", 0.60},
	{"garlic", 0.40},
	{"lard", 0.92},
	{"ghee", 0.90},
	{"flour", 0.53},
	{"oil", 0.92},
	{"yeast", 0.43},
	{"water", 1.0},
	{"milk", 1.03},
	{"butter", 0.91},
	{"sugar", 0.845},
	{"salt", 1.217},
	{"beer", 1.01},
	{"egg", 1.03},
	{"vanilla", 0.879},
	{"starter", 1.0},
	{"cream", 1.01},
	{"oats", 0.36},
}

func LookupDensity(ingredientName string) float64 {
	normalized := strings.ToLower(ingredientName)
	normalized = regexp.MustCompile(`\(.*?\)`).ReplaceAllString(normalized, "")
	normalized = strings.TrimSpace(normalized)

	for _, entry := range ingredientDensities {
		if strings.Contains(normalized, entry.keyword) {
			return entry.density
		}
	}
	return 0
}

type countWeightEntry struct {
	keyword     string
	gramsPerUnit float64
}

// countWeightTable maps ingredient keywords to grams per single unit (count).
var countWeightTable = []countWeightEntry{
	{"egg", 50},
}

// LookupCountWeight returns grams per single unit for count-based ingredients,
// or 0 if the ingredient is not recognized.
func LookupCountWeight(ingredientName string) float64 {
	normalized := strings.ToLower(ingredientName)
	normalized = regexp.MustCompile(`\(.*?\)`).ReplaceAllString(normalized, "")
	normalized = strings.TrimSpace(normalized)

	for _, entry := range countWeightTable {
		if strings.Contains(normalized, entry.keyword) {
			return entry.gramsPerUnit
		}
	}
	return 0
}
