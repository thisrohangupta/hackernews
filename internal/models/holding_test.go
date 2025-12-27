package models

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestNewHolding(t *testing.T) {
	portfolioID := uuid.New()
	h := NewHolding(portfolioID, "AAPL", "Apple Inc.", "Schwab IRA")

	if h.ID == uuid.Nil {
		t.Error("Expected holding ID to be generated")
	}
	if h.PortfolioID != portfolioID {
		t.Errorf("Expected portfolio ID %v, got %v", portfolioID, h.PortfolioID)
	}
	if h.Ticker != "AAPL" {
		t.Errorf("Expected ticker AAPL, got %s", h.Ticker)
	}
	if h.AssetClass != AssetClassOther {
		t.Errorf("Expected asset class Other, got %s", h.AssetClass)
	}
}

func TestHolding_CalculateMarketValue(t *testing.T) {
	h := &Holding{
		Quantity:     decimal.NewFromInt(100),
		CurrentPrice: decimal.NewFromFloat(175.50),
	}

	h.CalculateMarketValue()

	expected := decimal.NewFromFloat(17550.00)
	if !h.MarketValue.Equal(expected) {
		t.Errorf("Expected market value %s, got %s", expected, h.MarketValue)
	}
}

func TestHolding_GainLoss(t *testing.T) {
	h := &Holding{
		MarketValue: decimal.NewFromFloat(17550.00),
		CostBasis:   decimal.NewFromFloat(15000.00),
	}

	gain := h.GainLoss()
	expected := decimal.NewFromFloat(2550.00)

	if !gain.Equal(expected) {
		t.Errorf("Expected gain %s, got %s", expected, gain)
	}
}

func TestHolding_GainLossPercent(t *testing.T) {
	h := &Holding{
		MarketValue: decimal.NewFromFloat(17550.00),
		CostBasis:   decimal.NewFromFloat(15000.00),
	}

	pct := h.GainLossPercent()
	expected := decimal.NewFromFloat(17.00)

	if !pct.Equal(expected) {
		t.Errorf("Expected gain percent %s, got %s", expected, pct)
	}
}

func TestHolding_GainLossPercent_ZeroCostBasis(t *testing.T) {
	h := &Holding{
		MarketValue: decimal.NewFromFloat(1000.00),
		CostBasis:   decimal.Zero,
	}

	pct := h.GainLossPercent()

	if !pct.IsZero() {
		t.Errorf("Expected zero percent for zero cost basis, got %s", pct)
	}
}

func TestHolding_IsCash(t *testing.T) {
	tests := []struct {
		assetClass AssetClass
		expected   bool
	}{
		{AssetClassCash, true},
		{AssetClassEquity, false},
		{AssetClassFixedIncome, false},
	}

	for _, tt := range tests {
		h := &Holding{AssetClass: tt.assetClass}
		if h.IsCash() != tt.expected {
			t.Errorf("IsCash() for %s: expected %v, got %v", tt.assetClass, tt.expected, h.IsCash())
		}
	}
}

func TestHolding_NeedsClassification(t *testing.T) {
	tests := []struct {
		assetClass AssetClass
		expected   bool
	}{
		{AssetClassOther, true},
		{AssetClassEquity, false},
		{AssetClassCash, false},
	}

	for _, tt := range tests {
		h := &Holding{AssetClass: tt.assetClass}
		if h.NeedsClassification() != tt.expected {
			t.Errorf("NeedsClassification() for %s: expected %v, got %v",
				tt.assetClass, tt.expected, h.NeedsClassification())
		}
	}
}

func TestAssetClass_DisplayName(t *testing.T) {
	tests := []struct {
		class    AssetClass
		expected string
	}{
		{AssetClassEquity, "Equities"},
		{AssetClassFixedIncome, "Fixed Income"},
		{AssetClassAlternative, "Alternatives"},
		{AssetClassCrypto, "Cryptocurrency"},
		{AssetClassCash, "Cash"},
		{AssetClassOther, "Other"},
	}

	for _, tt := range tests {
		if tt.class.DisplayName() != tt.expected {
			t.Errorf("DisplayName() for %s: expected %s, got %s",
				tt.class, tt.expected, tt.class.DisplayName())
		}
	}
}

func TestAllAssetClasses(t *testing.T) {
	classes := AllAssetClasses()
	if len(classes) != 6 {
		t.Errorf("Expected 6 asset classes, got %d", len(classes))
	}
}
