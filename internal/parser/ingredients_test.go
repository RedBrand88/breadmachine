package parser

import (
	"strings"
	"testing"
)

func doughLines(lines ...string) []IngredientGroup {
	return []IngredientGroup{{Phase: "dough", Lines: lines}}
}

func TestParseIngredients_Integer(t *testing.T) {
	dough, _ := ParseIngredients(doughLines("200 g flour"))
	if dough[0].Quantity != "200" || dough[0].Unit != "g" || dough[0].IngredientName != "flour" {
		t.Errorf("got qty=%v unit=%q name=%q", dough[0].Quantity, dough[0].Unit, dough[0].IngredientName)
	}
}

func TestParseIngredients_TblsUnit_RecognizedAsTbsp(t *testing.T) {
	// Regression: "tbls" (the same spelling offered by the frontend's unit
	// dropdown) was missing from KnownUnits, so pasted recipe text using it
	// left the unit unmatched entirely.
	dough, _ := ParseIngredients(doughLines("2 Tbls butter"))
	if dough[0].Unit != "tbsp" {
		t.Errorf("expected unit tbsp, got %q", dough[0].Unit)
	}
}

func TestParseIngredients_Decimal(t *testing.T) {
	dough, _ := ParseIngredients(doughLines("2.5 tsp salt"))
	if dough[0].Quantity != "2.5" {
		t.Errorf("expected '2.5', got %q", dough[0].Quantity)
	}
}

func TestParseIngredients_DecimalZeroPointFive(t *testing.T) {
	dough, _ := ParseIngredients(doughLines("0.5 tsp salt"))
	if dough[0].Quantity != "0.5" {
		t.Errorf("expected '0.5', got %q", dough[0].Quantity)
	}
}

func TestParseIngredients_MetricFirst_DualMeasurement(t *testing.T) {
	dough, _ := ParseIngredients(doughLines("500 g (4 cups) all purpose flour"))
	if dough[0].Quantity != "500" || dough[0].Unit != "g" {
		t.Errorf("got qty=%v unit=%q", dough[0].Quantity, dough[0].Unit)
	}
	if dough[0].IngredientName != "all purpose flour" {
		t.Errorf("got name=%q", dough[0].IngredientName)
	}
}

func TestParseIngredients_VolumeFirst_DualMeasurement(t *testing.T) {
	dough, _ := ParseIngredients(doughLines("1 cup (240 grams) water"))
	if dough[0].Quantity != "1" || dough[0].Unit != "cup" {
		t.Errorf("got qty=%v unit=%q", dough[0].Quantity, dough[0].Unit)
	}
	if dough[0].IngredientName != "water" {
		t.Errorf("got name=%q", dough[0].IngredientName)
	}
}

func TestParseIngredients_WholeItem(t *testing.T) {
	dough, _ := ParseIngredients(doughLines("1 bell pepper, cut into chunks"))
	if dough[0].Unit != "count" {
		t.Errorf("whole item should have count unit, got %q", dough[0].Unit)
	}
	if dough[0].IngredientName != "bell pepper" {
		t.Errorf("got name=%q", dough[0].IngredientName)
	}
}

func TestParseIngredients_NoQuantityPattern_ToTaste(t *testing.T) {
	dough, _ := ParseIngredients(doughLines("Salt to taste"))
	if dough[0].Quantity != "" {
		t.Errorf("expected quantity='', got %q", dough[0].Quantity)
	}
	if dough[0].IngredientName != "salt" {
		t.Errorf("got name=%q", dough[0].IngredientName)
	}
	if !dough[0].ParseOK {
		t.Error("to-taste pattern should set ParseOK=true")
	}
}

func TestParseIngredients_TrailingParenthetical_PreservedInName(t *testing.T) {
	// Trailing parens like "(100% hydration)" are kept — they are useful context.
	dough, _ := ParseIngredients(doughLines("50 g bubbly, active sourdough starter (100% hydration)"))
	if dough[0].IngredientName != "sourdough starter (100% hydration)" {
		t.Errorf("got name=%q", dough[0].IngredientName)
	}
	if !dough[0].ParseOK {
		t.Error("expected ParseOK=true")
	}
}

