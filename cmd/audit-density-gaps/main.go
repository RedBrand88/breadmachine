package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	firebase "firebase.google.com/go/v4"
	"github.com/BreadBrand/breadmachine/models"
	"github.com/BreadBrand/breadmachine/utility"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// gapReport tracks, for each ingredient name with a lookup gap, the distinct
// recipe titles it was found in.
type gapReport map[string]map[string]bool

func (r gapReport) record(ingredientName, recipeTitle string) {
	name := strings.ToLower(strings.TrimSpace(ingredientName))
	if r[name] == nil {
		r[name] = map[string]bool{}
	}
	r[name][recipeTitle] = true
}

func (r gapReport) print(label string) {
	if len(r) == 0 {
		fmt.Printf("%s: no gaps found\n", label)
		return
	}
	names := make([]string, 0, len(r))
	for name := range r {
		names = append(names, name)
	}
	sort.Strings(names)

	fmt.Printf("%s: %d gap(s)\n", label, len(r))
	for _, name := range names {
		titles := make([]string, 0, len(r[name]))
		for t := range r[name] {
			titles = append(titles, t)
		}
		sort.Strings(titles)
		fmt.Printf("  %q — used in: %s\n", name, strings.Join(titles, ", "))
	}
}

func main() {
	ctx := context.Background()

	opt := option.WithCredentialsFile("/etc/breadmachine/serviceAccountKey.json")
	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		log.Fatalf("failed to create firebase app: %v", err)
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		log.Fatalf("failed to create firestore client: %v", err)
	}
	defer client.Close()

	log.Println("Firebase initialized successfully")

	iter := client.Collection("Recipes").Documents(ctx)
	defer iter.Stop()

	densityGaps := gapReport{}
	countWeightGaps := gapReport{}
	recipesScanned := 0

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatalf("failed to iterate: %v", err)
		}

		var recipe models.Recipe
		if err := doc.DataTo(&recipe); err != nil {
			fmt.Printf("skipped (failed to decode): %s (%v)\n", doc.Ref.ID, err)
			continue
		}
		recipesScanned++

		checkIngredients(recipe.DoughIngredients, recipe.Title, densityGaps, countWeightGaps)
		checkIngredients(recipe.OtherIngredients, recipe.Title, densityGaps, countWeightGaps)
	}

	fmt.Printf("\nscanned %d recipes\n\n", recipesScanned)
	densityGaps.print("Density gaps (LookupDensity == 0)")
	fmt.Println()
	countWeightGaps.print("Count-weight gaps (LookupCountWeight == 0)")

	if len(densityGaps) > 0 || len(countWeightGaps) > 0 {
		os.Exit(1)
	}
}

// checkIngredients records a gap for any ingredient whose relevant lookup
// (density for everything, count-weight for unit=="count") comes back 0 —
// the same silent-fallback shape that caused the blueberries mixed-unit
// display bug.
func checkIngredients(ings []models.Ingredient, recipeTitle string, densityGaps, countWeightGaps gapReport) {
	for _, ing := range ings {
		if strings.ToLower(ing.Unit) == "count" {
			if utility.LookupCountWeight(ing.IngredientName) == 0 {
				countWeightGaps.record(ing.IngredientName, recipeTitle)
			}
			continue
		}
		if utility.LookupDensity(ing.IngredientName) == 0 {
			densityGaps.record(ing.IngredientName, recipeTitle)
		}
	}
}
