package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
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

		// Skip documents already migrated
		if _, hasDough := data["doughIngredients"]; hasDough {
			skipped++
			fmt.Printf("skipped (already migrated): %s\n", doc.Ref.ID)
			continue
		}

		ingredients, ok := data["ingredients"].([]interface{})
		if !ok {
			skipped++
			fmt.Printf("skipped (no ingredients field): %s\n", doc.Ref.ID)
			continue
		}

		// All existing ingredients move to doughIngredients.
		// The old schema had no topping/filling concept; all phases are dough-related.
		doughIngredients := ingredients

		_, err = doc.Ref.Update(ctx, []firestore.Update{
			{Path: "doughIngredients", Value: doughIngredients},
			{Path: "otherIngredients", Value: []interface{}{}},
			{Path: "ingredients", Value: firestore.Delete},
			{Path: "meta.updatedAt", Value: time.Now()},
		})
		if err != nil {
			log.Fatalf("failed to update doc %s: %v", doc.Ref.ID, err)
		}

		updated++
		fmt.Printf("migrated: %s (%d ingredients → doughIngredients)\n", doc.Ref.ID, len(ingredients))
	}

	fmt.Printf("\ndone — migrated %d, skipped %d\n", updated, skipped)
}
