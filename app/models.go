package main

// Recipe represents a full recipe including ingredients and tags.
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

// RecipeSimple represents a lightweight recipe view used on listing pages.
type RecipeSimple struct {
	ID          int          `json:"id"`
	Title       string       `json:"title"`
	TimeMinutes int          `json:"time_minutes"`
	Price       string       `json:"price"`
	Link        string       `json:"link"`
	Ingredients []Ingredient `json:"ingredients"`
	Tags        []Tag        `json:"tags"`
}

// Ingredient represents an ingredient that can be associated with recipes.
type Ingredient struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Amount string `json:"amount"`
	Unit   string `json:"unit"`
}

// Tag represents a classification tag for recipes.
type Tag struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// User represents a user in the system (without password).
type User struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

// UserRequest is the payload for creating or updating a user.
type UserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

// AuthToken represents authentication credentials supplied by a client.
type AuthToken struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
