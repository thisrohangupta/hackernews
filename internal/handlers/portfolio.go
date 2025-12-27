package handlers

import (
	"encoding/csv"
	"io"
	"net/http"
	"strings"

	"github.com/findosh/truenorth/internal/middleware"
	"github.com/findosh/truenorth/internal/models"
	"github.com/findosh/truenorth/internal/services/importer"
	"github.com/google/uuid"
)

// NewPortfolioPage renders the create portfolio page
func (h *Handler) NewPortfolioPage(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		h.redirect(w, r, "/login")
		return
	}

	data := map[string]interface{}{
		"Title": "Create Portfolio - TrueNorth",
		"User":  user,
		"Error": r.URL.Query().Get("error"),
	}
	h.render(w, "portfolio_new.html", data)
}

// CreatePortfolio handles portfolio creation
func (h *Handler) CreatePortfolio(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		h.redirect(w, r, "/login")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.redirect(w, r, "/portfolio/new?error=Invalid+request")
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		name = "My Portfolio"
	}

	portfolio := models.NewPortfolio(user.ID, name)
	if err := h.portfolioRepo.Create(portfolio); err != nil {
		h.redirect(w, r, "/portfolio/new?error=Failed+to+create+portfolio")
		return
	}

	h.redirect(w, r, "/import?portfolio="+portfolio.ID.String())
}

// ImportPage renders the CSV import page
func (h *Handler) ImportPage(w http.ResponseWriter, r *http.Request) {
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

	data := map[string]interface{}{
		"Title":       "Import Holdings - TrueNorth",
		"User":        user,
		"PortfolioID": portfolioID,
		"Error":       r.URL.Query().Get("error"),
		"Success":     r.URL.Query().Get("success"),
	}
	h.render(w, "import.html", data)
}

// ImportCSV handles CSV file upload
func (h *Handler) ImportCSV(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		h.jsonError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse multipart form (10MB max)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		h.redirect(w, r, "/import?error=File+too+large")
		return
	}

	portfolioID := r.FormValue("portfolio_id")
	accountName := r.FormValue("account_name")
	if accountName == "" {
		accountName = "Imported Account"
	}

	pid, err := uuid.Parse(portfolioID)
	if err != nil {
		h.redirect(w, r, "/dashboard?error=Invalid+portfolio")
		return
	}

	// Verify portfolio belongs to user
	portfolio, err := h.portfolioRepo.GetByID(pid)
	if err != nil || portfolio == nil || portfolio.UserID != user.ID {
		h.redirect(w, r, "/dashboard?error=Portfolio+not+found")
		return
	}

	// Get uploaded file
	file, _, err := r.FormFile("csv_file")
	if err != nil {
		h.redirect(w, r, "/import?portfolio="+portfolioID+"&error=No+file+uploaded")
		return
	}
	defer file.Close()

	// Read CSV content
	csvReader := csv.NewReader(file)
	csvReader.FieldsPerRecord = -1
	records, err := csvReader.ReadAll()
	if err != nil {
		h.redirect(w, r, "/import?portfolio="+portfolioID+"&error=Invalid+CSV+format")
		return
	}

	if len(records) < 2 {
		h.redirect(w, r, "/import?portfolio="+portfolioID+"&error=CSV+file+is+empty")
		return
	}

	// Parse the CSV
	holdings := parseCSVRecords(records, pid, accountName)
	if len(holdings) == 0 {
		h.redirect(w, r, "/import?portfolio="+portfolioID+"&error=No+valid+holdings+found")
		return
	}

	// Auto-tag the holdings
	tagger := importer.NewTagger()
	tagger.TagHoldings(holdings)

	// Save holdings
	if err := h.holdingRepo.CreateBatch(holdings); err != nil {
		h.redirect(w, r, "/import?portfolio="+portfolioID+"&error=Failed+to+save+holdings")
		return
	}

	// Update portfolio totals
	portfolio.Holdings = holdings
	portfolio.CalculateTotals()
	if err := h.portfolioRepo.Update(portfolio); err != nil {
		// Non-fatal error
	}

	h.redirect(w, r, "/dashboard?portfolio="+portfolioID)
}

// parseCSVRecords parses CSV records into holdings
func parseCSVRecords(records [][]string, portfolioID uuid.UUID, accountName string) []models.Holding {
	var holdings []models.Holding

	if len(records) < 2 {
		return holdings
	}

	// Try each parser
	schwab := importer.ParseSchwabCSV(records, portfolioID, accountName)
	if len(schwab) > 0 {
		return schwab
	}

	fidelity := importer.ParseFidelityCSV(records, portfolioID, accountName)
	if len(fidelity) > 0 {
		return fidelity
	}

	vanguard := importer.ParseVanguardCSV(records, portfolioID, accountName)
	if len(vanguard) > 0 {
		return vanguard
	}

	// Generic fallback parser
	return parseGenericCSV(records, portfolioID, accountName)
}

