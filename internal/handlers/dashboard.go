package handlers

import (
	"net/http"

	"github.com/findosh/truenorth/internal/middleware"
	"github.com/findosh/truenorth/internal/models"
)

// Dashboard renders the main dashboard
func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		h.redirect(w, r, "/login")
		return
	}

	// Get user's portfolios
	portfolios, err := h.portfolioRepo.GetByUserID(user.ID)
	if err != nil {
		http.Error(w, "Failed to load portfolios", http.StatusInternalServerError)
		return
	}

	// If no portfolios, redirect to create one
	if len(portfolios) == 0 {
		h.redirect(w, r, "/portfolio/new")
		return
	}

	// Get the first portfolio (or selected one)
	portfolio := portfolios[0]
	if portfolioID := r.URL.Query().Get("portfolio"); portfolioID != "" {
		for _, p := range portfolios {
			if p.ID.String() == portfolioID {
				portfolio = p
				break
			}
		}
	}

	// Load full portfolio with holdings
	fullPortfolio, err := h.portfolioRepo.GetByID(portfolio.ID)
	if err != nil {
		http.Error(w, "Failed to load portfolio", http.StatusInternalServerError)
		return
	}

	// Calculate totals and allocation
	fullPortfolio.CalculateTotals()
	allocation := fullPortfolio.CalculateAllocation()

	// Detect alerts
	alertDetector := models.NewAlertDetector()
	alerts := alertDetector.DetectAlerts(fullPortfolio, allocation)

	// Calculate analytics (P1 features)
	var performance *models.PortfolioPerformance
	var riskReward *models.RiskRewardMatrix
	var expenses *models.PortfolioExpenses
	var marketStatus interface{}

	if h.analyticsService != nil && len(fullPortfolio.Holdings) > 0 {
		period := r.URL.Query().Get("period")
		if period == "" {
			period = models.Period1Year
		}
		performance = h.analyticsService.CalculatePortfolioPerformance(fullPortfolio, period)
		riskReward = h.analyticsService.CalculateRiskRewardMatrix(fullPortfolio)
		expenses = h.analyticsService.CalculateExpenses(fullPortfolio)
	}

	if h.marketDataSvc != nil {
		marketStatus = h.marketDataSvc.GetMarketStatus()
	}

	data := map[string]interface{}{
		"Title":        "Dashboard - TrueNorth",
		"User":         user,
		"Portfolios":   portfolios,
		"Portfolio":    fullPortfolio,
		"Allocation":   allocation,
		"Alerts":       alerts,
		"AlertCount":   len(alerts),
		"HasHoldings":  len(fullPortfolio.Holdings) > 0,
		"Performance":  performance,
		"RiskReward":   riskReward,
		"Expenses":     expenses,
		"MarketStatus": marketStatus,
	}

	h.render(w, "dashboard.html", data)
}

// Home renders the landing page
func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user != nil {
		h.redirect(w, r, "/dashboard")
		return
	}

	data := map[string]interface{}{
		"Title": "TrueNorth - Unified Portfolio Intelligence",
	}
	h.render(w, "home.html", data)
}
