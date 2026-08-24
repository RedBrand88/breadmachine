package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"github.com/BreadBrand/breadmachine/utility"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

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
			skipped++
			continue
		}

		var firestoreUpdates []firestore.Update
		filled := 0

		if hasDough {
			backfilled, n := backfillSlice(doughRaw)
			filled += n
			firestoreUpdates = append(firestoreUpdates, firestore.Update{
				Path:  "doughIngredients",
				Value: backfilled,
			})
		}

		if hasOther {
			backfilled, n := backfillSlice(otherRaw)
			filled += n
			firestoreUpdates = append(firestoreUpdates, firestore.Update{
				Path:  "otherIngredients",
				Value: backfilled,
			})
		}

		if filled == 0 {
			fmt.Printf("skipped (already complete): %s\n", doc.Ref.ID)
			continue
		}

		if _, err := doc.Ref.Update(ctx, firestoreUpdates); err != nil {
			log.Fatalf("failed to update doc %s: %v", doc.Ref.ID, err)
		}

		updated++
		fmt.Printf("updated recipe %s (%d ingredients filled)\n", doc.Ref.ID, filled)
	}

	fmt.Printf("\ndone — updated %d recipes, skipped %d\n", updated, skipped)
}

// backfillSlice corrects densityGPerMl and grams for count-unit ingredients.
// Returns the updated slice and a count of how many fields were fixed.
func backfillSlice(ings []interface{}) ([]interface{}, int) {
	filled := 0
	result := make([]interface{}, len(ings))
	for i, ing := range ings {
		ingMap, ok := ing.(map[string]interface{})
		if !ok {
			result[i] = ing
			continue
		}

		name, _ := ingMap["ingredientName"].(string)

		existing, _ := ingMap["densityGPerMl"].(float64)
		if existing == 0 {
			density := utility.LookupDensity(name)
			ingMap["densityGPerMl"] = density
			if density > 0 {
				fmt.Printf("  %s → density %.3f\n", name, density)
				filled++
			}
		}

		unit, _ := ingMap["unit"].(string)
		if strings.ToLower(unit) == "count" {
			qty, _ := ingMap["quantity"].(float64)
			if qty > 0 {
				perUnit := utility.LookupCountWeight(name)
				if perUnit > 0 {
					correct := qty * perUnit
					stored, _ := ingMap["grams"].(float64)
					if stored != correct {
						ingMap["grams"] = correct
						fmt.Printf("  %s → grams %.0f (was %.0f)\n", name, correct, stored)
						filled++
					}
				}
			}
		}

		result[i] = ingMap
	}
	return result, filled
}
