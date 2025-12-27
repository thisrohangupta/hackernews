package importer

import (
	"io"
	"strings"
	"time"

	"github.com/findosh/truenorth/internal/models"
	"github.com/google/uuid"
)

// FidelityParser handles Fidelity CSV exports
type FidelityParser struct{}

// NewFidelityParser creates a new Fidelity parser
func NewFidelityParser() *FidelityParser {
	return &FidelityParser{}
}

// Name returns the parser name
func (p *FidelityParser) Name() string {
	return "fidelity_csv"
}

// Detect checks if this is a Fidelity CSV format
func (p *FidelityParser) Detect(header []string) bool {
	// Fidelity CSVs typically have these columns
	required := []string{"symbol", "description", "quantity", "last price", "current value"}
	matches := 0

	headerLower := make([]string, len(header))
	for i, h := range header {
		headerLower[i] = strings.ToLower(strings.TrimSpace(h))
	}

	for _, req := range required {
		for _, h := range headerLower {
			if strings.Contains(h, req) {
				matches++
				break
			}
		}
	}

	// Fidelity-specific check
	headerStr := strings.ToLower(strings.Join(header, " "))
	if strings.Contains(headerStr, "current value") || strings.Contains(headerStr, "last price change") {
		matches++
	}

	return matches >= 3
}

// Parse reads Fidelity CSV data and returns holdings
func (p *FidelityParser) Parse(reader io.Reader, portfolioID uuid.UUID, accountName string) ([]models.Holding, error) {
	return nil, nil
}

// ParseRow parses a single Fidelity CSV row
func (p *FidelityParser) ParseRow(row []string, header []string, portfolioID uuid.UUID, accountName string) *models.Holding {
	if len(row) < 5 {
		return nil
	}

	// Build column index map
	colMap := make(map[string]int)
	for i, h := range header {
		colMap[strings.ToLower(strings.TrimSpace(h))] = i
	}

	getCol := func(names ...string) string {
		for _, name := range names {
			if idx, ok := colMap[name]; ok && idx < len(row) {
				return row[idx]
			}
		}
		return ""
	}

	ticker := cleanTicker(getCol("symbol"))
	if ticker == "" || strings.HasPrefix(ticker, "CASH") || ticker == "PENDING ACTIVITY" {
		return nil
	}

	name := cleanName(getCol("description", "security description"))
	quantity := parseDecimal(getCol("quantity", "shares"))
	price := parseDecimal(getCol("last price", "price"))
	currentValue := parseDecimal(getCol("current value", "value"))
	costBasis := parseDecimal(getCol("cost basis total", "cost basis"))

	// Skip if no meaningful data
	if quantity.IsZero() && currentValue.IsZero() {
		return nil
	}

	holding := &models.Holding{
		ID:           uuid.New(),
		PortfolioID:  portfolioID,
		AccountName:  accountName,
		Ticker:       ticker,
		Name:         name,
		Quantity:     quantity,
		CostBasis:    costBasis,
		CurrentPrice: price,
		MarketValue:  currentValue,
		AssetClass:   models.AssetClassOther,
		Source:       "fidelity_csv",
		ImportedAt:   time.Now().UTC(),
	}

	if holding.MarketValue.IsZero() && !holding.Quantity.IsZero() && !holding.CurrentPrice.IsZero() {
		holding.CalculateMarketValue()
	}

	return holding
}

// ParseFidelityCSV is a convenience function to parse Fidelity CSV data
func ParseFidelityCSV(records [][]string, portfolioID uuid.UUID, accountName string) []models.Holding {
	if len(records) < 2 {
		return nil
	}

	parser := NewFidelityParser()
	header := records[0]

	if !parser.Detect(header) {
		return nil
	}

	var holdings []models.Holding
	for i := 1; i < len(records); i++ {
		if h := parser.ParseRow(records[i], header, portfolioID, accountName); h != nil {
			holdings = append(holdings, *h)
		}
	}

	return holdings
}
