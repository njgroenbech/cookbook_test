package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func setupTemplates(t *testing.T) {
	t.Helper()
	originalTemplates := templates
	templates = loadTemplates(t)
	t.Cleanup(func() { templates = originalTemplates })
}

// --- homeHandler ---

func TestHomeHandler_DBError(t *testing.T) {
	mock := setupMockDB(t)
	mock.ExpectQuery("SELECT id, title, time_minutes, price, link FROM recipes").
		WillReturnError(errors.New("db error"))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	homeHandler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestHomeHandler_Success(t *testing.T) {
	mock := setupMockDB(t)
	setupTemplates(t)

	recipeRows := sqlmock.NewRows([]string{"id", "title", "time_minutes", "price", "link"}).
		AddRow(1, "Spaghetti Carbonara", 25, "12.50", "http://example.com")
	mock.ExpectQuery("SELECT id, title, time_minutes, price, link FROM recipes").
		WillReturnRows(recipeRows)

	tagRows := sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "Italian")
	mock.ExpectQuery("SELECT t.id, t.name").WithArgs(1).WillReturnRows(tagRows)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	homeHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// --- recipeDetailHandler ---

// TestRecipeDetailHandler_DBNotFound returns an empty result set; QueryRow.Scan returns sql.ErrNoRows → 404.
func TestRecipeDetailHandler_DBNotFound(t *testing.T) {
	mock := setupMockDB(t)
	mock.ExpectQuery("SELECT id, title, time_minutes, price, link, description FROM recipes WHERE id").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "time_minutes", "price", "link", "description"}))

	req := httptest.NewRequest(http.MethodGet, "/recipes/1/", nil)
	w := httptest.NewRecorder()
	recipeDetailHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestRecipeDetailHandler_DBError(t *testing.T) {
	mock := setupMockDB(t)
	mock.ExpectQuery("SELECT id, title, time_minutes, price, link, description FROM recipes WHERE id").
		WithArgs(1).WillReturnError(errors.New("connection error"))

	req := httptest.NewRequest(http.MethodGet, "/recipes/1/", nil)
	w := httptest.NewRecorder()
	recipeDetailHandler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestRecipeDetailHandler_Success(t *testing.T) {
	mock := setupMockDB(t)
	setupTemplates(t)

	row := sqlmock.NewRows([]string{"id", "title", "time_minutes", "price", "link", "description"}).
		AddRow(1, "Spaghetti Carbonara", 25, "12.50", "http://example.com", "Step 1: Cook pasta.\n\nStep 2: Add sauce.")
	mock.ExpectQuery("SELECT id, title, time_minutes, price, link, description FROM recipes WHERE id").
		WithArgs(1).WillReturnRows(row)

	mock.ExpectQuery("SELECT i.id, i.name, ri.amount, ri.unit").WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "amount", "unit"}).
			AddRow(1, "Spaghetti", "400", "g"))

	mock.ExpectQuery("SELECT t.id, t.name").WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "Italian"))

	req := httptest.NewRequest(http.MethodGet, "/recipes/1/", nil)
	w := httptest.NewRecorder()
	recipeDetailHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// --- recipeRecipesHandler ---

func TestRecipeRecipesHandler_DBError(t *testing.T) {
	mock := setupMockDB(t)
	mock.ExpectQuery("SELECT id, title, time_minutes, price, link, description FROM recipes").
		WillReturnError(errors.New("db error"))

	req := httptest.NewRequest(http.MethodGet, "/api/recipe/recipes/", nil)
	w := httptest.NewRecorder()
	recipeRecipesHandler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestRecipeRecipesHandler_Success(t *testing.T) {
	mock := setupMockDB(t)

	recipeRows := sqlmock.NewRows([]string{"id", "title", "time_minutes", "price", "link", "description"}).
		AddRow(1, "Spaghetti Carbonara", 25, "12.50", "http://example.com", "A description")
	mock.ExpectQuery("SELECT id, title, time_minutes, price, link, description FROM recipes").
		WillReturnRows(recipeRows)

	mock.ExpectQuery("SELECT i.id, i.name, ri.amount, ri.unit").WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "amount", "unit"}))

	mock.ExpectQuery("SELECT t.id, t.name").WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}))

	req := httptest.NewRequest(http.MethodGet, "/api/recipe/recipes/", nil)
	w := httptest.NewRecorder()
	recipeRecipesHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var recipes []Recipe
	if err := json.NewDecoder(w.Body).Decode(&recipes); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if len(recipes) != 1 || recipes[0].Title != "Spaghetti Carbonara" {
		t.Errorf("unexpected recipes: %v", recipes)
	}
}

// --- recipeIngredientsHandler ---