func TestParseIngredients_LeadingParenthetical_Stripped(t *testing.T) {
	// Leading parens (alternate measurement before ingredient name) are stripped.
	dough, _ := ParseIngredients(doughLines("500 g (4 cups) all purpose flour"))
	if dough[0].IngredientName != "all purpose flour" {
		t.Errorf("got name=%q", dough[0].IngredientName)
	}
}

func TestParseIngredients_CountParenthetical_PreservedInName(t *testing.T) {
	// Count info attached to ingredient name is preserved.
	dough, _ := ParseIngredients(doughLines("105g eggs(2 large eggs)"))
	if dough[0].IngredientName != "eggs(2 large eggs)" {
		t.Errorf("got name=%q", dough[0].IngredientName)
	}
	if dough[0].Quantity != "105" || dough[0].Unit != "g" {
		t.Errorf("got qty=%q unit=%q", dough[0].Quantity, dough[0].Unit)
	}
}

func TestParseIngredients_CommaNoteStripped(t *testing.T) {
	dough, _ := ParseIngredients(doughLines("1 cup water, warmed to 100-110 degrees F"))
	if dough[0].IngredientName != "water" {
		t.Errorf("got name=%q", dough[0].IngredientName)
	}
	if !strings.Contains(dough[0].RawLine, "warmed") {
		t.Error("comma note should be preserved in RawLine")
	}
}

func TestParseIngredients_AdjectiveBeforeComma_NounPreserved(t *testing.T) {
	// "unsalted, frozen butter" — the noun ("butter") comes after the comma.
	// Stripping at the comma would leave only the adjective "unsalted", which
	// is not a meaningful ingredient name. The full name must be preserved.
	cases := []struct {
		line string
		name string
	}{
		{"276 grams (8 tablespoons) unsalted, frozen butter", "unsalted, frozen butter"},
		{"113 g salted, softened butter", "salted, softened butter"},
	}
	for _, c := range cases {
		dough, _ := ParseIngredients(doughLines(c.line))
		if dough[0].IngredientName != c.name {
			t.Errorf("line=%q: got name=%q, want %q", c.line, dough[0].IngredientName, c.name)
		}
	}
}

func TestParseIngredients_CrossReferenceStripped(t *testing.T) {
	dough, _ := ParseIngredients(doughLines("100 grams ripe sourdough starter from the build above"))
	if dough[0].IngredientName != "sourdough starter" {
		t.Errorf("got name=%q", dough[0].IngredientName)
	}
}

func TestParseIngredients_YeastAlternatives_TakeFirst(t *testing.T) {
	dough, _ := ParseIngredients(doughLines("2.5g instant dry yeast or 3g active dry yeast or 7.5g fresh yeast"))
	if dough[0].Quantity != "2.5" {
		t.Errorf("expected '2.5', got %q", dough[0].Quantity)
	}
	if dough[0].IngredientName != "instant dry yeast" {
		t.Errorf("got name=%q", dough[0].IngredientName)
	}
	if !strings.Contains(dough[0].RawLine, "active dry yeast") {
		t.Error("alternatives should be preserved in RawLine")
	}
}

func TestParseIngredients_UnsupportedYeast_ParseOKTrue_ConfidenceLow(t *testing.T) {
	dough, _ := ParseIngredients(doughLines("1 tablespoon active dry yeast"))
	if !dough[0].ParseOK {
		t.Error("unsupported yeast should still have ParseOK=true (the line parsed fine)")
	}
	if dough[0].IngredientName != "active dry yeast" {
		t.Errorf("got name=%q", dough[0].IngredientName)
	}
}

func TestParseIngredients_SourdoughDiscard_Supported(t *testing.T) {
	dough, _ := ParseIngredients(doughLines("0.5 cup sourdough discard"))
	if IsUnsupportedYeast(dough[0].IngredientName) {
		t.Error("sourdough discard should not be classified as unsupported yeast")
	}
}

func TestParseIngredients_NegativeQuantity_ParseOKFalse(t *testing.T) {
	dough, _ := ParseIngredients(doughLines("-2 cups flour"))
	if dough[0].ParseOK {
		t.Error("negative quantity should set ParseOK=false")
	}
}

