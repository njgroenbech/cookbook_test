package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	_ "modernc.org/sqlite"
)

// Database structure
type Recipe struct {
	ID          int          `json:"id"`
	Title       string       `json:"title"`
	TimeMinutes int          `json:"time_minutes"`
	Price       string       `json:"price"`
	Link        string       `json:"link"`
	Description string       `json:"description"`
	Ingredients []Ingredient `json:"ingredients"`
	Tags        []Tag        `json:"tags"`
}

type RecipeSimple struct {
	ID          int          `json:"id"`
	Title       string       `json:"title"`
	TimeMinutes int          `json:"time_minutes"`
	Price       string       `json:"price"`
	Link        string       `json:"link"`
	Ingredients []Ingredient `json:"ingredients"`
	Tags        []Tag        `json:"tags"`
}

type Ingredient struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Amount string `json:"amount"`
	Unit   string `json:"unit"`
}

type Tag struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type User struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

type UserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type AuthToken struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RecipeImage struct {
	ID    int    `json:"id"`
	Image string `json:"image"`
}

// Prometheus metrics
var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)
	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
	dbQueryDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "db_query_duration_seconds",
			Help:    "Database query duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"query_type"},
	)
)

func init() {
	// Register Prometheus metrics
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestDuration)
	prometheus.MustRegister(dbQueryDuration)
}

// Global variables
var (
	db           *sql.DB
	templates    *template.Template
	databasePath = "./demo.db"
)

// prometheusMiddleware wraps HTTP handlers to record metrics
func prometheusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Create a custom response writer to capture status code
		srw := &statusResponseWriter{ResponseWriter: w}
		
		// Call the next handler
		next.ServeHTTP(srw, r)
		
		// Record metrics
		duration := time.Since(start).Seconds()
		httpRequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration)
		httpRequestsTotal.WithLabelValues(r.Method, r.URL.Path, strconv.Itoa(srw.statusCode)).Inc()
	})
}

