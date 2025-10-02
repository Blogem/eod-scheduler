package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/blogem/eod-scheduler/controllers"
	"github.com/blogem/eod-scheduler/database"
	"github.com/blogem/eod-scheduler/repositories"
	"github.com/blogem/eod-scheduler/services"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	// Initialize database
	dbPath := "eod_scheduler.db"
	if err := database.InitializeDatabase(dbPath); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.CloseDB()

	// Get database connection
	db := database.GetDB()

	// Initialize repositories
	repos := repositories.NewRepositories(db)

	// Initialize services
	srvs := services.NewServices(repos)

	// Initialize controllers
	ctrl := controllers.NewControllers(srvs)

	// Set up router
	r := setupRouter(ctrl)

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("🚀 EoD Scheduler starting on port %s\n", port)
	fmt.Printf("📂 Visit: http://localhost:%s\n", port)
	fmt.Printf("🗃️  Database: %s\n", dbPath)

	log.Fatal(http.ListenAndServe(":"+port, r))
}

// setupRouter configures all routes
func setupRouter(ctrl *controllers.Controllers) *chi.Mux {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30)) // 30 second timeout
	r.Use(middleware.Compress(5))

	// Add debugging middleware
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Printf("🔍 Request: %s %s\n", r.Method, r.URL.Path)
			next.ServeHTTP(w, r)
		})
	})

	// Static files (if we add any later)
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("static/"))))

	// Dashboard routes
	r.Get("/", ctrl.Dashboard.Index)

	// Team management routes
	r.Route("/team", func(r chi.Router) {
		r.Get("/", ctrl.Team.Index)
		r.Post("/", ctrl.Team.Create)
		r.Get("/{id}/edit", ctrl.Team.Edit)
		r.Post("/{id}", ctrl.Team.Update)
		r.Post("/{id}/delete", ctrl.Team.Delete)
	})

	// Working hours configuration routes
	r.Route("/hours", func(r chi.Router) {
		r.Get("/", ctrl.WorkingHours.Index)
		r.Post("/", ctrl.WorkingHours.Update)
	})

	// Schedule routes
	r.Route("/schedule", func(r chi.Router) {
		r.Get("/", ctrl.Schedule.Index)
		r.Get("/week/{date}", ctrl.Schedule.Week)
		r.Post("/generate", ctrl.Schedule.Generate)

		// Takeover routes
		r.Get("/takeover", ctrl.Schedule.ShowTakeoverForm)
		r.Post("/takeover", ctrl.Schedule.CreateTakeover)

		// Edit routes
		r.Get("/edit/{id}", ctrl.Schedule.ShowEditForm)
		r.Post("/edit/{id}", ctrl.Schedule.UpdateEntry)

		// Remove override
		r.Post("/remove/{id}", ctrl.Schedule.RemoveOverride)
	})

	// Health check endpoint
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status": "healthy", "service": "eod-scheduler"}`)
	})

	// Simple test endpoint
	r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "<h1>Test Route Works!</h1><p>Server is responding correctly.</p>")
	})

	return r
}
