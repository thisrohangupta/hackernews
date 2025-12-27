package models

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestNewPortfolio(t *testing.T) {
	userID := uuid.New()
	p := NewPortfolio(userID, "Test Portfolio")

	if p.ID == uuid.Nil {
		t.Error("Expected portfolio ID to be generated")
	}
	if p.UserID != userID {
		t.Errorf("Expected user ID %v, got %v", userID, p.UserID)
	}
	if p.Name != "Test Portfolio" {
		t.Errorf("Expected name 'Test Portfolio', got '%s'", p.Name)
	}
	if !p.TotalValue.IsZero() {
		t.Error("Expected TotalValue to be zero")
	}
}

func TestPortfolio_CalculateTotals(t *testing.T) {
	p := &Portfolio{
		ID:     uuid.New(),
		UserID: uuid.New(),
		Name:   "Test Portfolio",
		Holdings: []Holding{
			{
				Ticker:      "AAPL",
				MarketValue: decimal.NewFromFloat(10000.00),
				AssetClass:  AssetClassEquity,
			},
			{
				Ticker:      "BND",
				MarketValue: decimal.NewFromFloat(5000.00),
				AssetClass:  AssetClassFixedIncome,
			},
			{
				Ticker:      "SPAXX",
				MarketValue: decimal.NewFromFloat(2000.00),
				AssetClass:  AssetClassCash,
			},
		},
	}

	p.CalculateTotals()

	expectedTotal := decimal.NewFromFloat(17000.00)
	expectedCash := decimal.NewFromFloat(2000.00)

	if !p.TotalValue.Equal(expectedTotal) {
		t.Errorf("Expected total value %s, got %s", expectedTotal, p.TotalValue)
	}
	if !p.FreeCash.Equal(expectedCash) {
		t.Errorf("Expected free cash %s, got %s", expectedCash, p.FreeCash)
	}
}

func TestPortfolio_CalculateAllocation(t *testing.T) {
	p := &Portfolio{
		ID:         uuid.New(),
		TotalValue: decimal.NewFromFloat(100000.00),
		Holdings: []Holding{
			{
				Ticker:      "AAPL",
				Name:        "Apple Inc.",
				MarketValue: decimal.NewFromFloat(30000.00),
				AssetClass:  AssetClassEquity,
				Sector:      "Technology",
				Geography:   "US",
			},
			{
				Ticker:      "MSFT",
				Name:        "Microsoft",
				MarketValue: decimal.NewFromFloat(20000.00),
				AssetClass:  AssetClassEquity,
				Sector:      "Technology",
				Geography:   "US",
			},
			{
				Ticker:      "BND",
				Name:        "Vanguard Bond",
				MarketValue: decimal.NewFromFloat(50000.00),
				AssetClass:  AssetClassFixedIncome,
				Sector:      "Bonds",
				Geography:   "US",
			},
		},
	}

	alloc := p.CalculateAllocation()

	// Check asset class allocation
	equitySlice, ok := alloc.ByAssetClass[AssetClassEquity]
	if !ok {
		t.Fatal("Expected equity allocation")
	}
	if !equitySlice.Percentage.Equal(decimal.NewFromInt(50)) {
		t.Errorf("Expected equity percentage 50, got %s", equitySlice.Percentage)
	}

	bondSlice, ok := alloc.ByAssetClass[AssetClassFixedIncome]
	if !ok {
		t.Fatal("Expected fixed income allocation")
	}
	if !bondSlice.Percentage.Equal(decimal.NewFromInt(50)) {
		t.Errorf("Expected fixed income percentage 50, got %s", bondSlice.Percentage)
	}

	// Check sector allocation
	techSlice, ok := alloc.BySector["Technology"]
	if !ok {
		t.Fatal("Expected Technology sector")
	}
	if !techSlice.Percentage.Equal(decimal.NewFromInt(50)) {
		t.Errorf("Expected Technology percentage 50, got %s", techSlice.Percentage)
	}

	// Check top holdings
	if len(alloc.TopHoldings) != 3 {
		t.Errorf("Expected 3 top holdings, got %d", len(alloc.TopHoldings))
	}

	// First should be BND (highest value)
	if alloc.TopHoldings[0].Ticker != "BND" {
		t.Errorf("Expected BND as top holding, got %s", alloc.TopHoldings[0].Ticker)
	}
}

func TestPortfolio_CalculateAllocation_Empty(t *testing.T) {
	p := &Portfolio{
		ID:         uuid.New(),
		TotalValue: decimal.Zero,
		Holdings:   []Holding{},
	}

	alloc := p.CalculateAllocation()

	if len(alloc.ByAssetClass) != 0 {
		t.Error("Expected empty asset class allocation for empty portfolio")
	}
	if len(alloc.TopHoldings) != 0 {
		t.Error("Expected empty top holdings for empty portfolio")
	}
}

func TestPortfolio_getTopHoldings(t *testing.T) {
	p := &Portfolio{
		ID:         uuid.New(),
		TotalValue: decimal.NewFromFloat(100.00),
		Holdings: []Holding{
			{Ticker: "A", MarketValue: decimal.NewFromFloat(10.00)},
			{Ticker: "B", MarketValue: decimal.NewFromFloat(30.00)},
			{Ticker: "C", MarketValue: decimal.NewFromFloat(20.00)},
			{Ticker: "D", MarketValue: decimal.NewFromFloat(40.00)},
		},
	}

	top := p.getTopHoldings(3)

	if len(top) != 3 {
		t.Fatalf("Expected 3 holdings, got %d", len(top))
	}

	// Should be sorted by value descending: D, B, C
	expected := []string{"D", "B", "C"}
	for i, h := range top {
		if h.Ticker != expected[i] {
			t.Errorf("Position %d: expected %s, got %s", i, expected[i], h.Ticker)
		}
	}
}

func TestPortfolio_AggregatesSameTicker(t *testing.T) {
	p := &Portfolio{
		ID:         uuid.New(),
		TotalValue: decimal.NewFromFloat(30000.00),
		Holdings: []Holding{
			{Ticker: "AAPL", AccountName: "IRA", MarketValue: decimal.NewFromFloat(10000.00)},
			{Ticker: "AAPL", AccountName: "401k", MarketValue: decimal.NewFromFloat(20000.00)},
		},
	}

	alloc := p.CalculateAllocation()

	// Should aggregate AAPL from both accounts
	aaplTotal, ok := alloc.TickerTotals["AAPL"]
	if !ok {
		t.Fatal("Expected AAPL in ticker totals")
	}

	expected := decimal.NewFromFloat(30000.00)
	if !aaplTotal.Equal(expected) {
		t.Errorf("Expected AAPL total %s, got %s", expected, aaplTotal)
	}

	// Top holdings should show aggregated AAPL
	if len(alloc.TopHoldings) != 1 {
		t.Errorf("Expected 1 aggregated holding, got %d", len(alloc.TopHoldings))
	}
}
