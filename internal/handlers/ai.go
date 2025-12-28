package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/findosh/truenorth/internal/middleware"
	"github.com/findosh/truenorth/internal/services/ai"
)

// AIAsk handles natural language portfolio queries
func (h *Handler) AIAsk(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := middleware.GetUser(r)
	if user == nil {
		h.jsonError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse request
	var req struct {
		Query       string            `json:"query"`
		PortfolioID string            `json:"portfolio_id"`
		Context     map[string]string `json:"context,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Query == "" {
		h.jsonError(w, "Query is required", http.StatusBadRequest)
		return
	}

	// Check if AI service is available
	if h.aiService == nil {
		h.jsonError(w, "AI service not available", http.StatusServiceUnavailable)
		return
	}

	// Get portfolio for context
	portfolio, err := h.getPortfolioForUser(user, req.PortfolioID)
	if err != nil {
		h.jsonError(w, "Failed to load portfolio", http.StatusBadRequest)
		return
	}

	if portfolio != nil {
		portfolio.CalculateTotals()
	}

	// Build query
	query := &ai.Query{
		UserID:      user.ID.String(),
		Text:        req.Query,
		PortfolioID: req.PortfolioID,
		Context:     req.Context,
	}

	// Process query
	response, err := h.aiService.Ask(r.Context(), query, portfolio)
	if err != nil {
		h.jsonError(w, "Failed to process query: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// AITaxAnalysis provides tax optimization recommendations
func (h *Handler) AITaxAnalysis(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		h.jsonError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	portfolioID := r.URL.Query().Get("portfolio")

	// Get portfolio
	portfolio, err := h.getPortfolioForUser(user, portfolioID)
	if err != nil {
		h.jsonError(w, "Failed to load portfolio", http.StatusBadRequest)
		return
	}

	if portfolio != nil {
		portfolio.CalculateTotals()
	}

	// Check if AI service is available
	if h.aiService == nil {
		h.jsonError(w, "AI service not available", http.StatusServiceUnavailable)
		return
	}

	// Get tax analysis
	taxOptimizer := ai.NewTaxOptimizer()
	analysis := taxOptimizer.AnalyzeTaxOpportunities(portfolio)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analysis)
}

// AIUsage returns AI usage statistics for the current user
func (h *Handler) AIUsage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		h.jsonError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if h.aiService == nil {
		h.jsonError(w, "AI service not available", http.StatusServiceUnavailable)
		return
	}

	stats := h.aiService.GetUsageStats(user.ID.String())

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// AIHistory returns recent AI queries for the current user
func (h *Handler) AIHistory(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		h.jsonError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if h.aiService == nil {
		h.jsonError(w, "AI service not available", http.StatusServiceUnavailable)
		return
	}

	// Get history from audit log
	// Default to last 24 hours, limit 50 entries
	since := time.Now().Add(-24 * time.Hour)
	limit := 50

	// Parse query params
	if sinceStr := r.URL.Query().Get("since"); sinceStr != "" {
		if t, err := time.Parse(time.RFC3339, sinceStr); err == nil {
			since = t
		}
	}

	entries := h.aiService.GetAuditEntries(user.ID.String(), since, limit)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"entries": entries,
		"since":   since,
		"limit":   limit,
	})
}

// AICacheStats returns cache statistics (admin only in production)
func (h *Handler) AICacheStats(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		h.jsonError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if h.aiService == nil {
		h.jsonError(w, "AI service not available", http.StatusServiceUnavailable)
		return
	}

	stats := h.aiService.GetCacheStats()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// AIInvalidateCache invalidates cache for user's portfolio
func (h *Handler) AIInvalidateCache(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := middleware.GetUser(r)
	if user == nil {
		h.jsonError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	portfolioID := r.URL.Query().Get("portfolio")
	if portfolioID == "" {
		h.jsonError(w, "portfolio parameter required", http.StatusBadRequest)
		return
	}

	if h.aiService == nil {
		h.jsonError(w, "AI service not available", http.StatusServiceUnavailable)
		return
	}

	h.aiService.InvalidateCache(portfolioID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":      true,
		"portfolio_id": portfolioID,
	})
}