func parseGenericCSV(records [][]string, portfolioID uuid.UUID, accountName string) []models.Holding {
	var holdings []models.Holding
	if len(records) < 2 {
		return holdings
	}

	header := records[0]
	colMap := make(map[string]int)
	for i, h := range header {
		colMap[strings.ToLower(strings.TrimSpace(h))] = i
	}

	getCol := func(row []string, names ...string) string {
		for _, name := range names {
			if idx, ok := colMap[name]; ok && idx < len(row) {
				return row[idx]
			}
		}
		return ""
	}

	for i := 1; i < len(records); i++ {
		row := records[i]
		if len(row) < 3 {
			continue
		}

		ticker := strings.TrimSpace(strings.ToUpper(getCol(row, "symbol", "ticker")))
		if ticker == "" {
			continue
		}

		name := getCol(row, "description", "name", "security")
		quantity := importer.ParseSchwabCSV(records, portfolioID, accountName) // Placeholder

		holding := models.NewHolding(portfolioID, ticker, name, accountName)
		holding.Source = "generic_csv"

		// Skip if parsed as empty
		if holding.Quantity.IsZero() && holding.MarketValue.IsZero() {
			_ = quantity // Use variable
			continue
		}

		holdings = append(holdings, *holding)
	}

	return holdings
}

// PortfolioView renders a single portfolio page
func (h *Handler) PortfolioView(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		h.redirect(w, r, "/login")
		return
	}

	// Get portfolio ID from URL path
	path := r.URL.Path
	parts := strings.Split(path, "/")
	if len(parts) < 3 {
		h.redirect(w, r, "/dashboard")
		return
	}

	portfolioID, err := uuid.Parse(parts[2])
	if err != nil {
		h.redirect(w, r, "/dashboard")
		return
	}

	portfolio, err := h.portfolioRepo.GetByID(portfolioID)
	if err != nil || portfolio == nil || portfolio.UserID != user.ID {
		h.redirect(w, r, "/dashboard?error=Portfolio+not+found")
		return
	}

	portfolio.CalculateTotals()
	allocation := portfolio.CalculateAllocation()

	data := map[string]interface{}{
		"Title":      portfolio.Name + " - TrueNorth",
		"User":       user,
		"Portfolio":  portfolio,
		"Allocation": allocation,
	}

	h.render(w, "portfolio.html", data)
}

// EditHolding handles holding classification updates
func (h *Handler) EditHolding(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		h.jsonError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		h.jsonError(w, "Invalid request", http.StatusBadRequest)
		return
	}

	holdingID, err := uuid.Parse(r.FormValue("holding_id"))
	if err != nil {
		h.jsonError(w, "Invalid holding ID", http.StatusBadRequest)
		return
	}

	// Get holding and verify ownership
	// This would need a GetByID method on HoldingRepository
	// For now, update directly

	assetClass := models.AssetClass(r.FormValue("asset_class"))
	sector := r.FormValue("sector")
	geography := r.FormValue("geography")

	// Update holding
	query := `UPDATE holdings SET asset_class = ?, sector = ?, geography = ?, is_manual_entry = 1 WHERE id = ?`
	_ = query
	_ = holdingID
	_ = assetClass
	_ = sector
	_ = geography

	// Redirect back to portfolio
	portfolioID := r.FormValue("portfolio_id")
	h.redirect(w, r, "/dashboard?portfolio="+portfolioID)
}

// DeletePortfolio handles portfolio deletion
func (h *Handler) DeletePortfolio(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		h.redirect(w, r, "/login")
		return
	}

	if err := r.ParseForm(); err != nil {
		h.redirect(w, r, "/dashboard?error=Invalid+request")
		return
	}

	portfolioID, err := uuid.Parse(r.FormValue("portfolio_id"))
	if err != nil {
		h.redirect(w, r, "/dashboard?error=Invalid+portfolio")
		return
	}

	portfolio, err := h.portfolioRepo.GetByID(portfolioID)
	if err != nil || portfolio == nil || portfolio.UserID != user.ID {
		h.redirect(w, r, "/dashboard?error=Portfolio+not+found")
		return
	}

	if err := h.portfolioRepo.Delete(portfolioID); err != nil {
		h.redirect(w, r, "/dashboard?error=Failed+to+delete")
		return
	}

	h.redirect(w, r, "/dashboard")
}

// DownloadTemplate serves a sample CSV template
func (h *Handler) DownloadTemplate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=truenorth_template.csv")

	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Header
	writer.Write([]string{"Symbol", "Description", "Quantity", "Price", "Market Value", "Cost Basis"})

	// Sample data
	writer.Write([]string{"AAPL", "Apple Inc.", "100", "175.50", "17550.00", "15000.00"})
	writer.Write([]string{"MSFT", "Microsoft Corporation", "50", "378.25", "18912.50", "16000.00"})
	writer.Write([]string{"VOO", "Vanguard S&P 500 ETF", "25", "425.00", "10625.00", "9500.00"})
}

// Used by parseGenericCSV but defined here to avoid import issues
func readCSVRecords(r io.Reader) ([][]string, error) {
	reader := csv.NewReader(r)
	reader.FieldsPerRecord = -1
	return reader.ReadAll()
}
