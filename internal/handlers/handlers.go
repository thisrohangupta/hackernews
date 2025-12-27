// Package handlers provides HTTP request handlers
package handlers

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"

	"github.com/findosh/truenorth/internal/config"
	"github.com/findosh/truenorth/internal/services/analytics"
	"github.com/findosh/truenorth/internal/services/auth"
	"github.com/findosh/truenorth/internal/services/marketdata"
	"github.com/findosh/truenorth/internal/storage"
	"github.com/shopspring/decimal"
)

// Handler contains all HTTP handlers and dependencies
type Handler struct {
	cfg              *config.Config
	templates        *template.Template
	authService      *auth.Service
	analyticsService *analytics.Service
	marketDataSvc    *marketdata.Service
	userRepo         *storage.UserRepository
	portfolioRepo    *storage.PortfolioRepository
	holdingRepo      *storage.HoldingRepository
	scenarioRepo     *storage.ScenarioRepository
}

// New creates a new handler with all dependencies
func New(
	cfg *config.Config,
	templateDir string,
	authService *auth.Service,
	analyticsService *analytics.Service,
	marketDataSvc *marketdata.Service,
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
		cfg:              cfg,
		templates:        tmpl,
		authService:      authService,
		analyticsService: analyticsService,
		marketDataSvc:    marketDataSvc,
		userRepo:         userRepo,
		portfolioRepo:    portfolioRepo,
		holdingRepo:      holdingRepo,
		scenarioRepo:     scenarioRepo,
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
		"formatMoney":   formatMoney,
		"formatPercent": formatPercent,
		"formatDecimal": formatDecimal,
		"add":           func(a, b int) int { return a + b },
		"sub":           func(a, b int) int { return a - b },
		"isPositive":    isPositive,
		"isNegative":    isNegative,
		"signClass":     signClass,
	}
}

func formatMoney(v interface{}) string {
	var val float64
	switch t := v.(type) {
	case float64:
		val = t
	case decimal.Decimal:
		val = t.InexactFloat64()
	case string:
		return "$" + t
	default:
		return "$0"
	}

	if val >= 1000000 {
		return fmt.Sprintf("$%.2fM", val/1000000)
	} else if val >= 1000 {
		return fmt.Sprintf("$%.2fK", val/1000)
	}
	return fmt.Sprintf("$%.2f", val)
}

func formatPercent(v interface{}) string {
	var val float64
	switch t := v.(type) {
	case float64:
		val = t
	case decimal.Decimal:
		val = t.InexactFloat64()
	case string:
		return t + "%"
	default:
		return "0%"
	}
	return fmt.Sprintf("%.2f%%", val)
}

func formatDecimal(v interface{}) string {
	switch t := v.(type) {
	case decimal.Decimal:
		return t.StringFixed(2)
	case float64:
		return fmt.Sprintf("%.2f", t)
	default:
		return "0.00"
	}
}

func isPositive(v interface{}) bool {
	switch t := v.(type) {
	case decimal.Decimal:
		return t.IsPositive()
	case float64:
		return t > 0
	default:
		return false
	}
}

func isNegative(v interface{}) bool {
	switch t := v.(type) {
	case decimal.Decimal:
		return t.IsNegative()
	case float64:
		return t < 0
	default:
		return false
	}
}

func signClass(v interface{}) string {
	var val float64
	switch t := v.(type) {
	case decimal.Decimal:
		val = t.InexactFloat64()
	case float64:
		val = t
	default:
		return ""
	}

	if val > 0 {
		return "positive"
	} else if val < 0 {
		return "negative"
	}
	return ""
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