func TestRecipeIngredientsHandler_DBError(t *testing.T) {
	mock := setupMockDB(t)
	mock.ExpectQuery("SELECT id, name FROM ingredients").
		WillReturnError(errors.New("db error"))

	req := httptest.NewRequest(http.MethodGet, "/api/recipe/ingredients/", nil)
	w := httptest.NewRecorder()
	recipeIngredientsHandler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestRecipeIngredientsHandler_Success(t *testing.T) {
	mock := setupMockDB(t)

	rows := sqlmock.NewRows([]string{"id", "name"}).
		AddRow(1, "Spaghetti").
		AddRow(2, "Eggs")
	mock.ExpectQuery("SELECT id, name FROM ingredients").WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/api/recipe/ingredients/", nil)
	w := httptest.NewRecorder()
	recipeIngredientsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var ings []Ingredient
	if err := json.NewDecoder(w.Body).Decode(&ings); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if len(ings) != 2 {
		t.Errorf("expected 2 ingredients, got %d", len(ings))
	}
}

// TestRecipeIngredientsHandler_ScanError: a non-numeric id column causes Scan to fail;
// the handler logs the error, skips the row, and still returns 200 with an empty list.
func TestRecipeIngredientsHandler_ScanError(t *testing.T) {
	mock := setupMockDB(t)

	rows := sqlmock.NewRows([]string{"id", "name"}).AddRow("not-an-int", "Bad Row")
	mock.ExpectQuery("SELECT id, name FROM ingredients").WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/api/recipe/ingredients/", nil)
	w := httptest.NewRecorder()
	recipeIngredientsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 on scan error, got %d", w.Code)
	}
}

// --- recipeTagsHandler ---

func TestRecipeTagsHandler_DBError(t *testing.T) {
	mock := setupMockDB(t)
	mock.ExpectQuery("SELECT id, name FROM tags").
		WillReturnError(errors.New("db error"))

	req := httptest.NewRequest(http.MethodGet, "/api/recipe/tags/", nil)
	w := httptest.NewRecorder()
	recipeTagsHandler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestRecipeTagsHandler_Success(t *testing.T) {
	mock := setupMockDB(t)

	rows := sqlmock.NewRows([]string{"id", "name"}).
		AddRow(1, "Italian").
		AddRow(2, "Quick")
	mock.ExpectQuery("SELECT id, name FROM tags").WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/api/recipe/tags/", nil)
	w := httptest.NewRecorder()
	recipeTagsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var tags []Tag
	if err := json.NewDecoder(w.Body).Decode(&tags); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if len(tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(tags))
	}
}

// TestRecipeTagsHandler_ScanError: a non-numeric id column causes Scan to fail;
// the handler logs the error, skips the row, and still returns 200 with an empty list.
func TestRecipeTagsHandler_ScanError(t *testing.T) {
	mock := setupMockDB(t)

	rows := sqlmock.NewRows([]string{"id", "name"}).AddRow("not-an-int", "Bad Row")
	mock.ExpectQuery("SELECT id, name FROM tags").WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/api/recipe/tags/", nil)
	w := httptest.NewRecorder()
	recipeTagsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 on scan error, got %d", w.Code)
	}
}

// --- userCreateHandler ---

