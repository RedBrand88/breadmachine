package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"github.com/BreadBrand/breadmachine/utility"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// unitToGrams mirrors the conversion table in handlers/recipe.go.
// Volume units assume water density (1 g/ml); density adjustment is separate.
var unitToGrams = map[string]float64{
	"g":     1,
	"kg":    1000,
	"oz":    28.35,
	"lb":    453.592,
	"fl oz": 29.57,
	"ml":    1,
	"l":     1000,
	"tsp":   5,
	"tbsp":  15,
	"cup":   240,
}

func main() {
	dryRun := flag.Bool("dry-run", false, "print what would change without writing to Firestore")
	flag.Parse()

	if *dryRun {
		log.Println("dry-run mode — no writes will be made")
	}

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

	updated := 0
	skipped := 0

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatalf("failed to iterate: %v", err)
		}

		data := doc.Data()

		doughRaw, hasDough := data["doughIngredients"].([]interface{})
		otherRaw, hasOther := data["otherIngredients"].([]interface{})

		if !hasDough && !hasOther {
			fmt.Printf("skipped (no ingredient arrays): %s\n", doc.Ref.ID)
			skipped++
			continue
		}

		var firestoreUpdates []firestore.Update
		filled := 0

		if hasDough {
			backfilled, n := backfillSlice(doughRaw)
			filled += n
			if n > 0 {
				firestoreUpdates = append(firestoreUpdates, firestore.Update{
					Path:  "doughIngredients",
					Value: backfilled,
				})
			}
		}

		if hasOther {
			backfilled, n := backfillSlice(otherRaw)
			filled += n
			if n > 0 {
				firestoreUpdates = append(firestoreUpdates, firestore.Update{
					Path:  "otherIngredients",
					Value: backfilled,
				})
			}
		}

		if filled == 0 {
			fmt.Printf("skipped (already complete): %s\n", doc.Ref.ID)
			skipped++
			continue
		}

		if *dryRun {
			fmt.Printf("[dry-run] would update recipe %s (%d ingredients)\n", doc.Ref.ID, filled)
			updated++
			continue
		}

		if _, err := doc.Ref.Update(ctx, firestoreUpdates); err != nil {
			log.Fatalf("failed to update doc %s: %v", doc.Ref.ID, err)
		}

		updated++
		fmt.Printf("updated recipe %s (%d ingredients backfilled)\n", doc.Ref.ID, filled)
	}

	fmt.Printf("\ndone — updated %d recipes, skipped %d\n", updated, skipped)
}

// toFloat64 extracts a numeric field from a Firestore map regardless of
// whether Firestore stored it as int64 (whole numbers) or float64 (decimals).
func toFloat64(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int64:
		return float64(n)
	}
	return 0
}

// backfillSlice computes the grams field for any ingredient where it is zero
// or absent. unit and quantity are never modified — original measurements are
// preserved so the frontend converter toggle continues to work.
func backfillSlice(ings []interface{}) ([]interface{}, int) {
	filled := 0
	result := make([]interface{}, len(ings))
	for i, ing := range ings {
		ingMap, ok := ing.(map[string]interface{})
		if !ok {
			result[i] = ing
			continue
		}

		existing := toFloat64(ingMap["grams"])
		if existing > 0 {
			result[i] = ingMap
			continue
		}

		name, _ := ingMap["ingredientName"].(string)
		unit := strings.ToLower(func() string {
			u, _ := ingMap["unit"].(string)
			return u
		}())
		qty := toFloat64(ingMap["quantity"])

		if qty <= 0 {
			result[i] = ingMap
			continue
		}

		var grams float64
		if unit == "count" {
			perUnit := utility.LookupCountWeight(name)
			if perUnit == 0 {
				fmt.Printf("  skipping %q (count unit, unknown per-unit weight)\n", name)
				result[i] = ingMap
				continue
			}
			grams = qty * perUnit
		} else if mult, ok := unitToGrams[unit]; ok {
			grams = qty * mult
		} else {
			fmt.Printf("  skipping %q (unrecognised unit %q)\n", name, unit)
			result[i] = ingMap
			continue
		}

		ingMap["grams"] = grams
		fmt.Printf("  %s → grams %.2f (%.4g %s)\n", name, grams, qty, unit)
		filled++
		result[i] = ingMap
	}
	return result, filled
}
