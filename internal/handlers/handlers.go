// Package handlers provides HTTP request handlers
package handlers

import (
	"html/template"
	"net/http"
	"path/filepath"

	"github.com/findosh/truenorth/internal/config"
	"github.com/findosh/truenorth/internal/services/auth"
	"github.com/findosh/truenorth/internal/storage"
)

// Handler contains all HTTP handlers and dependencies
type Handler struct {
	cfg           *config.Config
	templates     *template.Template
	authService   *auth.Service
	userRepo      *storage.UserRepository
	portfolioRepo *storage.PortfolioRepository
	holdingRepo   *storage.HoldingRepository
	scenarioRepo  *storage.ScenarioRepository
}

// New creates a new handler with all dependencies
func New(
	cfg *config.Config,
	templateDir string,
	authService *auth.Service,
	userRepo *storage.UserRepository,
	portfolioRepo *storage.PortfolioRepository,
	holdingRepo *storage.HoldingRepository,
	scenarioRepo *storage.ScenarioRepository,
) (*Handler, error) {
	// Parse all templates
	pattern := filepath.Join(templateDir, "**", "*.html")
	tmpl, err := template.New("").Funcs(templateFuncs()).ParseGlob(pattern)
	if err != nil {
		// Try alternative pattern
		tmpl, err = parseTemplates(templateDir)
		if err != nil {
			return nil, err
		}
	}

	return &Handler{
		cfg:           cfg,
		templates:     tmpl,
		authService:   authService,
		userRepo:      userRepo,
		portfolioRepo: portfolioRepo,
		holdingRepo:   holdingRepo,
		scenarioRepo:  scenarioRepo,
	}, nil
}

func parseTemplates(dir string) (*template.Template, error) {
	tmpl := template.New("").Funcs(templateFuncs())

	// Parse layouts
	layouts, _ := filepath.Glob(filepath.Join(dir, "layouts", "*.html"))
	for _, f := range layouts {
		if _, err := tmpl.ParseFiles(f); err != nil {
			return nil, err
		}
	}

	// Parse pages
	pages, _ := filepath.Glob(filepath.Join(dir, "pages", "*.html"))
	for _, f := range pages {
		if _, err := tmpl.ParseFiles(f); err != nil {
			return nil, err
		}
	}

	// Parse components
	components, _ := filepath.Glob(filepath.Join(dir, "components", "*.html"))
	for _, f := range components {
		if _, err := tmpl.ParseFiles(f); err != nil {
			return nil, err
		}
	}

	return tmpl, nil
}

func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"formatMoney": formatMoney,
		"formatPercent": formatPercent,
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
	}
}

func formatMoney(v interface{}) string {
	switch val := v.(type) {
	case float64:
		if val >= 1000000 {
			return "$" + formatFloat(val/1000000) + "M"
		} else if val >= 1000 {
			return "$" + formatFloat(val/1000) + "K"
		}
		return "$" + formatFloat(val)
	case string:
		return "$" + val
	default:
		return "$0"
	}
}

func formatPercent(v interface{}) string {
	switch val := v.(type) {
	case float64:
		return formatFloat(val) + "%"
	case string:
		return val + "%"
	default:
		return "0%"
	}
}

func formatFloat(v float64) string {
	if v == float64(int(v)) {
		return string(rune(int(v)))
	}
	// Simple formatting
	return string(rune(int(v*100))) + string(rune(int(v*10)%10))
}

// render renders a template with the given data
func (h *Handler) render(w http.ResponseWriter, name string, data interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
	}
}

// redirect performs an HTTP redirect
func (h *Handler) redirect(w http.ResponseWriter, r *http.Request, url string) {
	http.Redirect(w, r, url, http.StatusSeeOther)
}

// jsonError writes a JSON error response
func (h *Handler) jsonError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write([]byte(`{"error":"` + message + `"}`))
}