func TestUserCreateHandler_DBError(t *testing.T) {
	mock := setupMockDB(t)
	mock.ExpectExec("INSERT INTO users").
		WillReturnError(errors.New("db error"))

	body := `{"email":"test@example.com","password":"secret","name":"Test User"}`
	req := httptest.NewRequest(http.MethodPost, "/api/user/create/", strings.NewReader(body))
	w := httptest.NewRecorder()
	userCreateHandler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestUserCreateHandler_Success(t *testing.T) {
	mock := setupMockDB(t)
	mock.ExpectExec("INSERT INTO users").
		WithArgs("test@example.com", "secret", "Test User").
		WillReturnResult(sqlmock.NewResult(1, 1))

	body := `{"email":"test@example.com","password":"secret","name":"Test User"}`
	req := httptest.NewRequest(http.MethodPost, "/api/user/create/", strings.NewReader(body))
	w := httptest.NewRecorder()
	userCreateHandler(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", w.Code)
	}
	var u User
	if err := json.NewDecoder(w.Body).Decode(&u); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if u.Email != "test@example.com" || u.Name != "Test User" {
		t.Errorf("unexpected user: %+v", u)
	}
}

// --- recipeRecipesCreateHandler ---

func TestRecipeRecipesCreateHandler_DBError(t *testing.T) {
	mock := setupMockDB(t)
	mock.ExpectQuery("INSERT INTO recipes").
		WillReturnError(errors.New("db error"))

	body := `{"title":"Test","time_minutes":30,"price":"10.00","link":"","description":"","tags":[],"ingredients":[]}`
	req := httptest.NewRequest(http.MethodPost, "/api/recipe/recipes/", strings.NewReader(body))
	w := httptest.NewRecorder()
	recipeRecipesCreateHandler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestRecipeRecipesCreateHandler_Success(t *testing.T) {
	mock := setupMockDB(t)
	mock.ExpectQuery("INSERT INTO recipes").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

	body := `{"title":"Test Recipe","time_minutes":30,"price":"15.00","link":"http://example.com","description":"Desc","tags":[],"ingredients":[]}`
	req := httptest.NewRequest(http.MethodPost, "/api/recipe/recipes/", strings.NewReader(body))
	w := httptest.NewRecorder()
	recipeRecipesCreateHandler(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", w.Code)
	}
	var recipe Recipe
	if err := json.NewDecoder(w.Body).Decode(&recipe); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if recipe.ID != 1 || recipe.Title != "Test Recipe" {
		t.Errorf("unexpected recipe: %+v", recipe)
	}
}

func TestRecipeRecipesCreateHandler_TagInsertError(t *testing.T) {
	mock := setupMockDB(t)

	mock.ExpectQuery("INSERT INTO recipes").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	// tag insert fails — handler logs and continues, still returns 201
	mock.ExpectExec("INSERT INTO recipe_tags").
		WithArgs(1, 5).
		WillReturnError(errors.New("tag insert error"))

	body := `{"title":"Test","time_minutes":10,"price":"5.00","link":"","description":"","tags":[{"id":5,"name":"Italian"}],"ingredients":[]}`
	req := httptest.NewRequest(http.MethodPost, "/api/recipe/recipes/", strings.NewReader(body))
	w := httptest.NewRecorder()
	recipeRecipesCreateHandler(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201 even on tag error, got %d", w.Code)
	}
}

// --- recipeIngredientsCreateHandler ---

func TestRecipeIngredientsCreateHandler_EmptyName(t *testing.T) {
	body := `{"name":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/recipe/ingredients/", strings.NewReader(body))
	w := httptest.NewRecorder()
	recipeIngredientsCreateHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestRecipeIngredientsCreateHandler_InvalidBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/recipe/ingredients/", strings.NewReader("not json"))
	w := httptest.NewRecorder()
	recipeIngredientsCreateHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestRecipeIngredientsCreateHandler_DBError(t *testing.T) {
	mock := setupMockDB(t)
	mock.ExpectQuery("INSERT INTO ingredients").
		WillReturnError(errors.New("db error"))

	body := `{"name":"Tomato"}`
	req := httptest.NewRequest(http.MethodPost, "/api/recipe/ingredients/", strings.NewReader(body))
	w := httptest.NewRecorder()
	recipeIngredientsCreateHandler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestRecipeIngredientsCreateHandler_Success(t *testing.T) {
	mock := setupMockDB(t)
	mock.ExpectQuery("INSERT INTO ingredients").
		WithArgs("Tomato").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(5))

	body := `{"name":"Tomato"}`
	req := httptest.NewRequest(http.MethodPost, "/api/recipe/ingredients/", strings.NewReader(body))
	w := httptest.NewRecorder()
	recipeIngredientsCreateHandler(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", w.Code)
	}
	var ing Ingredient
	if err := json.NewDecoder(w.Body).Decode(&ing); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if ing.Name != "Tomato" || ing.ID != 5 {
		t.Errorf("unexpected ingredient: %+v", ing)
	}
}

// --- adminHandler ---

func TestAdminHandler_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/admin", nil)
	w := httptest.NewRecorder()
	adminHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestAdminHandler_Success(t *testing.T) {
	setupTemplates(t)

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	w := httptest.NewRecorder()
	adminHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// --- healthzHandler ---

func TestHealthzHandler_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/healthz", nil)
	w := httptest.NewRecorder()
	healthzHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestHealthzHandler_DBDown(t *testing.T) {
	mockDB, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	originalDB := db
	db = mockDB
	defer func() { mockDB.Close(); db = originalDB }()

	mock.ExpectPing().WillReturnError(errors.New("connection refused"))

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	healthzHandler(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if resp["status"] != "unhealthy" {
		t.Errorf("expected unhealthy status, got %v", resp)
	}
}

func TestHealthzHandler_Success(t *testing.T) {
	mockDB, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	originalDB := db
	db = mockDB
	defer func() { mockDB.Close(); db = originalDB }()

	mock.ExpectPing()

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	healthzHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if resp["status"] != "ok" {
		t.Errorf("expected ok status, got %v", resp)
	}
}
