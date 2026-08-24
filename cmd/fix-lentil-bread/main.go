package main

import (
	"context"
	"fmt"
	"log"
	"time"

	firebase "firebase.google.com/go/v4"
	"github.com/BreadBrand/breadmachine/models"
	"google.golang.org/api/option"
)

const lentilBreadID = "xrbwau5U1AUuHlWINMMG"

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

	docRef := client.Collection("Recipes").Doc(lentilBreadID)
	snap, err := docRef.Get(ctx)
	if err != nil {
		log.Fatalf("failed to fetch recipe %s: %v", lentilBreadID, err)
	}

	var recipe models.Recipe
	if err := snap.DataTo(&recipe); err != nil {
		log.Fatalf("failed to deserialize recipe: %v", err)
	}

	fmt.Printf("loaded: %s (yeastType=%q)\n", recipe.Title, recipe.YeastType)

	recipe.YeastType = models.YeastTypeNone
	models.CalculateBakerPercentages(recipe.DoughIngredients)
	recipe.Meta.UpdatedAt = time.Now()

	if _, err := docRef.Set(ctx, recipe); err != nil {
		log.Fatalf("failed to save recipe: %v", err)
	}

	fmt.Println("updated yeastType → none")
	fmt.Println("recalculated baker's percentages:")
	for _, ing := range recipe.DoughIngredients {
		fmt.Printf("  %-30s %.1f%%\n", ing.IngredientName, ing.BakerPercentage)
	}
}
