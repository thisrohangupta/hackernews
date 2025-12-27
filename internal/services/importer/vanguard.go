package importer

import (
	"io"
	"strings"
	"time"

	"github.com/findosh/truenorth/internal/models"
	"github.com/google/uuid"
)

// VanguardParser handles Vanguard CSV exports
type VanguardParser struct{}

// NewVanguardParser creates a new Vanguard parser
func NewVanguardParser() *VanguardParser {
	return &VanguardParser{}
}

// Name returns the parser name
func (p *VanguardParser) Name() string {
	return "vanguard_csv"
}

// Detect checks if this is a Vanguard CSV format
func (p *VanguardParser) Detect(header []string) bool {
	// Vanguard CSVs typically have these columns
	required := []string{"symbol", "investment name", "shares", "share price", "total value"}
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

	// Vanguard-specific patterns
	headerStr := strings.ToLower(strings.Join(header, " "))
	if strings.Contains(headerStr, "investment name") || strings.Contains(headerStr, "vanguard") {
		matches++
	}

	return matches >= 3
}

// Parse reads Vanguard CSV data and returns holdings
func (p *VanguardParser) Parse(reader io.Reader, portfolioID uuid.UUID, accountName string) ([]models.Holding, error) {
	return nil, nil
}

// ParseRow parses a single Vanguard CSV row
func (p *VanguardParser) ParseRow(row []string, header []string, portfolioID uuid.UUID, accountName string) *models.Holding {
	if len(row) < 4 {
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

	ticker := cleanTicker(getCol("symbol", "ticker"))
	if ticker == "" || strings.Contains(strings.ToLower(ticker), "settlement") {
		return nil
	}

	name := cleanName(getCol("investment name", "name", "description"))
	shares := parseDecimal(getCol("shares", "quantity"))
	price := parseDecimal(getCol("share price", "price"))
	totalValue := parseDecimal(getCol("total value", "value", "market value"))

	// Skip if no meaningful data
	if shares.IsZero() && totalValue.IsZero() {
		return nil
	}

	holding := &models.Holding{
		ID:           uuid.New(),
		PortfolioID:  portfolioID,
		AccountName:  accountName,
		Ticker:       ticker,
		Name:         name,
		Quantity:     shares,
		CurrentPrice: price,
		MarketValue:  totalValue,
		AssetClass:   models.AssetClassOther,
		Source:       "vanguard_csv",
		ImportedAt:   time.Now().UTC(),
	}

	if holding.MarketValue.IsZero() && !holding.Quantity.IsZero() && !holding.CurrentPrice.IsZero() {
		holding.CalculateMarketValue()
	}

	return holding
}

// ParseVanguardCSV is a convenience function to parse Vanguard CSV data
func ParseVanguardCSV(records [][]string, portfolioID uuid.UUID, accountName string) []models.Holding {
	if len(records) < 2 {
		return nil
	}

	parser := NewVanguardParser()
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