func TestParseIngredients_Type00Flour_ParsedCorrectly(t *testing.T) {
	dough, _ := ParseIngredients(doughLines("500 g type 00 flour"))
	if dough[0].IngredientName != "type 00 flour" {
		t.Errorf("got name=%q", dough[0].IngredientName)
	}
	if dough[0].Quantity != "500" {
		t.Errorf("got qty=%q", dough[0].Quantity)
	}
}

func TestParseIngredients_ArrayRouting_StarterBuild_ToDough(t *testing.T) {
	groups := []IngredientGroup{
		{Phase: "starter build", Lines: []string{"30 g sourdough starter", "35 g flour"}},
		{Phase: "pesto", Lines: []string{"30 g olive oil"}},
	}
	dough, other := ParseIngredients(groups)
	if len(dough) != 2 {
		t.Errorf("starter build should route to doughIngredients, got %d", len(dough))
	}
	if len(other) != 1 {
		t.Errorf("pesto should route to otherIngredients, got %d", len(other))
	}
	if dough[0].Phase != "starter build" || dough[1].Phase != "starter build" {
		t.Errorf("expected phase 'starter build' preserved, got %q, %q", dough[0].Phase, dough[1].Phase)
	}
	if other[0].Phase != "pesto" {
		t.Errorf("expected phase 'pesto', got %q", other[0].Phase)
	}
}

func TestParseIngredients_ForTopping_RoutesToOther(t *testing.T) {
	groups := []IngredientGroup{
		{Phase: "", Lines: []string{"200 g flour", "turbinado sugar for topping"}},
	}
	dough, other := ParseIngredients(groups)
	if len(dough) != 1 {
		t.Errorf("expected 1 dough ingredient, got %d", len(dough))
	}
	if len(other) != 1 {
		t.Errorf("expected 1 other ingredient, got %d", len(other))
	}
	if other[0].Phase != "topping" {
		t.Errorf("expected phase 'topping', got %q", other[0].Phase)
	}
}

func TestParseIngredients_ForTopping_StripsPhrasFromName(t *testing.T) {
	groups := []IngredientGroup{
		{Phase: "", Lines: []string{"turbinado sugar for topping"}},
	}
	_, other := ParseIngredients(groups)
	if other[0].IngredientName != "turbinado sugar" {
		t.Errorf("expected name 'turbinado sugar', got %q", other[0].IngredientName)
	}
	if !other[0].ParseOK {
		t.Error("expected ParseOK=true")
	}
}

func TestParseIngredients_ForTopping_WithComma_RoutesToOther(t *testing.T) {
	groups := []IngredientGroup{
		{Phase: "", Lines: []string{"sesame seeds, for topping"}},
	}
	_, other := ParseIngredients(groups)
	if len(other) != 1 {
		t.Errorf("expected 1 other ingredient, got %d", len(other))
	}
	if other[0].IngredientName != "sesame seeds" {
		t.Errorf("expected name 'sesame seeds', got %q", other[0].IngredientName)
	}
}

func TestParseIngredients_ToppingPhaseSection_RoutesToOther(t *testing.T) {
	// A named "topping" section from DetectSections should still route correctly.
	groups := []IngredientGroup{
		{Phase: "topping", Lines: []string{"2 tbsp demerara sugar"}},
	}
	dough, other := ParseIngredients(groups)
	if len(dough) != 0 {
		t.Errorf("topping phase should not route to dough, got %d dough", len(dough))
	}
	if len(other) != 1 || other[0].Phase != "topping" {
		t.Errorf("expected 1 other with phase 'topping', got %+v", other)
	}
}

func TestParseIngredients_BulletStripped(t *testing.T) {
	for _, bullet := range []string{"- 200g flour", "* 200g flour", "• 200g flour", "— 200g flour"} {
		dough, _ := ParseIngredients(doughLines(bullet))
		if dough[0].IngredientName != "flour" {
			t.Errorf("bullet not stripped for %q: got name %q", bullet, dough[0].IngredientName)
		}
	}
}

func TestParseIngredients_ParseOKFalse_EmptyName(t *testing.T) {
	dough, _ := ParseIngredients(doughLines("100 g"))
	if dough[0].ParseOK {
		t.Error("empty ingredient name should set ParseOK=false")
	}
}

