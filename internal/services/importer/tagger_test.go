package importer

import (
	"testing"

	"github.com/findosh/truenorth/internal/models"
	"github.com/google/uuid"
)

func TestNewTagger(t *testing.T) {
	tagger := NewTagger()

	if tagger == nil {
		t.Fatal("Expected tagger to be created")
	}
	if len(tagger.tickerDB) == 0 {
		t.Error("Expected ticker database to be populated")
	}
}

func TestTagger_TagHolding_KnownTicker(t *testing.T) {
	tagger := NewTagger()

	tests := []struct {
		ticker     string
		wantClass  models.AssetClass
		wantSector string
		wantGeo    string
	}{
		{"AAPL", models.AssetClassEquity, "Technology", "US"},
		{"MSFT", models.AssetClassEquity, "Technology", "US"},
		{"JPM", models.AssetClassEquity, "Financial Services", "US"},
		{"VOO", models.AssetClassEquity, "Diversified", "US"},
		{"BND", models.AssetClassFixedIncome, "Bonds", "US"},
		{"VNQ", models.AssetClassAlternative, "Real Estate", "US"},
		{"SPAXX", models.AssetClassCash, "Cash", "US"},
		{"VWO", models.AssetClassEquity, "Diversified", "Emerging Markets"},
	}

	for _, tt := range tests {
		t.Run(tt.ticker, func(t *testing.T) {
			h := &models.Holding{
				ID:         uuid.New(),
				Ticker:     tt.ticker,
				AssetClass: models.AssetClassOther,
			}

			tagger.TagHolding(h)

			if h.AssetClass != tt.wantClass {
				t.Errorf("Asset class: got %s, want %s", h.AssetClass, tt.wantClass)
			}
			if h.Sector != tt.wantSector {
				t.Errorf("Sector: got %s, want %s", h.Sector, tt.wantSector)
			}
			if h.Geography != tt.wantGeo {
				t.Errorf("Geography: got %s, want %s", h.Geography, tt.wantGeo)
			}
		})
	}
}

func TestTagger_TagHolding_Heuristics(t *testing.T) {
	tagger := NewTagger()

	tests := []struct {
		name      string
		ticker    string
		holdName  string
		wantClass models.AssetClass
	}{
		{
			name:      "Money market fund",
			ticker:    "XXXX",
			holdName:  "XYZ Money Market Fund",
			wantClass: models.AssetClassCash,
		},
		{
			name:      "Bond fund by name",
			ticker:    "ABCD",
			holdName:  "ABC Corporate Bond Fund",
			wantClass: models.AssetClassFixedIncome,
		},
		{
			name:      "Bitcoin trust",
			ticker:    "BTCX",
			holdName:  "Bitcoin Investment Trust",
			wantClass: models.AssetClassCrypto,
		},
		{
			name:      "Real estate fund",
			ticker:    "REIT",
			holdName:  "ABC Real Estate Investment Trust",
			wantClass: models.AssetClassAlternative,
		},
		{
			name:      "Unknown short ticker - assume equity",
			ticker:    "XYZ",
			holdName:  "XYZ Corporation",
			wantClass: models.AssetClassEquity,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &models.Holding{
				ID:         uuid.New(),
				Ticker:     tt.ticker,
				Name:       tt.holdName,
				AssetClass: models.AssetClassOther,
			}

			tagger.TagHolding(h)

			if h.AssetClass != tt.wantClass {
				t.Errorf("Asset class: got %s, want %s", h.AssetClass, tt.wantClass)
			}
		})
	}
}

func TestTagger_TagHoldings_Batch(t *testing.T) {
	tagger := NewTagger()

	holdings := []models.Holding{
		{ID: uuid.New(), Ticker: "AAPL", Name: "Apple Inc.", AssetClass: models.AssetClassOther},
		{ID: uuid.New(), Ticker: "BND", Name: "Vanguard Bond", AssetClass: models.AssetClassOther},
		{ID: uuid.New(), Ticker: "SPAXX", Name: "Fidelity Money Market", AssetClass: models.AssetClassOther},
	}

	tagger.TagHoldings(holdings)

	if holdings[0].AssetClass != models.AssetClassEquity {
		t.Errorf("AAPL: expected Equity, got %s", holdings[0].AssetClass)
	}
	if holdings[1].AssetClass != models.AssetClassFixedIncome {
		t.Errorf("BND: expected FixedIncome, got %s", holdings[1].AssetClass)
	}
	if holdings[2].AssetClass != models.AssetClassCash {
		t.Errorf("SPAXX: expected Cash, got %s", holdings[2].AssetClass)
	}
}

func TestTagger_DetectSector(t *testing.T) {
	tagger := NewTagger()

	tests := []struct {
		name       string
		holdName   string
		wantSector string
	}{
		{"Tech company", "Apple Software Corp", "Technology"},
		{"Healthcare", "ABC Pharmaceuticals", "Healthcare"},
		{"Bank", "First National Bank Corp", "Financial Services"},
		{"Energy", "XYZ Oil & Gas", "Energy"},
		{"Unknown", "Random Company Inc", "Diversified"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sector := tagger.detectSector("XXX", tt.holdName)
			if sector != tt.wantSector {
				t.Errorf("Sector: got %s, want %s", sector, tt.wantSector)
			}
		})
	}
}

func TestTagger_DetectGeography(t *testing.T) {
	tagger := NewTagger()

	tests := []struct {
		name    string
		holdName string
		wantGeo string
	}{
		{"International fund", "Vanguard International Stock Fund", "International Developed"},
		{"Emerging markets", "iShares Emerging Markets ETF", "Emerging Markets"},
		{"Global fund", "Global Equity Fund", "Global"},
		{"Default to US", "S&P 500 Index Fund", "US"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			geo := tagger.detectGeography(tt.holdName)
			if geo != tt.wantGeo {
				t.Errorf("Geography: got %s, want %s", geo, tt.wantGeo)
			}
		})
	}
}

func TestParseDecimal(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"123.45", 123.45},
		{"$1,234.56", 1234.56},
		{"(100.00)", -100.00},
		{"1,000,000", 1000000},
		{"--", 0},
		{"n/a", 0},
		{"", 0},
		{"  $500.00  ", 500.00},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseDecimal(tt.input)
			if result.InexactFloat64() != tt.expected {
				t.Errorf("parseDecimal(%q) = %v, want %v", tt.input, result.InexactFloat64(), tt.expected)
			}
		})
	}
}

func TestCleanTicker(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"AAPL", "AAPL"},
		{"  aapl  ", "AAPL"},
		{"MSFT*", "MSFT"},
		{"googl**", "GOOGL"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := cleanTicker(tt.input)
			if result != tt.expected {
				t.Errorf("cleanTicker(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
