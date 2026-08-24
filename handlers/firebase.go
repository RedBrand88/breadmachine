package handlers

import (
	"context"
	"log"
	"net/http"
	"strings"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"

	"github.com/BreadBrand/breadmachine/models"
)

var client *firestore.Client
var authClient *auth.Client

// InitFirebase initializes the Firebase Firestore client
func InitFirebase() error {
	opt := option.WithCredentialsFile("/etc/breadmachine/serviceAccountKey.json")
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		return err
	}

	authClient, err = app.Auth(context.Background())
	if err != nil {
		return err
	}

	client, err = app.Firestore(context.Background())
	if err != nil {
		return err
	}

	log.Println("Firebase initialized successfully")
	return nil
}

// authenticate verifies the Firebase ID token in the request's Authorization
// header (format "Bearer <token>") and returns the caller's UID. ok is false
// if the header is missing, malformed, or the token fails verification —
// callers decide how to report that themselves, since each protected route
// currently returns a different error shape (see CreateRecipe vs. ParseHandler).
func authenticate(r *http.Request) (uid string, ok bool) {
	tokenString := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
	if tokenString == "" {
		return "", false
	}
	token, err := authClient.VerifyIDToken(r.Context(), tokenString)
	if err != nil {
		return "", false
	}
	return token.UID, true
}

// SaveRecipe saves a recipe to Firebase
func SaveRecipe(recipe models.Recipe) (string, error) {
	docRef, _, err := client.Collection("Recipes").Add(context.Background(), recipe)
	if err != nil {
		return "", err
	}
	return docRef.ID, nil
}

// FetchAllRecipesFromFirebase retrieves all recipes from Firestore
func FetchAllRecipesFromFirebase() ([]models.Recipe, error) {
	var recipes []models.Recipe
	iter := client.Collection("Recipes").Documents(context.Background())

	for {
		doc, err := iter.Next()
		if err != nil {
			break
		}
		var recipe models.Recipe
		doc.DataTo(&recipe)
		recipe.ID = doc.Ref.ID
		recipes = append(recipes, recipe)
	}
	return recipes, nil
}