func TestParseIngredients_Eggs_LargeStripped_CountUnit(t *testing.T) {
	dough, _ := ParseIngredients(doughLines("2 large eggs"))
	if dough[0].Quantity != "2" || dough[0].Unit != "count" || dough[0].IngredientName != "eggs" {
		t.Errorf("got qty=%q unit=%q name=%q", dough[0].Quantity, dough[0].Unit, dough[0].IngredientName)
	}
	if !dough[0].ParseOK {
		t.Error("expected ParseOK=true")
	}
}

func TestParseIngredients_SizeQualifier_Medium_Stripped(t *testing.T) {
	dough, _ := ParseIngredients(doughLines("1 medium onion"))
	if dough[0].IngredientName != "onion" {
		t.Errorf("expected 'onion', got %q", dough[0].IngredientName)
	}
	if dough[0].Unit != "count" {
		t.Errorf("expected unit 'count', got %q", dough[0].Unit)
	}
}

func TestParseIngredients_SizeQualifier_Small_Stripped(t *testing.T) {
	dough, _ := ParseIngredients(doughLines("2 small lemons"))
	if dough[0].IngredientName != "lemons" {
		t.Errorf("expected 'lemons', got %q", dough[0].IngredientName)
	}
	if dough[0].Unit != "count" {
		t.Errorf("expected unit 'count', got %q", dough[0].Unit)
	}
}

func TestParseIngredients_CountUnit_NotAppliedWhenUnitPresent(t *testing.T) {
	dough, _ := ParseIngredients(doughLines("2 tsp salt"))
	if dough[0].Unit != "tsp" {
		t.Errorf("known unit should not be replaced by count, got %q", dough[0].Unit)
	}
}

func TestParseIngredients_CountUnit_NotAppliedWithNoQuantity(t *testing.T) {
	dough, _ := ParseIngredients(doughLines("Salt to taste"))
	if dough[0].Unit != "" {
		t.Errorf("no-qty pattern should not get count unit, got %q", dough[0].Unit)
	}
}

func TestParseIngredients_StandaloneFraction_Half(t *testing.T) {
	dough, _ := ParseIngredients(doughLines("1/2 tsp baking soda"))
	if dough[0].Quantity != "1/2" || dough[0].Unit != "tsp" || dough[0].IngredientName != "baking soda" {
		t.Errorf("got qty=%q unit=%q name=%q", dough[0].Quantity, dough[0].Unit, dough[0].IngredientName)
	}
	if !dough[0].ParseOK {
		t.Error("expected ParseOK=true")
	}
}

func TestParseIngredients_StandaloneFraction_Quarter(t *testing.T) {
	dough, _ := ParseIngredients(doughLines("1/4 tsp salt"))
	if dough[0].Quantity != "1/4" || dough[0].Unit != "tsp" || dough[0].IngredientName != "salt" {
		t.Errorf("got qty=%q unit=%q name=%q", dough[0].Quantity, dough[0].Unit, dough[0].IngredientName)
	}
}

func TestParseIngredients_StandaloneFraction_ThreeQuarters(t *testing.T) {
	dough, _ := ParseIngredients(doughLines("3/4 cup flour"))
	if dough[0].Quantity != "3/4" || dough[0].Unit != "cup" || dough[0].IngredientName != "flour" {
		t.Errorf("got qty=%q unit=%q name=%q", dough[0].Quantity, dough[0].Unit, dough[0].IngredientName)
	}
}

func TestParseIngredients_StandaloneFraction_TwoThirds(t *testing.T) {
	dough, _ := ParseIngredients(doughLines("2/3 cup water"))
	if dough[0].Quantity != "2/3" || dough[0].Unit != "cup" || dough[0].IngredientName != "water" {
		t.Errorf("got qty=%q unit=%q name=%q", dough[0].Quantity, dough[0].Unit, dough[0].IngredientName)
	}
}

