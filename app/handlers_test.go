package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// --- userMeHandler ---

func TestUserMeHandler_Get(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/user/me/", nil)
	w := httptest.NewRecorder()
	userMeHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var u User
	if err := json.NewDecoder(w.Body).Decode(&u); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if u.Email == "" || u.Name == "" {
		t.Error("expected non-empty email and name")
	}
}

func TestUserMeHandler_Put(t *testing.T) {
	body := `{"email":"new@example.com","name":"New Name","password":"x"}`
	req := httptest.NewRequest(http.MethodPut, "/api/user/me/", strings.NewReader(body))
	w := httptest.NewRecorder()
	userMeHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var u User
	if err := json.NewDecoder(w.Body).Decode(&u); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if u.Email != "new@example.com" {
		t.Errorf("expected email new@example.com, got %s", u.Email)
	}
}

func TestUserMeHandler_Put_InvalidBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPut, "/api/user/me/", strings.NewReader("not json"))
	w := httptest.NewRecorder()
	userMeHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestUserMeHandler_Patch(t *testing.T) {
	body := `{"name":"Patched Name"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/user/me/", strings.NewReader(body))
	w := httptest.NewRecorder()
	userMeHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var u User
	if err := json.NewDecoder(w.Body).Decode(&u); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if u.Name != "Patched Name" {
		t.Errorf("expected name 'Patched Name', got %s", u.Name)
	}
}

func TestUserMeHandler_Patch_InvalidBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPatch, "/api/user/me/", strings.NewReader("not json"))
	w := httptest.NewRecorder()
	userMeHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestUserMeHandler_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/api/user/me/", nil)
	w := httptest.NewRecorder()
	userMeHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

// --- userTokenHandler ---

func TestUserTokenHandler_Post(t *testing.T) {
	body := `{"email":"user@example.com","password":"secret"}`
	req := httptest.NewRequest(http.MethodPost, "/api/user/token/", strings.NewReader(body))
	w := httptest.NewRecorder()
	userTokenHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var token AuthToken
	if err := json.NewDecoder(w.Body).Decode(&token); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if token.Email != "user@example.com" {
		t.Errorf("expected echoed email, got %s", token.Email)
	}
}

func TestUserTokenHandler_Post_InvalidBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/user/token/", strings.NewReader("not json"))
	w := httptest.NewRecorder()
	userTokenHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestUserTokenHandler_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/user/token/", nil)
	w := httptest.NewRecorder()
	userTokenHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

// --- recipeDetailHandler (error paths only — no DB required) ---

func TestRecipeDetailHandler_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/recipes/1/", nil)
	w := httptest.NewRecorder()
	recipeDetailHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestRecipeDetailHandler_InvalidID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/recipes/abc/", nil)
	w := httptest.NewRecorder()
	recipeDetailHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// --- prometheusMiddleware + statusResponseWriter ---

func TestPrometheusMiddleware_PassThrough(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("hello"))
	})

	handler := prometheusMiddleware(inner)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201 to pass through, got %d", w.Code)
	}
	if w.Body.String() != "hello" {
		t.Errorf("expected body 'hello', got %s", w.Body.String())
	}
}

func TestStatusResponseWriter_DefaultCode(t *testing.T) {
	w := httptest.NewRecorder()
	srw := &statusResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

	if srw.statusCode != http.StatusOK {
		t.Errorf("expected default status 200, got %d", srw.statusCode)
	}
}

func TestStatusResponseWriter_WriteHeader(t *testing.T) {
	w := httptest.NewRecorder()
	srw := &statusResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
	srw.WriteHeader(http.StatusNotFound)

	if srw.statusCode != http.StatusNotFound {
		t.Errorf("expected captured status 404, got %d", srw.statusCode)
	}
	if w.Code != http.StatusNotFound {
		t.Errorf("expected underlying writer status 404, got %d", w.Code)
	}
}

// --- userCreateHandler (error paths only — no DB required) ---

func TestUserCreateHandler_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/user/create/", nil)
	w := httptest.NewRecorder()
	userCreateHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestUserCreateHandler_InvalidBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/user/create/", bytes.NewBufferString("not json"))
	w := httptest.NewRecorder()
	userCreateHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// --- recipeRecipesCreateHandler (error paths only — no DB required) ---

func TestRecipeRecipesCreateHandler_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/recipe/recipes/", nil)
	w := httptest.NewRecorder()
	recipeRecipesCreateHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestRecipeRecipesCreateHandler_InvalidBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/recipe/recipes/", bytes.NewBufferString("not json"))
	w := httptest.NewRecorder()
	recipeRecipesCreateHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// --- homeHandler (method check only — no DB required) ---

func TestHomeHandler_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	w := httptest.NewRecorder()
	homeHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

// --- recipeRecipesHandler (method check only — no DB required) ---

func TestRecipeRecipesHandler_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/api/recipe/recipes/", nil)
	w := httptest.NewRecorder()
	recipeRecipesHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

// --- recipeIngredientsHandler (method check only — no DB required) ---

func TestRecipeIngredientsHandler_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/recipe/ingredients/", nil)
	w := httptest.NewRecorder()
	recipeIngredientsHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

// --- recipeTagsHandler (method check only — no DB required) ---

func TestRecipeTagsHandler_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/recipe/tags/", nil)
	w := httptest.NewRecorder()
	recipeTagsHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

// --- recipeDetailHandler short path ---

func TestRecipeDetailHandler_ShortPath(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	recipeDetailHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for short path, got %d", w.Code)
	}
}
