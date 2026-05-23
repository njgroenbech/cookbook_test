package main

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func setupMockDB(t *testing.T) sqlmock.Sqlmock {
	t.Helper()
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	originalDB := db
	db = mockDB
	t.Cleanup(func() {
		mockDB.Close()
		db = originalDB
	})
	return mock
}

func TestGetTagsForRecipe_Success(t *testing.T) {
	mock := setupMockDB(t)

	rows := sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "Italian")
	mock.ExpectQuery("SELECT t.id, t.name").WithArgs(1).WillReturnRows(rows)

	tags, err := getTagsForRecipe(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tags) != 1 || tags[0].Name != "Italian" {
		t.Errorf("unexpected tags: %v", tags)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestGetTagsForRecipe_QueryError(t *testing.T) {
	mock := setupMockDB(t)

	mock.ExpectQuery("SELECT t.id, t.name").WithArgs(1).WillReturnError(errors.New("query failed"))

	_, err := getTagsForRecipe(1)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestGetIngredientsForRecipe_Success(t *testing.T) {
	mock := setupMockDB(t)

	rows := sqlmock.NewRows([]string{"id", "name", "amount", "unit"}).
		AddRow(1, "Spaghetti", "400", "g")
	mock.ExpectQuery("SELECT i.id, i.name, ri.amount, ri.unit").WithArgs(1).WillReturnRows(rows)

	ings, err := getIngredientsForRecipe(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ings) != 1 || ings[0].Name != "Spaghetti" {
		t.Errorf("unexpected ingredients: %v", ings)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestGetIngredientsForRecipe_QueryError(t *testing.T) {
	mock := setupMockDB(t)

	mock.ExpectQuery("SELECT i.id, i.name").WithArgs(1).WillReturnError(errors.New("query failed"))

	_, err := getIngredientsForRecipe(1)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestGetAllRecipesSimple_Empty(t *testing.T) {
	mock := setupMockDB(t)

	mock.ExpectQuery("SELECT id, title, time_minutes, price, link FROM recipes").
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "time_minutes", "price", "link"}))

	recipes, err := getAllRecipesSimple()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if recipes != nil {
		t.Errorf("expected nil slice, got %v", recipes)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestGetAllRecipesSimple_WithData(t *testing.T) {
	mock := setupMockDB(t)

	recipeRows := sqlmock.NewRows([]string{"id", "title", "time_minutes", "price", "link"}).
		AddRow(1, "Spaghetti Carbonara", 25, "12.50", "http://example.com")
	mock.ExpectQuery("SELECT id, title, time_minutes, price, link FROM recipes").
		WillReturnRows(recipeRows)

	tagRows := sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "Italian")
	mock.ExpectQuery("SELECT t.id, t.name").WithArgs(1).WillReturnRows(tagRows)

	recipes, err := getAllRecipesSimple()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(recipes) != 1 {
		t.Fatalf("expected 1 recipe, got %d", len(recipes))
	}
	if recipes[0].Title != "Spaghetti Carbonara" {
		t.Errorf("unexpected title: %s", recipes[0].Title)
	}
	if len(recipes[0].Tags) != 1 || recipes[0].Tags[0].Name != "Italian" {
		t.Errorf("unexpected tags: %v", recipes[0].Tags)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestGetAllRecipesSimple_QueryError(t *testing.T) {
	mock := setupMockDB(t)

	mock.ExpectQuery("SELECT id, title, time_minutes, price, link FROM recipes").
		WillReturnError(errors.New("db error"))

	_, err := getAllRecipesSimple()
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestGetAllRecipes_Empty(t *testing.T) {
	mock := setupMockDB(t)

	mock.ExpectQuery("SELECT id, title, time_minutes, price, link, description FROM recipes").
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "time_minutes", "price", "link", "description"}))

	recipes, err := getAllRecipes()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if recipes != nil {
		t.Errorf("expected nil slice, got %v", recipes)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestGetAllRecipes_WithData(t *testing.T) {
	mock := setupMockDB(t)

	recipeRows := sqlmock.NewRows([]string{"id", "title", "time_minutes", "price", "link", "description"}).
		AddRow(1, "Spaghetti Carbonara", 25, "12.50", "http://example.com", "A great pasta dish.")
	mock.ExpectQuery("SELECT id, title, time_minutes, price, link, description FROM recipes").
		WillReturnRows(recipeRows)

	ingRows := sqlmock.NewRows([]string{"id", "name", "amount", "unit"})
	mock.ExpectQuery("SELECT i.id, i.name, ri.amount, ri.unit").WithArgs(1).WillReturnRows(ingRows)

	tagRows := sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "Italian")
	mock.ExpectQuery("SELECT t.id, t.name").WithArgs(1).WillReturnRows(tagRows)

	recipes, err := getAllRecipes()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(recipes) != 1 || recipes[0].Title != "Spaghetti Carbonara" {
		t.Errorf("unexpected recipes: %v", recipes)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestGetAllRecipes_QueryError(t *testing.T) {
	mock := setupMockDB(t)

	mock.ExpectQuery("SELECT id, title, time_minutes, price, link, description FROM recipes").
		WillReturnError(errors.New("db error"))

	_, err := getAllRecipes()
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestGetRecipeByID_Success(t *testing.T) {
	mock := setupMockDB(t)

	row := sqlmock.NewRows([]string{"id", "title", "time_minutes", "price", "link", "description"}).
		AddRow(1, "Spaghetti Carbonara", 25, "12.50", "http://example.com", "A great pasta dish.")
	mock.ExpectQuery("SELECT id, title, time_minutes, price, link, description FROM recipes WHERE id").
		WithArgs(1).WillReturnRows(row)

	mock.ExpectQuery("SELECT i.id, i.name, ri.amount, ri.unit").WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "amount", "unit"}))

	mock.ExpectQuery("SELECT t.id, t.name").WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}))

	recipe, err := getRecipeByID(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if recipe.Title != "Spaghetti Carbonara" {
		t.Errorf("unexpected title: %s", recipe.Title)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestGetRecipeByID_NotFound(t *testing.T) {
	mock := setupMockDB(t)

	mock.ExpectQuery("SELECT id, title, time_minutes, price, link, description FROM recipes WHERE id").
		WithArgs(999).WillReturnError(sql.ErrNoRows)

	_, err := getRecipeByID(999)
	if err != sql.ErrNoRows {
		t.Errorf("expected ErrNoRows, got %v", err)
	}
}

func TestAddRecipeIngredient_Success(t *testing.T) {
	mock := setupMockDB(t)

	mock.ExpectExec("INSERT INTO recipe_ingredients").
		WithArgs(1, 2, "400", "g").
		WillReturnResult(sqlmock.NewResult(1, 1))

	addRecipeIngredient(1, 2, "400", "g")

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestAddRecipeIngredient_Error(t *testing.T) {
	mock := setupMockDB(t)

	mock.ExpectExec("INSERT INTO recipe_ingredients").
		WithArgs(1, 2, "400", "g").
		WillReturnError(errors.New("exec error"))

	// Should not panic; logs the error internally.
	addRecipeIngredient(1, 2, "400", "g")

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestAddRecipeTag_Success(t *testing.T) {
	mock := setupMockDB(t)

	mock.ExpectExec("INSERT INTO recipe_tags").
		WithArgs(1, 3).
		WillReturnResult(sqlmock.NewResult(1, 1))

	addRecipeTag(1, 3)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestAddRecipeTag_Error(t *testing.T) {
	mock := setupMockDB(t)

	mock.ExpectExec("INSERT INTO recipe_tags").
		WithArgs(1, 3).
		WillReturnError(errors.New("exec error"))

	// Should not panic; logs the error internally.
	addRecipeTag(1, 3)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestGetAllRecipesSimple_TagsError(t *testing.T) {
	mock := setupMockDB(t)

	recipeRows := sqlmock.NewRows([]string{"id", "title", "time_minutes", "price", "link"}).
		AddRow(1, "Test Recipe", 25, "12.50", "http://example.com")
	mock.ExpectQuery("SELECT id, title, time_minutes, price, link FROM recipes").
		WillReturnRows(recipeRows)

	mock.ExpectQuery("SELECT t.id, t.name").WithArgs(1).
		WillReturnError(errors.New("tags query error"))

	_, err := getAllRecipesSimple()
	if err == nil {
		t.Error("expected error from tags sub-query, got nil")
	}
}

func TestGetAllRecipes_IngredientsError(t *testing.T) {
	mock := setupMockDB(t)

	recipeRows := sqlmock.NewRows([]string{"id", "title", "time_minutes", "price", "link", "description"}).
		AddRow(1, "Test Recipe", 25, "12.50", "http://example.com", "Desc")
	mock.ExpectQuery("SELECT id, title, time_minutes, price, link, description FROM recipes").
		WillReturnRows(recipeRows)

	mock.ExpectQuery("SELECT i.id, i.name, ri.amount, ri.unit").WithArgs(1).
		WillReturnError(errors.New("ingredients query error"))

	_, err := getAllRecipes()
	if err == nil {
		t.Error("expected error from ingredients sub-query, got nil")
	}
}

func TestGetAllRecipes_TagsError(t *testing.T) {
	mock := setupMockDB(t)

	recipeRows := sqlmock.NewRows([]string{"id", "title", "time_minutes", "price", "link", "description"}).
		AddRow(1, "Test Recipe", 25, "12.50", "http://example.com", "Desc")
	mock.ExpectQuery("SELECT id, title, time_minutes, price, link, description FROM recipes").
		WillReturnRows(recipeRows)

	mock.ExpectQuery("SELECT i.id, i.name, ri.amount, ri.unit").WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "amount", "unit"}))

	mock.ExpectQuery("SELECT t.id, t.name").WithArgs(1).
		WillReturnError(errors.New("tags query error"))

	_, err := getAllRecipes()
	if err == nil {
		t.Error("expected error from tags sub-query, got nil")
	}
}

func TestGetRecipeByID_IngredientsError(t *testing.T) {
	mock := setupMockDB(t)

	row := sqlmock.NewRows([]string{"id", "title", "time_minutes", "price", "link", "description"}).
		AddRow(1, "Test Recipe", 25, "12.50", "http://example.com", "Desc")
	mock.ExpectQuery("SELECT id, title, time_minutes, price, link, description FROM recipes WHERE id").
		WithArgs(1).WillReturnRows(row)

	mock.ExpectQuery("SELECT i.id, i.name, ri.amount, ri.unit").WithArgs(1).
		WillReturnError(errors.New("ingredients error"))

	_, err := getRecipeByID(1)
	if err == nil {
		t.Error("expected error from ingredients sub-query, got nil")
	}
}

func TestGetRecipeByID_TagsError(t *testing.T) {
	mock := setupMockDB(t)

	row := sqlmock.NewRows([]string{"id", "title", "time_minutes", "price", "link", "description"}).
		AddRow(1, "Test Recipe", 25, "12.50", "http://example.com", "Desc")
	mock.ExpectQuery("SELECT id, title, time_minutes, price, link, description FROM recipes WHERE id").
		WithArgs(1).WillReturnRows(row)

	mock.ExpectQuery("SELECT i.id, i.name, ri.amount, ri.unit").WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "amount", "unit"}))

	mock.ExpectQuery("SELECT t.id, t.name").WithArgs(1).
		WillReturnError(errors.New("tags error"))

	_, err := getRecipeByID(1)
	if err == nil {
		t.Error("expected error from tags sub-query, got nil")
	}
}
