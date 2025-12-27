package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/findosh/truenorth/internal/middleware"
	"github.com/findosh/truenorth/internal/models"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ScenariosPage renders the what-if scenarios page
func (h *Handler) ScenariosPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		h.redirect(w, r, "/login")
		return
	}

	portfolioID := r.URL.Query().Get("portfolio")
	if portfolioID == "" {
		h.redirect(w, r, "/dashboard")
		return
	}

	pid, err := uuid.Parse(portfolioID)
	if err != nil {
		h.redirect(w, r, "/dashboard")
		return
	}

	portfolio, err := h.portfolioRepo.GetByID(pid)
	if err != nil || portfolio == nil || portfolio.UserID != user.ID {
		h.redirect(w, r, "/dashboard?error=Portfolio+not+found")
		return
	}

	portfolio.CalculateTotals()
	allocation := portfolio.CalculateAllocation()

	// Get current allocation percentages
	currentAlloc := make(map[string]float64)
	for class, slice := range allocation.ByAssetClass {
		currentAlloc[string(class)] = slice.Percentage.InexactFloat64()
	}

	// Get saved scenarios
	scenarios, _ := h.scenarioRepo.GetByPortfolioID(pid)

	data := map[string]interface{}{
		"Title":           "Scenarios - TrueNorth",
		"User":            user,
		"Portfolio":       portfolio,
		"CurrentAlloc":    currentAlloc,
		"Scenarios":       scenarios,
		"AssetClasses":    models.AllAssetClasses(),
		"AssetClassStats": models.AssetClassReturns,
	}

	h.render(w, "scenarios.html", data)
}

// SimulateScenario handles scenario simulation requests
func (h *Handler) SimulateScenario(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		h.jsonError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var input struct {
		PortfolioID string             `json:"portfolio_id"`
		Allocations map[string]float64 `json:"allocations"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.jsonError(w, "Invalid request", http.StatusBadRequest)
		return
	}

	pid, err := uuid.Parse(input.PortfolioID)
	if err != nil {
		h.jsonError(w, "Invalid portfolio ID", http.StatusBadRequest)
		return
	}

	portfolio, err := h.portfolioRepo.GetByID(pid)
	if err != nil || portfolio == nil || portfolio.UserID != user.ID {
		h.jsonError(w, "Portfolio not found", http.StatusNotFound)
		return
	}

	portfolio.CalculateTotals()

	// Create scenario from input
	scenario := models.NewScenario(pid, "Simulation")
	for classStr, pct := range input.Allocations {
		class := models.AssetClass(classStr)
		scenario.SetAllocation(class, decimal.NewFromFloat(pct))
	}

	// Calculate projections
	scenario.CalculateProjections(portfolio.TotalValue)

	// Get current allocation for comparison
	allocation := portfolio.CalculateAllocation()
	currentAlloc := make(map[models.AssetClass]decimal.Decimal)
	for class, slice := range allocation.ByAssetClass {
		currentAlloc[class] = slice.Percentage
	}

	comparison := scenario.Compare(currentAlloc, portfolio.TotalValue)

	// Return results
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"projections": scenario.Projections,
		"comparison":  comparison,
		"valid":       scenario.IsValid(),
		"total_alloc": scenario.TotalAllocation().InexactFloat64(),
	})
}

// SaveScenario saves a scenario for later reference
func (h *Handler) SaveScenario(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		h.jsonError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var input struct {
		PortfolioID string             `json:"portfolio_id"`
		Name        string             `json:"name"`
		Allocations map[string]float64 `json:"allocations"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.jsonError(w, "Invalid request", http.StatusBadRequest)
		return
	}

	pid, err := uuid.Parse(input.PortfolioID)
	if err != nil {
		h.jsonError(w, "Invalid portfolio ID", http.StatusBadRequest)
		return
	}

	portfolio, err := h.portfolioRepo.GetByID(pid)
	if err != nil || portfolio == nil || portfolio.UserID != user.ID {
		h.jsonError(w, "Portfolio not found", http.StatusNotFound)
		return
	}

	portfolio.CalculateTotals()

	// Create and save scenario
	name := strings.TrimSpace(input.Name)
	if name == "" {
		name = "Saved Scenario"
	}

	scenario := models.NewScenario(pid, name)
	for classStr, pct := range input.Allocations {
		class := models.AssetClass(classStr)
		scenario.SetAllocation(class, decimal.NewFromFloat(pct))
	}

	scenario.CalculateProjections(portfolio.TotalValue)

	if err := h.scenarioRepo.Create(scenario); err != nil {
		h.jsonError(w, "Failed to save scenario", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"id":      scenario.ID.String(),
	})
}

// DeleteScenario removes a saved scenario
func (h *Handler) DeleteScenario(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		h.jsonError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	scenarioID := r.URL.Query().Get("id")
	sid, err := uuid.Parse(scenarioID)
	if err != nil {
		h.jsonError(w, "Invalid scenario ID", http.StatusBadRequest)
		return
	}

	// Delete the scenario
	if err := h.scenarioRepo.Delete(sid); err != nil {
		h.jsonError(w, "Failed to delete", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}
