package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Portfolio represents a unified view across all accounts (OmniFolio)
type Portfolio struct {
	ID          uuid.UUID       `json:"id"`
	UserID      uuid.UUID       `json:"user_id"`
	Name        string          `json:"name"` // e.g., "Family Portfolio"
	Holdings    []Holding       `json:"holdings,omitempty"`
	TotalValue  decimal.Decimal `json:"total_value"`
	FreeCash    decimal.Decimal `json:"free_cash"`
	LastUpdated time.Time       `json:"last_updated"`
	CreatedAt   time.Time       `json:"created_at"`
}

// NewPortfolio creates a new portfolio with generated ID
func NewPortfolio(userID uuid.UUID, name string) *Portfolio {
	now := time.Now().UTC()
	return &Portfolio{
		ID:          uuid.New(),
		UserID:      userID,
		Name:        name,
		Holdings:    []Holding{},
		TotalValue:  decimal.Zero,
		FreeCash:    decimal.Zero,
		LastUpdated: now,
		CreatedAt:   now,
	}
}

// CalculateTotals recalculates TotalValue and FreeCash from holdings
func (p *Portfolio) CalculateTotals() {
	total := decimal.Zero
	cash := decimal.Zero

	for _, h := range p.Holdings {
		total = total.Add(h.MarketValue)
		if h.AssetClass == AssetClassCash {
			cash = cash.Add(h.MarketValue)
		}
	}

	p.TotalValue = total
	p.FreeCash = cash
	p.LastUpdated = time.Now().UTC()
}

// AllocationSummary provides portfolio breakdown by various dimensions
type AllocationSummary struct {
	ByAssetClass map[AssetClass]AllocationSlice `json:"by_asset_class"`
	BySector     map[string]AllocationSlice     `json:"by_sector"`
	ByGeography  map[string]AllocationSlice     `json:"by_geography"`
	ByAccount    map[string]AllocationSlice     `json:"by_account"`
	TopHoldings  []HoldingSummary               `json:"top_holdings"`
	TickerTotals map[string]decimal.Decimal     `json:"ticker_totals"`
}

// AllocationSlice represents a portion of the portfolio
type AllocationSlice struct {
	Value      decimal.Decimal `json:"value"`
	Percentage decimal.Decimal `json:"percentage"`
	Count      int             `json:"count"`
}

// HoldingSummary is a simplified view of a holding for top 10 display
type HoldingSummary struct {
	Ticker      string          `json:"ticker"`
	Name        string          `json:"name"`
	MarketValue decimal.Decimal `json:"market_value"`
	Percentage  decimal.Decimal `json:"percentage"`
	AssetClass  AssetClass      `json:"asset_class"`
}

// CalculateAllocation computes the full allocation breakdown
func (p *Portfolio) CalculateAllocation() *AllocationSummary {
	summary := &AllocationSummary{
		ByAssetClass: make(map[AssetClass]AllocationSlice),
		BySector:     make(map[string]AllocationSlice),
		ByGeography:  make(map[string]AllocationSlice),
		ByAccount:    make(map[string]AllocationSlice),
		TopHoldings:  []HoldingSummary{},
		TickerTotals: make(map[string]decimal.Decimal),
	}

	if p.TotalValue.IsZero() {
		return summary
	}

	// Aggregate by dimensions
	for _, h := range p.Holdings {
		// By asset class
		slice := summary.ByAssetClass[h.AssetClass]
		slice.Value = slice.Value.Add(h.MarketValue)
		slice.Count++
		summary.ByAssetClass[h.AssetClass] = slice

		// By sector
		if h.Sector != "" {
			slice := summary.BySector[h.Sector]
			slice.Value = slice.Value.Add(h.MarketValue)
			slice.Count++
			summary.BySector[h.Sector] = slice
		}

		// By geography
		if h.Geography != "" {
			slice := summary.ByGeography[h.Geography]
			slice.Value = slice.Value.Add(h.MarketValue)
			slice.Count++
			summary.ByGeography[h.Geography] = slice
		}

		// By account
		slice = summary.ByAccount[h.AccountName]
		slice.Value = slice.Value.Add(h.MarketValue)
		slice.Count++
		summary.ByAccount[h.AccountName] = slice

		// Ticker totals (aggregate same ticker across accounts)
		summary.TickerTotals[h.Ticker] = summary.TickerTotals[h.Ticker].Add(h.MarketValue)
	}

	// Calculate percentages
	hundred := decimal.NewFromInt(100)
	for class, slice := range summary.ByAssetClass {
		slice.Percentage = slice.Value.Div(p.TotalValue).Mul(hundred).Round(2)
		summary.ByAssetClass[class] = slice
	}
	for sector, slice := range summary.BySector {
		slice.Percentage = slice.Value.Div(p.TotalValue).Mul(hundred).Round(2)
		summary.BySector[sector] = slice
	}
	for geo, slice := range summary.ByGeography {
		slice.Percentage = slice.Value.Div(p.TotalValue).Mul(hundred).Round(2)
		summary.ByGeography[geo] = slice
	}
	for acct, slice := range summary.ByAccount {
		slice.Percentage = slice.Value.Div(p.TotalValue).Mul(hundred).Round(2)
		summary.ByAccount[acct] = slice
	}

	// Build top holdings (sorted by value, top 10)
	summary.TopHoldings = p.getTopHoldings(10)

	return summary
}

// getTopHoldings returns the top N holdings by market value
func (p *Portfolio) getTopHoldings(n int) []HoldingSummary {
	// Aggregate by ticker first
	tickerHoldings := make(map[string]*HoldingSummary)
	for _, h := range p.Holdings {
		if existing, ok := tickerHoldings[h.Ticker]; ok {
			existing.MarketValue = existing.MarketValue.Add(h.MarketValue)
		} else {
			tickerHoldings[h.Ticker] = &HoldingSummary{
				Ticker:      h.Ticker,
				Name:        h.Name,
				MarketValue: h.MarketValue,
				AssetClass:  h.AssetClass,
			}
		}
	}

	// Convert to slice and sort
	holdings := make([]HoldingSummary, 0, len(tickerHoldings))
	for _, h := range tickerHoldings {
		if !p.TotalValue.IsZero() {
			h.Percentage = h.MarketValue.Div(p.TotalValue).Mul(decimal.NewFromInt(100)).Round(2)
		}
		holdings = append(holdings, *h)
	}

	// Simple bubble sort (sufficient for small n)
	for i := 0; i < len(holdings); i++ {
		for j := i + 1; j < len(holdings); j++ {
			if holdings[j].MarketValue.GreaterThan(holdings[i].MarketValue) {
				holdings[i], holdings[j] = holdings[j], holdings[i]
			}
		}
	}

	if len(holdings) > n {
		holdings = holdings[:n]
	}
	return holdings
}
