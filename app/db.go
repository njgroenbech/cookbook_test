package main

import (
    "database/sql"
    "log"
    "time"
)

// db is the global database handle shared across the package.
var db *sql.DB

// initDB creates the necessary tables if they do not exist.
// It runs the schema DDL statements and seeds the database when empty.
func initDB() {
    start := time.Now()
    defer dbQueryDuration.WithLabelValues("init").Observe(time.Since(start).Seconds())

    schema := `
    CREATE TABLE IF NOT EXISTS users (
        id SERIAL PRIMARY KEY,
        email TEXT NOT NULL UNIQUE,
        password TEXT NOT NULL,
        name TEXT NOT NULL
    );

    CREATE TABLE IF NOT EXISTS recipes (
        id SERIAL PRIMARY KEY,
        title TEXT NOT NULL,
        time_minutes INTEGER NOT NULL,
        price TEXT NOT NULL,
        link TEXT,
        description TEXT,
        image TEXT
    );

    CREATE TABLE IF NOT EXISTS ingredients (
        id SERIAL PRIMARY KEY,
        name TEXT NOT NULL
    );

    CREATE TABLE IF NOT EXISTS tags (
        id SERIAL PRIMARY KEY,
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

    var count int
    err = db.QueryRow("SELECT COUNT(*) FROM recipes").Scan(&count)
    if err != nil {
        log.Fatal("Failed to check recipe count:", err)
    }

    if count == 0 {
        seedDatabase()
    }
}

// seedDatabase inserts initial sample data into the database.
func seedDatabase() {
    start := time.Now()
    defer dbQueryDuration.WithLabelValues("seed").Observe(time.Since(start).Seconds())

    log.Printf("Seeding database with sample data...")

    ingredients := []string{
        "Spaghetti", "Eggs", "Pancetta", "Parmesan Cheese", "Black Pepper", "Salt",
        "Chicken Breast", "Breadcrumbs", "Mozzarella Cheese", "Tomato Sauce", "Olive Oil",
        "Garlic", "Penne Pasta", "Bell Peppers", "Zucchini", "Cherry Tomatoes", "Basil",
        "Butter", "Flour", "Salmon Fillet", "Lemon", "Dill",
    }

    for _, ing := range ingredients {
        _, err := db.Exec("INSERT INTO ingredients (name) VALUES ($1)", ing)
        if err != nil {
            log.Printf("Failed to insert ingredient %s: %v", ing, err)
        }
    }

    tags := []string{"Italian", "Quick", "Dinner", "Vegetarian", "Healthy", "Seafood"}
    for _, tag := range tags {
        _, err := db.Exec("INSERT INTO tags (name) VALUES ($1)", tag)
        if err != nil {
            log.Printf("Failed to insert tag %s: %v", tag, err)
        }
    }

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
        var recipeID int
        err := db.QueryRow(
            "INSERT INTO recipes (title, time_minutes, price, link, description) VALUES ($1, $2, $3, $4, $5) RETURNING id",
            recipe.title, recipe.timeMinutes, recipe.price, recipe.link, recipe.description,
        ).Scan(&recipeID)
        if err != nil {
            log.Printf("Failed to insert recipe %s: %v", recipe.title, err)
            continue
        }

        switch recipe.title {
        case "Spaghetti Carbonara":
            addRecipeIngredient(recipeID, 1, "400", "g")
            addRecipeIngredient(recipeID, 2, "4", "large")
            addRecipeIngredient(recipeID, 3, "200", "g")
            addRecipeIngredient(recipeID, 4, "100", "g")
            addRecipeIngredient(recipeID, 5, "1", "tsp")
            addRecipeIngredient(recipeID, 6, "1", "tsp")
            addRecipeTag(recipeID, 1)
            addRecipeTag(recipeID, 3)

        case "Chicken Parmesan":
            addRecipeIngredient(recipeID, 7, "2", "pieces")
            addRecipeIngredient(recipeID, 8, "150", "g")
            addRecipeIngredient(recipeID, 9, "100", "g")
            addRecipeIngredient(recipeID, 10, "300", "ml")
            addRecipeIngredient(recipeID, 11, "3", "tbsp")
            addRecipeIngredient(recipeID, 4, "50", "g")
            addRecipeIngredient(recipeID, 2, "2", "large")
            addRecipeTag(recipeID, 1)
            addRecipeTag(recipeID, 3)

        case "Pasta Primavera":
            addRecipeIngredient(recipeID, 13, "350", "g")
            addRecipeIngredient(recipeID, 14, "1", "piece")
            addRecipeIngredient(recipeID, 15, "1", "piece")
            addRecipeIngredient(recipeID, 16, "200", "g")
            addRecipeIngredient(recipeID, 12, "3", "cloves")
            addRecipeIngredient(recipeID, 11, "3", "tbsp")
            addRecipeIngredient(recipeID, 17, "15", "leaves")
            addRecipeIngredient(recipeID, 4, "50", "g")
            addRecipeTag(recipeID, 1)
            addRecipeTag(recipeID, 2)
            addRecipeTag(recipeID, 4)
            addRecipeTag(recipeID, 5)

        case "Garlic Butter Salmon":
            addRecipeIngredient(recipeID, 20, "4", "fillets")
            addRecipeIngredient(recipeID, 18, "3", "tbsp")
            addRecipeIngredient(recipeID, 12, "4", "cloves")
            addRecipeIngredient(recipeID, 21, "1", "piece")
            addRecipeIngredient(recipeID, 22, "2", "tbsp")
            addRecipeIngredient(recipeID, 11, "2", "tbsp")
            addRecipeTag(recipeID, 2)
            addRecipeTag(recipeID, 3)
            addRecipeTag(recipeID, 5)
            addRecipeTag(recipeID, 6)
        }
    }
}

// addRecipeIngredient adds an ingredient association for a recipe.
func addRecipeIngredient(recipeID int, ingredientID int, amount, unit string) {
    start := time.Now()
    defer dbQueryDuration.WithLabelValues("insert_recipe_ingredient").Observe(time.Since(start).Seconds())

    _, err := db.Exec(
        "INSERT INTO recipe_ingredients (recipe_id, ingredient_id, amount, unit) VALUES ($1, $2, $3, $4)",
        recipeID, ingredientID, amount, unit,
    )
    if err != nil {
        log.Printf("Failed to add recipe ingredient: %v", err)
    }
}

// addRecipeTag associates a tag with a recipe.
func addRecipeTag(recipeID int, tagID int) {
    start := time.Now()
    defer dbQueryDuration.WithLabelValues("insert_recipe_tag").Observe(time.Since(start).Seconds())

    _, err := db.Exec(
        "INSERT INTO recipe_tags (recipe_id, tag_id) VALUES ($1, $2)",
        recipeID, tagID,
    )
    if err != nil {
        log.Printf("Failed to add recipe tag: %v", err)
    }
}

// getAllRecipesSimple retrieves lightweight recipe entries.
func getAllRecipesSimple() ([]RecipeSimple, error) {
    start := time.Now()
    defer dbQueryDuration.WithLabelValues("select_all_simple").Observe(time.Since(start).Seconds())

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

        recipe.Tags, err = getTagsForRecipe(recipe.ID)
        if err != nil {
            return nil, err
        }

        recipes = append(recipes, recipe)
    }

    return recipes, nil
}

// getAllRecipes retrieves full recipes including ingredients and tags.
func getAllRecipes() ([]Recipe, error) {
    start := time.Now()
    defer dbQueryDuration.WithLabelValues("select_all").Observe(time.Since(start).Seconds())

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

        recipe.Ingredients, err = getIngredientsForRecipe(recipe.ID)
        if err != nil {
            return nil, err
        }

        recipe.Tags, err = getTagsForRecipe(recipe.ID)
        if err != nil {
            return nil, err
        }

        recipes = append(recipes, recipe)
    }

    return recipes, nil
}

// getRecipeByID retrieves a recipe by its ID including ingredients and tags.
func getRecipeByID(id int) (*Recipe, error) {
    start := time.Now()
    defer dbQueryDuration.WithLabelValues("select_by_id").Observe(time.Since(start).Seconds())

    var recipe Recipe
    err := db.QueryRow("SELECT id, title, time_minutes, price, link, description FROM recipes WHERE id = $1", id).
        Scan(&recipe.ID, &recipe.Title, &recipe.TimeMinutes, &recipe.Price, &recipe.Link, &recipe.Description)
    if err != nil {
        return nil, err
    }

    recipe.Ingredients, err = getIngredientsForRecipe(recipe.ID)
    if err != nil {
        return nil, err
    }

    recipe.Tags, err = getTagsForRecipe(recipe.ID)
    if err != nil {
        return nil, err
    }

    return &recipe, nil
}

// getIngredientsForRecipe returns ingredients associated with a recipe.
func getIngredientsForRecipe(recipeID int) ([]Ingredient, error) {
    start := time.Now()
    defer dbQueryDuration.WithLabelValues("select_ingredients").Observe(time.Since(start).Seconds())

    rows, err := db.Query(`
        SELECT i.id, i.name, ri.amount, ri.unit 
        FROM ingredients i 
        JOIN recipe_ingredients ri ON i.id = ri.ingredient_id 
        WHERE ri.recipe_id = $1`, recipeID)
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

// getTagsForRecipe returns tags associated with a recipe.
func getTagsForRecipe(recipeID int) ([]Tag, error) {
    start := time.Now()
    defer dbQueryDuration.WithLabelValues("select_tags").Observe(time.Since(start).Seconds())

    rows, err := db.Query(`
        SELECT t.id, t.name 
        FROM tags t 
        JOIN recipe_tags rt ON t.id = rt.tag_id 
        WHERE rt.recipe_id = $1`, recipeID)
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
