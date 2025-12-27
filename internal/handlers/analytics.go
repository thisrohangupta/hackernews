package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/findosh/truenorth/internal/middleware"
	"github.com/findosh/truenorth/internal/models"
)

// APIPerformance returns portfolio performance data as JSON
func (h *Handler) APIPerformance(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		h.jsonError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	portfolioID := r.URL.Query().Get("portfolio")
	period := r.URL.Query().Get("period")
	if period == "" {
		period = models.Period1Year
	}

	portfolio, err := h.getPortfolioForUser(user, portfolioID)
	if err != nil {
		h.jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	if h.analyticsService == nil {
		h.jsonError(w, "Analytics service not available", http.StatusServiceUnavailable)
		return
	}

	portfolio.CalculateTotals()
	performance := h.analyticsService.CalculatePortfolioPerformance(portfolio, period)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(performance)
}

// APIRiskReward returns risk-reward matrix as JSON
func (h *Handler) APIRiskReward(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		h.jsonError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	portfolioID := r.URL.Query().Get("portfolio")

	portfolio, err := h.getPortfolioForUser(user, portfolioID)
	if err != nil {
		h.jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	if h.analyticsService == nil {
		h.jsonError(w, "Analytics service not available", http.StatusServiceUnavailable)
		return
	}

	portfolio.CalculateTotals()
	riskReward := h.analyticsService.CalculateRiskRewardMatrix(portfolio)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(riskReward)
}

// APIExpenses returns expense analysis as JSON
func (h *Handler) APIExpenses(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		h.jsonError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	portfolioID := r.URL.Query().Get("portfolio")

	portfolio, err := h.getPortfolioForUser(user, portfolioID)
	if err != nil {
		h.jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	if h.analyticsService == nil {
		h.jsonError(w, "Analytics service not available", http.StatusServiceUnavailable)
		return
	}

	portfolio.CalculateTotals()
	expenses := h.analyticsService.CalculateExpenses(portfolio)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(expenses)
}

// APITimeSeries returns historical value time series as JSON
func (h *Handler) APITimeSeries(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		h.jsonError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	portfolioID := r.URL.Query().Get("portfolio")
	period := r.URL.Query().Get("period")
	if period == "" {
		period = models.Period1Year
	}

	portfolio, err := h.getPortfolioForUser(user, portfolioID)
	if err != nil {
		h.jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	if h.analyticsService == nil {
		h.jsonError(w, "Analytics service not available", http.StatusServiceUnavailable)
		return
	}

	portfolio.CalculateTotals()
	timeSeries := h.analyticsService.GenerateTimeSeries(portfolio, period)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(timeSeries)
}

// APIMarketStatus returns current market status
func (h *Handler) APIMarketStatus(w http.ResponseWriter, r *http.Request) {
	if h.marketDataSvc == nil {
		h.jsonError(w, "Market data service not available", http.StatusServiceUnavailable)
		return
	}

	status := h.marketDataSvc.GetMarketStatus()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// APIQuote returns a quote for a ticker
func (h *Handler) APIQuote(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		h.jsonError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	ticker := r.URL.Query().Get("ticker")
	if ticker == "" {
		h.jsonError(w, "ticker parameter required", http.StatusBadRequest)
		return
	}

	if h.marketDataSvc == nil {
		h.jsonError(w, "Market data service not available", http.StatusServiceUnavailable)
		return
	}

	quote, err := h.marketDataSvc.GetQuote(ticker)
	if err != nil {
		h.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(quote)
}

// APIRefreshPrices updates portfolio with live prices
func (h *Handler) APIRefreshPrices(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		h.jsonError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	portfolioID := r.URL.Query().Get("portfolio")

	portfolio, err := h.getPortfolioForUser(user, portfolioID)
	if err != nil {
		h.jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	if h.marketDataSvc == nil {
		h.jsonError(w, "Market data service not available", http.StatusServiceUnavailable)
		return
	}

	// Update prices
	if err := h.marketDataSvc.UpdatePortfolioValues(portfolio); err != nil {
		h.jsonError(w, "Failed to refresh prices: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Save updated holdings
	for _, holding := range portfolio.Holdings {
		if err := h.holdingRepo.Update(&holding); err != nil {
			// Log but don't fail
			continue
		}
	}

	// Update portfolio last sync
	if err := h.portfolioRepo.Update(portfolio); err != nil {
		h.jsonError(w, "Failed to save portfolio: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"total_value": portfolio.TotalValue,
		"holdings":    len(portfolio.Holdings),
	})
}

// Helper to get portfolio for authenticated user
func (h *Handler) getPortfolioForUser(user *models.User, portfolioID string) (*models.Portfolio, error) {
	portfolios, err := h.portfolioRepo.GetByUserID(user.ID)
	if err != nil {
		return nil, err
	}

	if len(portfolios) == 0 {
		return nil, err
	}

	// If specific ID requested, find it
	if portfolioID != "" {
		for _, p := range portfolios {
			if p.ID.String() == portfolioID {
				return h.portfolioRepo.GetByID(p.ID)
			}
		}
	}

	// Return first portfolio
	return h.portfolioRepo.GetByID(portfolios[0].ID)
}