// statusResponseWriter captures the HTTP status code
type statusResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *statusResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func main() {
	var err error

	// Initialize database
	db, err = sql.Open("sqlite", databasePath)
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	// Initialize database schema
	initDB()

	// Create template function map
	funcMap := template.FuncMap{
		"add":   func(a, b int) int { return a + b },
		"split": strings.Split,
	}

	// Load templates with custom functions
	templates = template.Must(template.New("").Funcs(funcMap).ParseGlob("templates/*.html"))

	// Set up static file server
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Set up Prometheus metrics endpoint
	http.Handle("/metrics", promhttp.Handler())

	// Create a mux to apply middleware to API routes
	apiMux := http.NewServeMux()

	// Set up routes
	apiMux.HandleFunc("/", homeHandler)
	apiMux.HandleFunc("/recipes/", recipeDetailHandler)
	apiMux.HandleFunc("/api", apiOverviewHandler)
	apiMux.HandleFunc("/api/user/create/", userCreateHandler)
	apiMux.HandleFunc("/api/user/me/", userMeHandler)
	apiMux.HandleFunc("/api/user/token/", userTokenHandler)
	apiMux.HandleFunc("/api/recipe/recipes/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			recipeRecipesHandler(w, r)
		} else if r.Method == http.MethodPost {
			recipeRecipesCreateHandler(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	apiMux.HandleFunc("/api/recipe/ingredients/", recipeIngredientsHandler)
	apiMux.HandleFunc("/api/recipe/tags/", recipeTagsHandler)

	// Serve the raw OpenAPI spec
	apiMux.HandleFunc("/swagger/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		yamlFile, err := os.ReadFile("api-schema.yaml")
		if err != nil {
			http.Error(w, "Could not find api-schema.yaml", 500)
			return
		}
		w.Header().Set("Content-Type", "application/yaml")
		w.Write(yamlFile)
	})

	// Swagger UI
	apiMux.HandleFunc("/swagger", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<!DOCTYPE html>
<html>
<head>
    <title>API Documentation</title>
    <meta charset="utf-8"/>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
    <script>
        SwaggerUIBundle({
            url: "/swagger/openapi.yaml",
            dom_id: '#swagger-ui',
            presets: [SwaggerUIBundle.presets.apis, SwaggerUIBundle.SwaggerUIStandalonePreset],
            layout: "BaseLayout"
        });
    </script>
</body>
</html>`)
	})

	// Apply middleware to all routes
	http.Handle("/", prometheusMiddleware(apiMux))

	// Start server
	fmt.Println("Server starting on :3000...")
	log.Fatal(http.ListenAndServe(":3000", nil))
}

func initDB() {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		email TEXT NOT NULL UNIQUE,
		password TEXT NOT NULL,
		name TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS recipes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		time_minutes INTEGER NOT NULL,
		price TEXT NOT NULL,
		link TEXT,
		description TEXT,
		image TEXT
	);

	CREATE TABLE IF NOT EXISTS ingredients (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS tags (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS recipe_ingredients (
		recipe_id INTEGER,
		ingredient_id INTEGER,
		amount TEXT,
		unit TEXT,
		FOREIGN KEY (recipe_id) REFERENCES recipes(id),
		FOREIGN KEY (ingredient_id) REFERENCES ingredients(id)
	);

	CREATE TABLE IF NOT EXISTS recipe_tags (
		recipe_id INTEGER,
		tag_id INTEGER,
		FOREIGN KEY (recipe_id) REFERENCES recipes(id),
		FOREIGN KEY (tag_id) REFERENCES tags(id)
	);`

	_, err := db.Exec(schema)
	if err != nil {
		log.Fatal("Failed to create tables:", err)
	}

	// Check if we need to seed data
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM recipes").Scan(&count)
	if err != nil {
		log.Fatal("Failed to check recipe count:", err)
	}

	if count == 0 {
		seedDatabase()
	}
}

func seedDatabase() {
	fmt.Println("Seeding database with sample data...")

	// Insert ingredients
	ingredients := []string{
		"Spaghetti", "Eggs", "Pancetta", "Parmesan Cheese", "Black Pepper", "Salt",
		"Chicken Breast", "Breadcrumbs", "Mozzarella Cheese", "Tomato Sauce", "Olive Oil",
		"Garlic", "Penne Pasta", "Bell Peppers", "Zucchini", "Cherry Tomatoes", "Basil",
		"Butter", "Flour", "Salmon Fillet", "Lemon", "Dill",
	}

	for _, ing := range ingredients {
		_, err := db.Exec("INSERT INTO ingredients (name) VALUES (?)", ing)
		if err != nil {
			log.Printf("Failed to insert ingredient %s: %v", ing, err)
		}
	}

	// Insert tags
	tags := []string{"Italian", "Quick", "Dinner", "Vegetarian", "Healthy", "Seafood"}
	for _, tag := range tags {
		_, err := db.Exec("INSERT INTO tags (name) VALUES (?)", tag)
		if err != nil {
			log.Printf("Failed to insert tag %s: %v", tag, err)
		}
	}

	// Insert recipes
	recipes := []struct {
		title       string
		timeMinutes int
		price       string
		link        string
		description string
	}{
		{
			title:       "Spaghetti Carbonara",
			timeMinutes: 25,
			price:       "12.50",
			link:        "http://example.com/carbonara",
			description: "Step 1: Bring a large pot of salted water to boil and cook 400g spaghetti according to package directions.\n\nStep 2: While pasta cooks, cut 200g pancetta into small cubes and fry in a large pan over medium heat until crispy (about 5 minutes).\n\nStep 3: In a bowl, whisk together 4 large eggs, 100g grated Parmesan cheese, and plenty of black pepper.\n\nStep 4: When pasta is ready, reserve 1 cup of pasta water, then drain the pasta.\n\nStep 5: Remove the pan with pancetta from heat. Add the hot pasta to the pan and toss.\n\nStep 6: Pour the egg mixture over the pasta and toss quickly. The heat from the pasta will cook the eggs. Add pasta water bit by bit if needed to create a creamy sauce.\n\nStep 7: Serve immediately with extra Parmesan cheese and black pepper.",
		},
		{
			title:       "Chicken Parmesan",
			timeMinutes: 50,
			price:       "18.00",
			link:        "http://example.com/chicken-parm",
			description: "Step 1: Preheat oven to 200C (400F).\n\nStep 2: Place 2 chicken breasts between plastic wrap and pound to 2cm thickness.\n\nStep 3: Set up breading station: flour in one plate, 2 beaten eggs in another, and 150g breadcrumbs mixed with 50g Parmesan in a third.\n\nStep 4: Season chicken with salt and pepper, then coat in flour, dip in egg, and press into breadcrumb mixture.\n\nStep 5: Heat 3 tablespoons olive oil in a large oven-safe skillet over medium-high heat. Fry chicken until golden brown, about 4 minutes per side.\n\nStep 6: Pour 300ml tomato sauce over the chicken, then top each breast with 100g sliced mozzarella.\n\nStep 7: Transfer skillet to oven and bake for 15-20 minutes until cheese is melted and bubbly.\n\nStep 8: Garnish with fresh basil and serve with pasta or salad.",
		},
		{
			title:       "Pasta Primavera",
			timeMinutes: 30,
			price:       "10.00",
			link:        "http://example.com/primavera",
			description: "Step 1: Cook 350g penne pasta in salted boiling water according to package directions. Reserve 1 cup pasta water before draining.\n\nStep 2: While pasta cooks, chop 1 red bell pepper, 1 zucchini into bite-sized pieces, and halve 200g cherry tomatoes.\n\nStep 3: Heat 3 tablespoons olive oil in a large pan over medium-high heat. Add 3 minced garlic cloves and cook for 30 seconds.\n\nStep 4: Add bell peppers and zucchini to the pan. Cook for 5-7 minutes until vegetables are tender.\n\nStep 5: Add cherry tomatoes and cook for another 2-3 minutes until they start to soften.\n\nStep 6: Add the drained pasta to the pan with vegetables. Toss everything together, adding pasta water as needed to create a light sauce.\n\nStep 7: Season with salt and black pepper. Remove from heat and stir in fresh basil leaves.\n\nStep 8: Serve hot with grated Parmesan cheese on top.",
		},
		{
			title:       "Garlic Butter Salmon",
			timeMinutes: 20,
			price:       "22.00",
			link:        "http://example.com/salmon",
			description: "Step 1: Pat 4 salmon fillets (150g each) dry with paper towels and season both sides with salt and pepper.\n\nStep 2: Heat 2 tablespoons olive oil in a large skillet over medium-high heat.\n\nStep 3: Place salmon fillets skin-side up in the pan. Cook for 4-5 minutes until golden brown.\n\nStep 4: Flip the salmon and cook for another 3-4 minutes.\n\nStep 5: Reduce heat to medium and add 3 tablespoons butter, 4 minced garlic cloves, and juice of 1 lemon to the pan.\n\nStep 6: Spoon the garlic butter sauce over the salmon repeatedly for 1-2 minutes.\n\nStep 7: Remove from heat and sprinkle with fresh dill.\n\nStep 8: Serve immediately with the pan sauce, accompanied by rice or vegetables.",
		},
	}

	for _, recipe := range recipes {
		result, err := db.Exec(
			"INSERT INTO recipes (title, time_minutes, price, link, description) VALUES (?, ?, ?, ?, ?)",
			recipe.title, recipe.timeMinutes, recipe.price, recipe.link, recipe.description,
		)
		if err != nil {
			log.Printf("Failed to insert recipe %s: %v", recipe.title, err)
			continue
		}

		recipeID, _ := result.LastInsertId()

		// Add recipe ingredients and tags based on recipe type
		switch recipe.title {
		case "Spaghetti Carbonara":
			addRecipeIngredient(recipeID, 1, "400", "g")   // Spaghetti
			addRecipeIngredient(recipeID, 2, "4", "large") // Eggs
			addRecipeIngredient(recipeID, 3, "200", "g")   // Pancetta
			addRecipeIngredient(recipeID, 4, "100", "g")   // Parmesan Cheese
			addRecipeIngredient(recipeID, 5, "1", "tsp")   // Black Pepper
			addRecipeIngredient(recipeID, 6, "1", "tsp")   // Salt
			addRecipeTag(recipeID, 1)                      // Italian
			addRecipeTag(recipeID, 3)                      // Dinner

		case "Chicken Parmesan":
			addRecipeIngredient(recipeID, 7, "2", "pieces") // Chicken Breast
			addRecipeIngredient(recipeID, 8, "150", "g")    // Breadcrumbs
			addRecipeIngredient(recipeID, 9, "100", "g")    // Mozzarella Cheese
			addRecipeIngredient(recipeID, 10, "300", "ml")  // Tomato Sauce
			addRecipeIngredient(recipeID, 11, "3", "tbsp")  // Olive Oil
			addRecipeIngredient(recipeID, 4, "50", "g")     // Parmesan Cheese
			addRecipeIngredient(recipeID, 2, "2", "large")  // Eggs
			addRecipeTag(recipeID, 1)                       // Italian
			addRecipeTag(recipeID, 3)                       // Dinner

		case "Pasta Primavera":
			addRecipeIngredient(recipeID, 13, "350", "g")     // Penne Pasta
			addRecipeIngredient(recipeID, 14, "1", "piece")   // Bell Peppers
			addRecipeIngredient(recipeID, 15, "1", "piece")   // Zucchini
			addRecipeIngredient(recipeID, 16, "200", "g")     // Cherry Tomatoes
			addRecipeIngredient(recipeID, 12, "3", "cloves")  // Garlic
			addRecipeIngredient(recipeID, 11, "3", "tbsp")    // Olive Oil
			addRecipeIngredient(recipeID, 17, "15", "leaves") // Basil
			addRecipeIngredient(recipeID, 4, "50", "g")       // Parmesan Cheese
			addRecipeTag(recipeID, 1)                         // Italian
			addRecipeTag(recipeID, 2)                         // Quick
			addRecipeTag(recipeID, 4)                         // Vegetarian
			addRecipeTag(recipeID, 5)                         // Healthy

		case "Garlic Butter Salmon":
			addRecipeIngredient(recipeID, 20, "4", "fillets") // Salmon Fillet
			addRecipeIngredient(recipeID, 18, "3", "tbsp")    // Butter
			addRecipeIngredient(recipeID, 12, "4", "cloves")  // Garlic
			addRecipeIngredient(recipeID, 21, "1", "piece")   // Lemon
			addRecipeIngredient(recipeID, 22, "2", "tbsp")    // Dill
			addRecipeIngredient(recipeID, 11, "2", "tbsp")    // Olive Oil
			addRecipeTag(recipeID, 2)                         // Quick
			addRecipeTag(recipeID, 3)                         // Dinner
			addRecipeTag(recipeID, 5)                         // Healthy
			addRecipeTag(recipeID, 6)                         // Seafood
		}
	}
}

func addRecipeIngredient(recipeID int64, ingredientID int, amount, unit string) {
	_, err := db.Exec(
		"INSERT INTO recipe_ingredients (recipe_id, ingredient_id, amount, unit) VALUES (?, ?, ?, ?)",
		recipeID, ingredientID, amount, unit,
	)
	if err != nil {
		log.Printf("Failed to add recipe ingredient: %v", err)
	}
}

func addRecipeTag(recipeID int64, tagID int) {
	_, err := db.Exec(
		"INSERT INTO recipe_tags (recipe_id, tag_id) VALUES (?, ?)",
		recipeID, tagID,
	)
	if err != nil {
		log.Printf("Failed to add recipe tag: %v", err)
	}
}

// Handler functions
func homeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Route invoked: GET /")

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

func recipeDetailHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Route invoked: GET /recipes/<id>/")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from URL path
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

func apiOverviewHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Route invoked: GET /api")

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

func userCreateHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Route invoked: POST /api/user/create/")

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

	// Insert user into database
	_, err = db.Exec(
		"INSERT INTO users (email, password, name) VALUES (?, ?, ?)",
		userReq.Email, userReq.Password, userReq.Name,
	)
	if err != nil {
		log.Printf("Failed to create user: %v", err)
		http.Error(w, "Failed to create user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return user data (without password)
	user := User{
		Email: userReq.Email,
		Name:  userReq.Name,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func userMeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Route invoked: GET/PUT/PATCH /api/user/me/")

	switch r.Method {
	case http.MethodGet:
		// Return mock user data
		user := User{
			Email: "user@example.com",
			Name:  "Example User",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)

	case http.MethodPut:
		var userReq UserRequest
		err := json.NewDecoder(r.Body).Decode(&userReq)
		if err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		user := User{
			Email: userReq.Email,
			Name:  userReq.Name,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)

	case http.MethodPatch:
		var userReq map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&userReq)
		if err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		user := User{
			Email: "user@example.com",
			Name:  "Example User",
		}

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

func userTokenHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Route invoked: POST /api/user/token/")

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

	// Return the auth token (in a real app, this would be a JWT)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(authReq)
}

func recipeRecipesHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Route invoked: GET /api/recipe/recipes/")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters (currently unused but kept for API compatibility)
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

func recipeRecipesCreateHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Route invoked: POST /api/recipe/recipes/")

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

	// Insert recipe into database
	result, err := db.Exec(
		"INSERT INTO recipes (title, time_minutes, price, link, description) VALUES (?, ?, ?, ?, ?)",
		recipeReq.Title, recipeReq.TimeMinutes, recipeReq.Price, recipeReq.Link, recipeReq.Description,
	)
	if err != nil {
		http.Error(w, "Failed to create recipe", http.StatusInternalServerError)
		return
	}

	recipeID, _ := result.LastInsertId()

	// Return the created recipe
	recipe := Recipe{
		ID:          int(recipeID),
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

func recipeIngredientsHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Route invoked: GET /api/recipe/ingredients/")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters (currently unused but kept for API compatibility)
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

func recipeTagsHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Route invoked: GET /api/recipe/tags/")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters (currently unused but kept for API compatibility)
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

// Database helper functions
func getAllRecipesSimple() ([]RecipeSimple, error) {
	rows, err := db.Query("SELECT id, title, time_minutes, price, link FROM recipes")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recipes []RecipeSimple
	for rows.Next() {
		var recipe RecipeSimple
		if err := rows.Scan(&recipe.ID, &recipe.Title, &recipe.TimeMinutes, &recipe.Price, &recipe.Link); err != nil {
			return nil, err
		}

		// Get tags for this recipe
		recipe.Tags, err = getTagsForRecipe(recipe.ID)
		if err != nil {
			return nil, err
		}

		recipes = append(recipes, recipe)
	}

	return recipes, nil
}

func getAllRecipes() ([]Recipe, error) {
	rows, err := db.Query("SELECT id, title, time_minutes, price, link, description FROM recipes")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recipes []Recipe
	for rows.Next() {
		var recipe Recipe
		if err := rows.Scan(&recipe.ID, &recipe.Title, &recipe.TimeMinutes, &recipe.Price, &recipe.Link, &recipe.Description); err != nil {
			return nil, err
		}

		// Get ingredients for this recipe
		recipe.Ingredients, err = getIngredientsForRecipe(recipe.ID)
		if err != nil {
			return nil, err
		}

		// Get tags for this recipe
		recipe.Tags, err = getTagsForRecipe(recipe.ID)
		if err != nil {
			return nil, err
		}

		recipes = append(recipes, recipe)
	}

	return recipes, nil
}

func getRecipeByID(id int) (*Recipe, error) {
	var recipe Recipe
	err := db.QueryRow("SELECT id, title, time_minutes, price, link, description FROM recipes WHERE id = ?", id).
		Scan(&recipe.ID, &recipe.Title, &recipe.TimeMinutes, &recipe.Price, &recipe.Link, &recipe.Description)
	if err != nil {
		return nil, err
	}

	// Get ingredients for this recipe
	recipe.Ingredients, err = getIngredientsForRecipe(recipe.ID)
	if err != nil {
		return nil, err
	}

	// Get tags for this recipe
	recipe.Tags, err = getTagsForRecipe(recipe.ID)
	if err != nil {
		return nil, err
	}

	return &recipe, nil
}

func getIngredientsForRecipe(recipeID int) ([]Ingredient, error) {
	rows, err := db.Query(`
		SELECT i.id, i.name, ri.amount, ri.unit 
		FROM ingredients i 
		JOIN recipe_ingredients ri ON i.id = ri.ingredient_id 
		WHERE ri.recipe_id = ?`, recipeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ingredients []Ingredient
	for rows.Next() {
		var ing Ingredient
		if err := rows.Scan(&ing.ID, &ing.Name, &ing.Amount, &ing.Unit); err != nil {
			return nil, err
		}
		ingredients = append(ingredients, ing)
	}

	return ingredients, nil
}

func getTagsForRecipe(recipeID int) ([]Tag, error) {
	rows, err := db.Query(`
		SELECT t.id, t.name 
		FROM tags t 
		JOIN recipe_tags rt ON t.id = rt.tag_id 
		WHERE rt.recipe_id = ?`, recipeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tags []Tag
	for rows.Next() {
		var tag Tag
		if err := rows.Scan(&tag.ID, &tag.Name); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}

	return tags, nil
}
