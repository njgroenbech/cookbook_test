package main

import (
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
    _ "github.com/lib/pq"
)

// httpRequestsTotal counts HTTP requests by method/path/status.
var httpRequestsTotal = prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Name: "http_requests_total",
        Help: "Total number of HTTP requests",
    },
    []string{"method", "path", "status"},
)

// httpRequestDuration records the duration of HTTP handlers.
var httpRequestDuration = prometheus.NewHistogramVec(
    prometheus.HistogramOpts{
        Name:    "http_request_duration_seconds",
        Help:    "HTTP request duration in seconds",
        Buckets: prometheus.DefBuckets,
    },
    []string{"method", "path"},
)

// dbQueryDuration records database query durations by query type.
var dbQueryDuration = prometheus.NewHistogramVec(
    prometheus.HistogramOpts{
        Name:    "db_query_duration_seconds",
        Help:    "Database query duration in seconds",
        Buckets: prometheus.DefBuckets,
    },
    []string{"query_type"},
)

// init registers Prometheus metrics used by the application.
func init() {
    prometheus.MustRegister(httpRequestsTotal)
    prometheus.MustRegister(httpRequestDuration)
    prometheus.MustRegister(dbQueryDuration)
}

// templates holds parsed HTML templates used for rendering pages.
var templates *template.Template

// prometheusMiddleware wraps HTTP handlers to record request metrics and status codes.
func prometheusMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()

        // Create a custom response writer to capture status code; default to 200.
        srw := &statusResponseWriter{ResponseWriter: w, statusCode: 200}

        // Call the next handler
        next.ServeHTTP(srw, r)

        // Record metrics
        duration := time.Since(start).Seconds()
        httpRequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration)
        httpRequestsTotal.WithLabelValues(r.Method, r.URL.Path, strconv.Itoa(srw.statusCode)).Inc()
    })
}

// statusResponseWriter wraps http.ResponseWriter to capture the HTTP status code.
type statusResponseWriter struct {
    http.ResponseWriter
    statusCode int
}

// WriteHeader records the status code and forwards the call to the underlying writer.
func (w *statusResponseWriter) WriteHeader(code int) {
    w.statusCode = code
    w.ResponseWriter.WriteHeader(code)
}

// main initializes the application, registers routes, and starts the HTTP server.
func main() {
    // Initialize database using connectDB()
    var err error
    db, err = connectDB()
    if err != nil {
        log.Fatal("Failed to connect to database:", err)
    }
    defer db.Close()

    // Initialize database schema and seed when necessary
    initDB()

    // Create template function map and parse templates
    funcMap := template.FuncMap{
        "add":   func(a, b int) int { return a + b },
        "split": strings.Split,
    }
    templates = template.Must(template.New("").Funcs(funcMap).ParseGlob("templates/*.html"))

    // Set up static files and metrics endpoint
    http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
    http.Handle("/metrics", promhttp.Handler())

    // Create API mux and register handlers
    apiMux := http.NewServeMux()
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
    apiMux.HandleFunc("/admin", adminHandler)
    apiMux.HandleFunc("/api/recipe/ingredients/", recipeIngredientsHandler)
    apiMux.HandleFunc("/api/recipe/tags/", recipeTagsHandler)
    apiMux.HandleFunc("/healthz", healthzHandler)

    apiMux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("ok"))
    })

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

    // Apply middleware to API routes and start server
    http.Handle("/", prometheusMiddleware(apiMux))
    log.Printf("Server starting on :3000...")
    log.Fatal(http.ListenAndServe(":3000", nil))
}
