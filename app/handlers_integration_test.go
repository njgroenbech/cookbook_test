//go:build integration

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// TestMain sets up a real database connection, seeds it, and tears it down
// after all integration tests have run.
func TestMain(m *testing.M) {
	var err error
	db, err = connectDB()
	if err != nil {
		fmt.Fprintf(os.Stderr, "skipping integration tests: cannot connect to DB: %v\n", err)
		os.Exit(0)
	}
	defer db.Close()

	// Always start from a clean slate so seeding is deterministic.
	// Ignore errors — tables may not exist on the very first run.
	db.Exec("TRUNCATE users, recipe_tags, recipe_ingredients, recipes, ingredients, tags RESTART IDENTITY CASCADE")

	initDB()

	funcMap := template.FuncMap{
		"add":   func(a, b int) int { return a + b },
		"split": strings.Split,
	}
	templates = template.Must(template.New("").Funcs(funcMap).ParseGlob("templates/*.html"))

	os.Exit(m.Run())
}

// --- homeHandler ---

func TestHomeHandler_Get(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	homeHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	// Seeded data includes "Spaghetti Carbonara" — confirm it rendered
	if !strings.Contains(w.Body.String(), "Spaghetti Carbonara") {
		t.Error("expected rendered page to contain seeded recipe title")
	}
}

// --- recipeDetailHandler ---

func TestRecipeDetailHandler_Get(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/recipes/1/", nil)
	w := httptest.NewRecorder()
	recipeDetailHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "Spaghetti Carbonara") {
		t.Error("expected recipe detail page to contain recipe title")
	}
}

func TestRecipeDetailHandler_NotFound(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/recipes/99999/", nil)
	w := httptest.NewRecorder()
	recipeDetailHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// --- userCreateHandler ---

func TestUserCreateHandler_Post(t *testing.T) {
	body := `{"email":"integration@test.com","name":"Integration User","password":"secret"}`
	req := httptest.NewRequest(http.MethodPost, "/api/user/create/", strings.NewReader(body))
	w := httptest.NewRecorder()
	userCreateHandler(w, req)

	t.Cleanup(func() {
		db.Exec("DELETE FROM users WHERE email = 'integration@test.com'")
	})

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var u User
	if err := json.NewDecoder(w.Body).Decode(&u); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if u.Email != "integration@test.com" {
		t.Errorf("expected email integration@test.com, got %s", u.Email)
	}
}

// --- recipeRecipesHandler ---

func TestRecipeRecipesHandler_Get(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/recipe/recipes/", nil)
	w := httptest.NewRecorder()
	recipeRecipesHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var recipes []Recipe
	if err := json.NewDecoder(w.Body).Decode(&recipes); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(recipes) == 0 {
		t.Error("expected at least one seeded recipe")
	}
}

// --- recipeRecipesCreateHandler ---

func TestRecipeRecipesCreateHandler_Post(t *testing.T) {
	body := `{"title":"Test Recipe","time_minutes":15,"price":"8.00","link":"","description":"Test description","tags":[],"ingredients":[]}`
	req := httptest.NewRequest(http.MethodPost, "/api/recipe/recipes/", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	recipeRecipesCreateHandler(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var recipe Recipe
	if err := json.NewDecoder(w.Body).Decode(&recipe); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if recipe.ID == 0 {
		t.Error("expected non-zero recipe ID in response")
	}

	t.Cleanup(func() {
		db.Exec("DELETE FROM recipes WHERE id = $1", recipe.ID)
	})
}

// --- recipeIngredientsHandler ---

func TestRecipeIngredientsHandler_Get(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/recipe/ingredients/", nil)
	w := httptest.NewRecorder()
	recipeIngredientsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var ingredients []Ingredient
	if err := json.NewDecoder(w.Body).Decode(&ingredients); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(ingredients) == 0 {
		t.Error("expected at least one seeded ingredient")
	}
}

// --- recipeTagsHandler ---

func TestRecipeTagsHandler_Get(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/recipe/tags/", nil)
	w := httptest.NewRecorder()
	recipeTagsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var tags []Tag
	if err := json.NewDecoder(w.Body).Decode(&tags); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(tags) == 0 {
		t.Error("expected at least one seeded tag")
	}
}
