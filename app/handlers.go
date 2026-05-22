package main

import (
    "database/sql"
    "encoding/json"
    "log"
    "net/http"
    "strconv"
    "strings"
)

// homeHandler serves the site homepage and renders a list of recipes.
// Route: GET /
func homeHandler(w http.ResponseWriter, r *http.Request) {
    log.Printf("Route invoked: GET /")

    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    recipes, err := getAllRecipesSimple()
    if err != nil {
        http.Error(w, "Failed to get recipes", http.StatusInternalServerError)
        return
    }

    data := struct {
        Recipes []RecipeSimple
    }{
        Recipes: recipes,
    }

    err = templates.ExecuteTemplate(w, "home.html", data)
    if err != nil {
        log.Printf("Template execution error: %v", err)
        http.Error(w, "Failed to render template: "+err.Error(), http.StatusInternalServerError)
    }
}

// recipeDetailHandler serves a detailed recipe page by ID.
// Route: GET /recipes/<id>/
func recipeDetailHandler(w http.ResponseWriter, r *http.Request) {
    log.Printf("Route invoked: GET /recipes/<id>/")

    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    pathParts := strings.Split(r.URL.Path, "/")
    if len(pathParts) < 3 {
        http.Error(w, "Invalid recipe ID", http.StatusBadRequest)
        return
    }

    id, err := strconv.Atoi(pathParts[2])
    if err != nil {
        http.Error(w, "Invalid recipe ID", http.StatusBadRequest)
        return
    }

    recipe, err := getRecipeByID(id)
    if err != nil {
        if err == sql.ErrNoRows {
            http.Error(w, "Recipe not found", http.StatusNotFound)
        } else {
            http.Error(w, "Failed to get recipe", http.StatusInternalServerError)
        }
        return
    }

    err = templates.ExecuteTemplate(w, "recipe_detail.html", recipe)
    if err != nil {
        log.Printf("Template execution error: %v", err)
        http.Error(w, "Failed to render template: "+err.Error(), http.StatusInternalServerError)
    }
}

