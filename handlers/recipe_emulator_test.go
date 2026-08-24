package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/BreadBrand/breadmachine/models"
	"github.com/google/uuid"
)

func seedRecipe(t *testing.T, recipe models.Recipe) string {
	t.Helper()
	docRef := client.Collection("Recipes").NewDoc()
	recipe.ID = docRef.ID
	if _, err := docRef.Set(context.Background(), recipe); err != nil {
		t.Fatalf("failed to seed recipe: %v", err)
	}
	return docRef.ID
}

func TestGetAllRecipes_Emulator_IncludesSeededRecipe(t *testing.T) {
	requireEmulators(t)

	id := seedRecipe(t, models.Recipe{Title: "Emulator Seeded Loaf " + uuid.NewString()})

	req := httptest.NewRequest(http.MethodGet, "/api/recipes", nil)
	w := httptest.NewRecorder()
	GetAllRecipes(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", w.Code)
	}

	var recipes []models.Recipe
	if err := json.NewDecoder(w.Body).Decode(&recipes); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	for _, r := range recipes {
		if r.ID == id {
			return
		}
	}
	t.Fatalf("seeded recipe %s not found in GetAllRecipes response (%d recipes returned)", id, len(recipes))
}

func TestGetRecipe_Emulator_Found(t *testing.T) {
	requireEmulators(t)

	id := seedRecipe(t, models.Recipe{Title: "Findable Loaf"})

	req := httptest.NewRequest(http.MethodGet, "/api/recipes/"+id, nil)
	req.SetPathValue("id", id)
	w := httptest.NewRecorder()
	GetRecipe(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", w.Code)
	}
	var recipe models.Recipe
	if err := json.NewDecoder(w.Body).Decode(&recipe); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if recipe.Title != "Findable Loaf" {
		t.Errorf("title: want %q, got %q", "Findable Loaf", recipe.Title)
	}
}

func TestGetRecipe_Emulator_NotFound(t *testing.T) {
	requireEmulators(t)

	req := httptest.NewRequest(http.MethodGet, "/api/recipes/does-not-exist", nil)
	req.SetPathValue("id", "does-not-exist-"+uuid.NewString())
	w := httptest.NewRecorder()
	GetRecipe(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status: want 404, got %d", w.Code)
	}
}

func TestCreateRecipe_Emulator_Unauthorized(t *testing.T) {
	requireEmulators(t)

	req := httptest.NewRequest(http.MethodPost, "/api/recipes", bytes.NewReader([]byte(`{}`)))
	w := httptest.NewRecorder()
	CreateRecipe(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status: want 401, got %d", w.Code)
	}
}

func TestCreateRecipe_Emulator_InvalidJSON(t *testing.T) {
	requireEmulators(t)
	token := mintIDToken(t, "create-invalid-json-user")

	req := httptest.NewRequest(http.MethodPost, "/api/recipes", bytes.NewReader([]byte("not json")))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	CreateRecipe(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: want 400, got %d", w.Code)
	}
}

func TestCreateRecipe_Emulator_InvalidYeastType(t *testing.T) {
	requireEmulators(t)
	token := mintIDToken(t, "create-invalid-yeast-user")

	body, _ := json.Marshal(models.Recipe{Title: "Bad Yeast Loaf", YeastType: "not-a-real-type"})
	req := httptest.NewRequest(http.MethodPost, "/api/recipes", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	CreateRecipe(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: want 400, got %d", w.Code)
	}
}

func TestCreateRecipe_Emulator_Success_PersistsNormalizedRecipe(t *testing.T) {
	requireEmulators(t)
	uid := "create-success-user"
	token := mintIDToken(t, uid)

	reqBody := models.Recipe{
		Title: "Emulator Create Loaf",
		DoughIngredients: []models.Ingredient{
			{IngredientName: "bread flour", Unit: "g", Quantity: 500},
			{IngredientName: "butter", Unit: "tbsp", Quantity: 2},
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/recipes", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	CreateRecipe(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d, body: %s", w.Code, w.Body.String())
	}

	var created models.Recipe
	if err := json.NewDecoder(w.Body).Decode(&created); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if created.ID == "" {
		t.Fatal("expected a generated ID")
	}
	if created.UserID != uid {
		t.Errorf("userId: want %q, got %q", uid, created.UserID)
	}
	// The recipe also has flour in grams, making it gram-dominant, so
	// convertToGrams rewrites the tbsp butter to grams too (handlers/normalize.go).
	if created.DoughIngredients[1].Unit != "g" || created.DoughIngredients[1].Grams != 30 {
		t.Errorf("butter: want unit=g grams=30 (2 tbsp × 15, converted), got unit=%q grams=%v",
			created.DoughIngredients[1].Unit, created.DoughIngredients[1].Grams)
	}

	// Confirm it actually landed in Firestore, not just the HTTP response.
	docSnap, err := client.Collection("Recipes").Doc(created.ID).Get(context.Background())
	if err != nil || !docSnap.Exists() {
		t.Fatalf("created recipe not found in Firestore: %v", err)
	}
}

func TestDeleteRecipe_Emulator_Unauthorized(t *testing.T) {
	requireEmulators(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/recipes/whatever", nil)
	req.SetPathValue("id", "whatever")
	w := httptest.NewRecorder()
	DeleteRecipe(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status: want 401, got %d", w.Code)
	}
}

func TestDeleteRecipe_Emulator_NotFound(t *testing.T) {
	requireEmulators(t)
	token := mintIDToken(t, "delete-not-found-user")

	req := httptest.NewRequest(http.MethodDelete, "/api/recipes/does-not-exist", nil)
	req.SetPathValue("id", "does-not-exist-"+uuid.NewString())
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	DeleteRecipe(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status: want 404, got %d", w.Code)
	}
}

// Regression test for the ownership check added alongside the single-recipe
// routing bug fix: DELETE must reject a caller who doesn't own the recipe.
func TestDeleteRecipe_Emulator_Forbidden_WrongOwner(t *testing.T) {
	requireEmulators(t)
	id := seedRecipe(t, models.Recipe{Title: "Someone Else's Loaf", UserID: "owner-a"})
	token := mintIDToken(t, "owner-b")

	req := httptest.NewRequest(http.MethodDelete, "/api/recipes/"+id, nil)
	req.SetPathValue("id", id)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	DeleteRecipe(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status: want 403, got %d", w.Code)
	}

	docSnap, err := client.Collection("Recipes").Doc(id).Get(context.Background())
	if err != nil || !docSnap.Exists() {
		t.Fatal("recipe should NOT have been deleted by a non-owner")
	}
}

func TestDeleteRecipe_Emulator_Success_OwnerCanDelete(t *testing.T) {
	requireEmulators(t)
	uid := "owner-c"
	id := seedRecipe(t, models.Recipe{Title: "My Own Loaf", UserID: uid})
	token := mintIDToken(t, uid)

	req := httptest.NewRequest(http.MethodDelete, "/api/recipes/"+id, nil)
	req.SetPathValue("id", id)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	DeleteRecipe(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status: want 204, got %d", w.Code)
	}

	docSnap, err := client.Collection("Recipes").Doc(id).Get(context.Background())
	if err == nil && docSnap.Exists() {
		t.Fatal("recipe should have been deleted")
	}
}
