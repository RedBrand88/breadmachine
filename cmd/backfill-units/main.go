package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"github.com/BreadBrand/breadmachine/internal/parser"
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
			fmt.Printf("skipped (no ingredient arrays): %s\n", doc.Ref.ID)
			skipped++
			continue
		}

		var firestoreUpdates []firestore.Update
		normalised := 0

		if hasDough {
			updatedSlice, n := normaliseUnitsSlice(doughRaw)
			if n > 0 {
				normalised += n
				firestoreUpdates = append(firestoreUpdates, firestore.Update{
					Path:  "doughIngredients",
					Value: updatedSlice,
				})
			}
		}

		if hasOther {
			updatedSlice, n := normaliseUnitsSlice(otherRaw)
			if n > 0 {
				normalised += n
				firestoreUpdates = append(firestoreUpdates, firestore.Update{
					Path:  "otherIngredients",
					Value: updatedSlice,
				})
			}
		}

		if normalised == 0 {
			fmt.Printf("skipped (already canonical): %s\n", doc.Ref.ID)
			skipped++
			continue
		}

		if _, err := doc.Ref.Update(ctx, firestoreUpdates); err != nil {
			log.Fatalf("failed to update doc %s: %v", doc.Ref.ID, err)
		}

		updated++
		fmt.Printf("updated recipe %s (%d units normalised)\n", doc.Ref.ID, normalised)
	}

	fmt.Printf("\ndone — updated %d recipes, skipped %d\n", updated, skipped)
}

// normaliseUnitsSlice applies canonical unit normalization to every ingredient in the slice.
// Returns the updated slice and a count of how many unit strings were changed.
func normaliseUnitsSlice(ings []interface{}) ([]interface{}, int) {
	changed := 0
	result := make([]interface{}, len(ings))
	for i, ing := range ings {
		ingMap, ok := ing.(map[string]interface{})
		if !ok {
			result[i] = ing
			continue
		}

		unit, _ := ingMap["unit"].(string)
		canonical := parser.CanonicalUnit(strings.ToLower(unit))
		if canonical != unit {
			fmt.Printf("  %q → %q (ingredient: %s)\n", unit, canonical, ingMap["ingredientName"])
			ingMap["unit"] = canonical
			changed++
		}

		result[i] = ingMap
	}
	return result, changed
}