// apiOverviewHandler returns the available API routes as JSON.
// Route: GET /api
func apiOverviewHandler(w http.ResponseWriter, r *http.Request) {
    log.Printf("Route invoked: GET /api")

    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    routes := map[string]string{
        "create_user_url":  "http://localhost:3000/api/user/create/",
        "current_user_url": "http://localhost:3000/api/user/me/",
        "user_token_url":   "http://localhost:3000/api/user/token/",
        "recipes_url":      "http://localhost:3000/api/recipe/recipes/{?ingredients,tags}",
        "recipe_url":       "http://localhost:3000/api/recipe/recipes/{id}/",
        "recipe_image_url": "http://localhost:3000/api/recipe/recipes/{id}/upload-image/",
        "ingredients_url":  "http://localhost:3000/api/recipe/ingredients/{?assigned_only}",
        "ingredient_url":   "http://localhost:3000/api/recipe/ingredients/{id}/",
        "tags_url":         "http://localhost:3000/api/recipe/tags/{?assigned_only}",
        "tag_url":          "http://localhost:3000/api/recipe/tags/{id}/",
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(routes)
}

// userCreateHandler creates a new user in the database.
// Route: POST /api/user/create/
func userCreateHandler(w http.ResponseWriter, r *http.Request) {
    log.Printf("Route invoked: POST /api/user/create/")

    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    var userReq UserRequest
    err := json.NewDecoder(r.Body).Decode(&userReq)
    if err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    _, err = db.Exec(
        "INSERT INTO users (email, password, name) VALUES ($1, $2, $3)",
        userReq.Email, userReq.Password, userReq.Name,
    )
    if err != nil {
        log.Printf("Failed to create user: %v", err)
        http.Error(w, "Failed to create user: "+err.Error(), http.StatusInternalServerError)
        return
    }

    user := User{
        Email: userReq.Email,
        Name:  userReq.Name,
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(user)
}

// userMeHandler returns or updates the current user (mocked behaviour).
// Route: GET/PUT/PATCH /api/user/me/
func userMeHandler(w http.ResponseWriter, r *http.Request) {
    log.Printf("Route invoked: GET/PUT/PATCH /api/user/me/")

    switch r.Method {
    case http.MethodGet:
        user := User{Email: "user@example.com", Name: "Example User"}
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(user)

    case http.MethodPut:
        var userReq UserRequest
        err := json.NewDecoder(r.Body).Decode(&userReq)
        if err != nil {
            http.Error(w, "Invalid request body", http.StatusBadRequest)
            return
        }
        user := User{Email: userReq.Email, Name: userReq.Name}
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(user)

    case http.MethodPatch:
        var userReq map[string]interface{}
        err := json.NewDecoder(r.Body).Decode(&userReq)
        if err != nil {
            http.Error(w, "Invalid request body", http.StatusBadRequest)
            return
        }
        user := User{Email: "user@example.com", Name: "Example User"}
        if email, ok := userReq["email"].(string); ok {
            user.Email = email
        }
        if name, ok := userReq["name"].(string); ok {
            user.Name = name
        }
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(user)

    default:
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
    }
}

// userTokenHandler returns the supplied auth token (mock behaviour).
// Route: POST /api/user/token/
func userTokenHandler(w http.ResponseWriter, r *http.Request) {
    log.Printf("Route invoked: POST /api/user/token/")

    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    var authReq AuthToken
    err := json.NewDecoder(r.Body).Decode(&authReq)
    if err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(authReq)
}

// recipeRecipesHandler lists full recipes as JSON.
// Route: GET /api/recipe/recipes/
func recipeRecipesHandler(w http.ResponseWriter, r *http.Request) {
    log.Printf("Route invoked: GET /api/recipe/recipes/")

    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    _ = r.URL.Query().Get("ingredients")
    _ = r.URL.Query().Get("tags")

    recipes, err := getAllRecipes()
    if err != nil {
        http.Error(w, "Failed to get recipes", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(recipes)
}

// recipeRecipesCreateHandler creates a new recipe and returns it.
// Route: POST /api/recipe/recipes/
func recipeRecipesCreateHandler(w http.ResponseWriter, r *http.Request) {
    log.Printf("Route invoked: POST /api/recipe/recipes/")

    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    var recipeReq struct {
        Title       string       `json:"title"`
        TimeMinutes int          `json:"time_minutes"`
        Price       string       `json:"price"`
        Link        string       `json:"link"`
        Tags        []Tag        `json:"tags"`
        Ingredients []Ingredient `json:"ingredients"`
        Description string       `json:"description"`
    }

    err := json.NewDecoder(r.Body).Decode(&recipeReq)
    if err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    var recipeID int
    err = db.QueryRow(
        "INSERT INTO recipes (title, time_minutes, price, link, description) VALUES ($1, $2, $3, $4, $5) RETURNING id",
        recipeReq.Title, recipeReq.TimeMinutes, recipeReq.Price, recipeReq.Link, recipeReq.Description,
    ).Scan(&recipeID)
    if err != nil {
        http.Error(w, "Failed to create recipe", http.StatusInternalServerError)
        return
    }

    for _, tag := range recipeReq.Tags {
        _, err = db.Exec(
            "INSERT INTO recipe_tags (recipe_id, tag_id) VALUES ($1, $2)",
            recipeID, tag.ID,
        )
        if err != nil {
            log.Printf("Failed to add tag %d to recipe %d: %v", tag.ID, recipeID, err)
        }
    }

    recipe := Recipe{
        ID:          recipeID,
        Title:       recipeReq.Title,
        TimeMinutes: recipeReq.TimeMinutes,
        Price:       recipeReq.Price,
        Link:        recipeReq.Link,
        Tags:        recipeReq.Tags,
        Ingredients: recipeReq.Ingredients,
        Description: recipeReq.Description,
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(recipe)
}

// recipeIngredientsHandler lists all ingredients as JSON.
// Route: GET /api/recipe/ingredients/
func recipeIngredientsHandler(w http.ResponseWriter, r *http.Request) {
    log.Printf("Route invoked: GET /api/recipe/ingredients/")

    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    _ = r.URL.Query().Get("assigned_only")

    rows, err := db.Query("SELECT id, name FROM ingredients")
    if err != nil {
        http.Error(w, "Failed to get ingredients", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    var ingredients []Ingredient
    for rows.Next() {
        var ing Ingredient
        if err := rows.Scan(&ing.ID, &ing.Name); err != nil {
            log.Printf("Failed to scan ingredient: %v", err)
            continue
        }
        ingredients = append(ingredients, ing)
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(ingredients)
}

// recipeIngredientsCreateHandler creates a new ingredient and returns it.
// Route: POST /api/recipe/ingredients/
func recipeIngredientsCreateHandler(w http.ResponseWriter, r *http.Request) {
    log.Printf("Route invoked: POST /api/recipe/ingredients/")

    var req struct {
        Name string `json:"name"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    var id int
    err := db.QueryRow(
        "INSERT INTO ingredients (name) VALUES ($1) RETURNING id",
        req.Name,
    ).Scan(&id)
    if err != nil {
        log.Printf("Failed to create ingredient: %v", err)
        http.Error(w, "Failed to create ingredient", http.StatusInternalServerError)
        return
    }

    ing := Ingredient{ID: id, Name: req.Name}
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(ing)
}

// adminHandler serves the database test panel.
// Route: GET /admin
func adminHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Route invoked: GET /admin")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := templates.ExecuteTemplate(w, "admin.html", nil)
	if err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, "Failed to render template: "+err.Error(), http.StatusInternalServerError)
	}
}

// recipeTagsHandler lists all tags as JSON.
// Route: GET /api/recipe/tags/
func recipeTagsHandler(w http.ResponseWriter, r *http.Request) {
    log.Printf("Route invoked: GET /api/recipe/tags/")

    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    _ = r.URL.Query().Get("assigned_only")

    rows, err := db.Query("SELECT id, name FROM tags")
    if err != nil {
        http.Error(w, "Failed to get tags", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    var tags []Tag
    for rows.Next() {
        var tag Tag
        if err := rows.Scan(&tag.ID, &tag.Name); err != nil {
            log.Printf("Failed to scan tag: %v", err)
            continue
        }
        tags = append(tags, tag)
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(tags)
}

// healthzHandler reports whether the app and its database are reachable.
// Route: GET /healthz
func healthzHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    w.Header().Set("Content-Type", "application/json")

    if err := db.Ping(); err != nil {
        w.WriteHeader(http.StatusServiceUnavailable)
        json.NewEncoder(w).Encode(map[string]string{"status": "unhealthy", "db": "down"})
        return
    }

    json.NewEncoder(w).Encode(map[string]string{"status": "ok", "db": "ok"})
}
