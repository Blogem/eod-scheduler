package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"

	"gitea.com/go-chi/session"
	"github.com/blogem/eod-scheduler/authenticator"
	"github.com/blogem/eod-scheduler/controllers"
	"github.com/blogem/eod-scheduler/database"
	authmiddleware "github.com/blogem/eod-scheduler/middleware"
	"github.com/blogem/eod-scheduler/repositories"
	"github.com/blogem/eod-scheduler/services"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Failed to load the env vars: %v", err)
	}

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

	// Initialize Auth0 provider
	auth, err := authenticator.NewAuth0Provider()
	if err != nil {
		log.Fatalf("Failed to initialize Auth0 provider: %v", err)
	}

	// Set up router
	r, err := setupRouter(ctrl, auth)
	if err != nil {
		log.Fatalf("Failed to setup router: %v", err)
	}

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
func setupRouter(ctrl *controllers.Controllers, auth authenticator.Provider) (*chi.Mux, error) {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second)) // 60 second timeout for OAuth callbacks
	r.Use(middleware.Compress(5))

	// Determine if we should use secure cookies (HTTPS)
	useSecureCookies := os.Getenv("USE_HTTPS") == "true"

	// Session middleware
	sessionHandler, err := session.Sessioner(session.Options{
		Provider:       "memory",
		ProviderConfig: "",
		CookieName:     "eod_session",
		Secure:         useSecureCookies, // Set to true when USE_HTTPS=true (production)
		Gclifetime:     3600,             // Session lifetime in seconds
		Maxlifetime:    3600,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize session: %w", err)
	}
	r.Use(sessionHandler)

	// Add debugging middleware
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Printf("🔍 Request: %s %s\n", r.Method, r.URL.Path)
			next.ServeHTTP(w, r)
		})
	})

	// Static files (if we add any later)
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("static/"))))

	// PUBLIC ROUTES (no authentication required)
	r.Get("/", ctrl.Dashboard.Index) // Home page - shows landing or dashboard based on auth
	r.Get("/login", ctrl.Auth.Login(auth))
	r.Get("/callback", ctrl.Auth.Callback(auth))
	r.Get("/logout", ctrl.Auth.Logout)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status": "healthy", "service": "eod-scheduler"}`)
	})
	r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "<h1>Test Route Works!</h1><p>Server is responding correctly.</p>")
	})

	// PROTECTED ROUTES (authentication required)
	r.Group(func(r chi.Router) {
		r.Use(authmiddleware.RequireAuth)

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
	})

	return r, nil
}
