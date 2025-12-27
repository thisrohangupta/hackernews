// TrueNorth - Unified Portfolio Intelligence Platform
// Entry point for the web server
package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/findosh/truenorth/internal/config"
	"github.com/findosh/truenorth/internal/handlers"
	"github.com/findosh/truenorth/internal/middleware"
	"github.com/findosh/truenorth/internal/services/auth"
	"github.com/findosh/truenorth/internal/storage"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize database
	db, err := storage.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := db.Migrate(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize repositories
	userRepo := storage.NewUserRepository(db)
	sessionRepo := storage.NewSessionRepository(db)
	portfolioRepo := storage.NewPortfolioRepository(db)
	holdingRepo := storage.NewHoldingRepository(db)
	scenarioRepo := storage.NewScenarioRepository(db)

	// Initialize services
	authService := auth.NewService(cfg, userRepo, sessionRepo)

	// Get template directory
	templateDir := getTemplateDir()

	// Initialize handlers
	h, err := handlers.New(
		cfg,
		templateDir,
		authService,
		userRepo,
		portfolioRepo,
		holdingRepo,
		scenarioRepo,
	)
	if err != nil {
		log.Fatalf("Failed to initialize handlers: %v", err)
	}

	// Initialize auth middleware
	authMiddleware := middleware.NewAuth(authService)

	// Setup routes
	mux := http.NewServeMux()

	// Static files
	staticDir := getStaticDir()
	fs := http.FileServer(http.Dir(staticDir))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// Public routes
	mux.HandleFunc("/", h.Home)
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			h.Login(w, r)
		} else {
			h.LoginPage(w, r)
		}
	})
	mux.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			h.Register(w, r)
		} else {
			h.RegisterPage(w, r)
		}
	})
	mux.HandleFunc("/logout", h.Logout)

	// Protected routes (require authentication)
	mux.Handle("/dashboard", authMiddleware.RequireAuth(http.HandlerFunc(h.Dashboard)))
	mux.Handle("/portfolio/new", authMiddleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			h.CreatePortfolio(w, r)
		} else {
			h.NewPortfolioPage(w, r)
		}
	})))
	mux.Handle("/portfolio/", authMiddleware.RequireAuth(http.HandlerFunc(h.PortfolioView)))
	mux.Handle("/import", authMiddleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			h.ImportCSV(w, r)
		} else {
			h.ImportPage(w, r)
		}
	})))
	mux.Handle("/scenarios", authMiddleware.RequireAuth(http.HandlerFunc(h.ScenariosPage)))

	// API routes
	mux.Handle("/api/scenarios/simulate", authMiddleware.RequireAuth(http.HandlerFunc(h.SimulateScenario)))
	mux.Handle("/api/scenarios", authMiddleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			h.SaveScenario(w, r)
		case http.MethodDelete:
			h.DeleteScenario(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})))
	mux.Handle("/api/template.csv", http.HandlerFunc(h.DownloadTemplate))

	// Apply global middleware
	handler := middleware.Chain(
		mux,
		middleware.Recover,
		middleware.SecurityHeaders,
		middleware.Logger,
	)

	// Start server
	addr := ":" + cfg.Port
	log.Printf("TrueNorth server starting on http://localhost%s", addr)
	log.Printf("Environment: %s", cfg.Environment)

	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func getTemplateDir() string {
	// Try relative path first
	if _, err := os.Stat("web/templates"); err == nil {
		return "web/templates"
	}

	// Try from executable location
	exe, _ := os.Executable()
	dir := filepath.Dir(exe)
	templateDir := filepath.Join(dir, "web", "templates")
	if _, err := os.Stat(templateDir); err == nil {
		return templateDir
	}

	// Fallback
	return "web/templates"
}

func getStaticDir() string {
	// Try relative path first
	if _, err := os.Stat("web/static"); err == nil {
		return "web/static"
	}

	// Try from executable location
	exe, _ := os.Executable()
	dir := filepath.Dir(exe)
	staticDir := filepath.Join(dir, "web", "static")
	if _, err := os.Stat(staticDir); err == nil {
		return staticDir
	}

	// Fallback
	return "web/static"
}