func TestParseIngredients_UnicodeHalfFraction_AfterNormalise(t *testing.T) {
	// ½ is converted to "1/2" by Normalise; ingredient parser then sees "1/2 teaspoon dried basil"
	dto := parseIngredientLine("1/2 teaspoon dried basil")
	if dto.Quantity != "1/2" || dto.Unit != "tsp" || dto.IngredientName != "dried basil" {
		t.Errorf("got qty=%q unit=%q name=%q", dto.Quantity, dto.Unit, dto.IngredientName)
	}
}

func TestParseIngredients_DualMeasurement_InlineSlash_AltUnitStripped(t *testing.T) {
	dough, _ := ParseIngredients(doughLines("30g / 2 tbsp ghee or unsalted butter , melted"))
	if dough[0].Quantity != "30" || dough[0].Unit != "g" {
		t.Errorf("got qty=%q unit=%q", dough[0].Quantity, dough[0].Unit)
	}
	if dough[0].IngredientName != "ghee or unsalted butter" {
		t.Errorf("got name=%q", dough[0].IngredientName)
	}
	if !dough[0].ParseOK {
		t.Error("expected ParseOK=true")
	}
}

func TestParseIngredients_MixedNumber_OneAndHalf(t *testing.T) {
	dough, _ := ParseIngredients(doughLines("1 1/2 tbsp whisked egg , at room temp (around 1/2 an egg)"))
	if dough[0].Quantity != "1 1/2" || dough[0].Unit != "tbsp" || dough[0].IngredientName != "whisked egg" {
		t.Errorf("got qty=%q unit=%q name=%q", dough[0].Quantity, dough[0].Unit, dough[0].IngredientName)
	}
	if !dough[0].ParseOK {
		t.Error("expected ParseOK=true")
	}
}

func TestParseIngredients_MixedNumber_OneAndThreeQuarters(t *testing.T) {
	dough, _ := ParseIngredients(doughLines("1 3/4 cups bread flour , or all-purpose/plain"))
	if dough[0].Quantity != "1 3/4" || dough[0].Unit != "cup" || dough[0].IngredientName != "bread flour" {
		t.Errorf("got qty=%q unit=%q name=%q", dough[0].Quantity, dough[0].Unit, dough[0].IngredientName)
	}
	if !dough[0].ParseOK {
		t.Error("expected ParseOK=true")
	}
}

func TestParseIngredients_Bunch_RecognisedAsUnit(t *testing.T) {
	dough, _ := ParseIngredients(doughLines("1 bunch cilantro"))
	if dough[0].Unit != "bunch" {
		t.Errorf("expected unit 'bunch', got %q", dough[0].Unit)
	}
	if dough[0].IngredientName != "cilantro" {
		t.Errorf("expected name 'cilantro', got %q", dough[0].IngredientName)
	}
	if dough[0].Quantity != "1" {
		t.Errorf("expected quantity '1', got %q", dough[0].Quantity)
	}
	if !dough[0].ParseOK {
		t.Error("expected ParseOK=true")
	}
}

func TestParseIngredients_Scald_RoutesToDough(t *testing.T) {
	groups := []IngredientGroup{
		{Phase: "dough", Lines: []string{"300 g bread flour", "150 g water"}},
		{Phase: "scald", Lines: []string{"60 g bread flour", "180 ml boiling water"}},
	}
	dough, other := ParseIngredients(groups)
	if len(dough) != 4 {
		t.Errorf("scald ingredients should route to doughIngredients, got dough=%d other=%d", len(dough), len(other))
	}
	if len(other) != 0 {
		t.Errorf("expected 0 other ingredients, got %d", len(other))
	}
	if dough[0].Phase != "dough" || dough[1].Phase != "dough" {
		t.Errorf("expected phase 'dough' preserved on first group, got %q, %q", dough[0].Phase, dough[1].Phase)
	}
	if dough[2].Phase != "scald" || dough[3].Phase != "scald" {
		t.Errorf("expected phase 'scald' preserved on second group, got %q, %q", dough[2].Phase, dough[3].Phase)
	}
}

