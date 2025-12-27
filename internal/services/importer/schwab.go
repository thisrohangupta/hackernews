package importer

import (
	"io"
	"strings"
	"time"

	"github.com/findosh/truenorth/internal/models"
	"github.com/google/uuid"
)

// SchwabParser handles Charles Schwab CSV exports
type SchwabParser struct {
	columnMap map[string]int
}

// NewSchwabParser creates a new Schwab parser
func NewSchwabParser() *SchwabParser {
	return &SchwabParser{}
}

// Name returns the parser name
func (p *SchwabParser) Name() string {
	return "schwab_csv"
}

// Detect checks if this is a Schwab CSV format
func (p *SchwabParser) Detect(header []string) bool {
	// Schwab CSVs typically have these columns
	required := []string{"symbol", "description", "quantity", "price", "market value"}
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

	return matches >= 3
}

// Parse reads Schwab CSV data and returns holdings
func (p *SchwabParser) Parse(reader io.Reader, portfolioID uuid.UUID, accountName string) ([]models.Holding, error) {
	// This is called via the service's ParseCSV which handles reading
	return nil, nil
}

// ParseRow parses a single Schwab CSV row
func (p *SchwabParser) ParseRow(row []string, header []string, portfolioID uuid.UUID, accountName string) *models.Holding {
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
	if ticker == "" || ticker == "CASH" || strings.HasPrefix(ticker, "--") {
		return nil
	}

	name := cleanName(getCol("description", "security description"))
	quantity := parseDecimal(getCol("quantity", "shares"))
	price := parseDecimal(getCol("price", "last price"))
	marketValue := parseDecimal(getCol("market value", "value"))
	costBasis := parseDecimal(getCol("cost basis", "cost basis total"))

	// Skip if no meaningful data
	if quantity.IsZero() && marketValue.IsZero() {
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
		MarketValue:  marketValue,
		AssetClass:   models.AssetClassOther,
		Source:       "schwab_csv",
		ImportedAt:   time.Now().UTC(),
	}

	// Calculate market value if not provided
	if holding.MarketValue.IsZero() && !holding.Quantity.IsZero() && !holding.CurrentPrice.IsZero() {
		holding.CalculateMarketValue()
	}

	return holding
}

// ParseSchwabCSV is a convenience function to parse Schwab CSV data
func ParseSchwabCSV(records [][]string, portfolioID uuid.UUID, accountName string) []models.Holding {
	if len(records) < 2 {
		return nil
	}

	parser := NewSchwabParser()
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