func TestParseIngredients_Tangzhong_RoutesToDough(t *testing.T) {
	groups := []IngredientGroup{
		{Phase: "tangzhong", Lines: []string{"30 g bread flour", "150 ml milk"}},
	}
	dough, other := ParseIngredients(groups)
	if len(dough) != 2 {
		t.Errorf("tangzhong should route to doughIngredients, got %d", len(dough))
	}
	if len(other) != 0 {
		t.Errorf("expected 0 other ingredients, got %d", len(other))
	}
	if dough[0].Phase != "tangzhong" || dough[1].Phase != "tangzhong" {
		t.Errorf("expected phase 'tangzhong' preserved, got %q, %q", dough[0].Phase, dough[1].Phase)
	}
}

func TestParseIngredients_CheckboxPrefix_Stripped(t *testing.T) {
	// Unicode checkbox squares (▢ □ ☐) appear in recipe prints as shopping-list
	// checkboxes and must be stripped before quantity/unit parsing.
	cases := []struct {
		line string
		qty  string
		unit string
		name string
	}{
		{"▢95 g all-purpose flour", "95", "g", "all-purpose flour"},
		{"▢ 125 g bread flour", "125", "g", "bread flour"},
		{"□5 g fine sea salt", "5", "g", "fine sea salt"},
		{"☐0.25 teaspoon baking soda", "0.25", "tsp", "baking soda"},
	}
	for _, c := range cases {
		dough, _ := ParseIngredients(doughLines(c.line))
		d := dough[0]
		if d.Quantity != c.qty || d.Unit != c.unit || d.IngredientName != c.name {
			t.Errorf("line=%q: got qty=%q unit=%q name=%q; want qty=%q unit=%q name=%q",
				c.line, d.Quantity, d.Unit, d.IngredientName, c.qty, c.unit, c.name)
		}
		if !d.ParseOK {
			t.Errorf("line=%q: expected ParseOK=true", c.line)
		}
	}
}

func TestParseIngredients_UnrecognizedPhase_WithFlour_RoutesToDough(t *testing.T) {
	// A never-seen-before header ("Stiff Sweet Starter") isn't in the explicit phase lists.
	// It must still route to dough because its ingredients actually contain flour.
	groups := []IngredientGroup{
		{Phase: "Stiff Sweet Starter", Lines: []string{
			"15 g sourdough starter", "15 g honey", "30 g water", "60 g bread flour",
		}},
	}
	dough, other := ParseIngredients(groups)
	if len(dough) != 4 {
		t.Errorf("expected 4 dough ingredients, got dough=%d other=%d", len(dough), len(other))
	}
	if len(other) != 0 {
		t.Errorf("expected 0 other ingredients, got %d", len(other))
	}
	for _, d := range dough {
		if d.Phase != "Stiff Sweet Starter" {
			t.Errorf("expected verbatim phase 'Stiff Sweet Starter', got %q", d.Phase)
		}
	}
}

func TestParseIngredients_UnrecognizedPhase_NoFlour_RoutesToOther(t *testing.T) {
	// A never-seen-before header with no flour-like ingredient must route to other,
	// regardless of what the header is named. This is the case that broke the earlier
	// "default everything to flour" approach: real recipes have named sections
	// (roasted vegetables, glazes) with no flour at all.
	groups := []IngredientGroup{
		{Phase: "Roasted Pepper", Lines: []string{
			"1 bell pepper, cut into chunks", "Olive oil", "Salt & pepper",
		}},
	}
	dough, other := ParseIngredients(groups)
	if len(dough) != 0 {
		t.Errorf("expected 0 dough ingredients, got %d", len(dough))
	}
	if len(other) != 3 {
		t.Errorf("expected 3 other ingredients, got %d", len(other))
	}
	for _, o := range other {
		if o.Phase != "Roasted Pepper" {
			t.Errorf("expected verbatim phase 'Roasted Pepper', got %q", o.Phase)
		}
	}
}

func TestParseIngredients_UnrecognizedPhase_VerbatimCasePreserved(t *testing.T) {
	// Verbatim display means original casing survives, not a lowercased constant.
	groups := []IngredientGroup{
		{Phase: "Main Dough", Lines: []string{"575 g high-protein bread flour"}},
	}
	dough, _ := ParseIngredients(groups)
	if dough[0].Phase != "Main Dough" {
		t.Errorf("expected verbatim phase 'Main Dough', got %q", dough[0].Phase)
	}
}
