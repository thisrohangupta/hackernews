package models

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestGetExpenseRatio_KnownTicker(t *testing.T) {
	tests := []struct {
		ticker   string
		expected float64
	}{
		{"VOO", 0.03},
		{"VTI", 0.03},
		{"SPY", 0.09},
		{"QQQ", 0.20},
		{"GBTC", 1.50},
		{"FZROX", 0.00}, // Fidelity Zero
	}

	for _, tt := range tests {
		t.Run(tt.ticker, func(t *testing.T) {
			ratio := GetExpenseRatio(tt.ticker, AssetClassEquity)
			expected := decimal.NewFromFloat(tt.expected)
			if !ratio.Equal(expected) {
				t.Errorf("GetExpenseRatio(%s) = %s, want %s", tt.ticker, ratio.String(), expected.String())
			}
		})
	}
}

func TestGetExpenseRatio_UnknownStock(t *testing.T) {
	// Short tickers (likely stocks) should return 0
	ratio := GetExpenseRatio("AAPL", AssetClassEquity)
	if !ratio.IsZero() {
		t.Errorf("Individual stock should have 0 expense ratio, got %s", ratio.String())
	}

	ratio = GetExpenseRatio("MSFT", AssetClassEquity)
	if !ratio.IsZero() {
		t.Errorf("Individual stock should have 0 expense ratio, got %s", ratio.String())
	}
}

func TestGetExpenseRatio_DefaultByAssetClass(t *testing.T) {
	tests := []struct {
		assetClass AssetClass
		ticker     string
		minRatio   float64
	}{
		{AssetClassFixedIncome, "UNKNOWNBOND", 0.15},
		{AssetClassAlternative, "UNKNOWNALT", 0.75},
		{AssetClassCrypto, "UNKNOWNCRYPTO", 1.00},
		{AssetClassCash, "UNKNOWNCASH", 0.40},
	}

	for _, tt := range tests {
		t.Run(string(tt.assetClass), func(t *testing.T) {
			ratio := GetExpenseRatio(tt.ticker, tt.assetClass)
			expected := decimal.NewFromFloat(tt.minRatio)
			if !ratio.Equal(expected) {
				t.Errorf("GetExpenseRatio(%s, %s) = %s, want %s",
					tt.ticker, tt.assetClass, ratio.String(), expected.String())
			}
		})
	}
}

func TestCalculateAnnualExpense(t *testing.T) {
	tests := []struct {
		marketValue  decimal.Decimal
		expenseRatio decimal.Decimal
		expected     decimal.Decimal
	}{
		{
			marketValue:  decimal.NewFromInt(100000),
			expenseRatio: decimal.NewFromFloat(0.03), // 0.03%
			expected:     decimal.NewFromInt(30),     // $30
		},
		{
			marketValue:  decimal.NewFromInt(1000000),
			expenseRatio: decimal.NewFromFloat(0.10), // 0.10%
			expected:     decimal.NewFromInt(1000),   // $1,000
		},
		{
			marketValue:  decimal.NewFromInt(500000),
			expenseRatio: decimal.NewFromFloat(1.00), // 1.00%
			expected:     decimal.NewFromInt(5000),   // $5,000
		},
		{
			marketValue:  decimal.NewFromInt(100000),
			expenseRatio: decimal.Zero,
			expected:     decimal.Zero,
		},
	}

	for _, tt := range tests {
		result := CalculateAnnualExpense(tt.marketValue, tt.expenseRatio)
		if !result.Equal(tt.expected) {
			t.Errorf("CalculateAnnualExpense(%s, %s) = %s, want %s",
				tt.marketValue.String(), tt.expenseRatio.String(),
				result.String(), tt.expected.String())
		}
	}
}

func TestCalculate10YearCost(t *testing.T) {
	initialValue := decimal.NewFromInt(1000000)  // $1M
	expenseRatio := decimal.NewFromFloat(0.50)   // 0.5%
	expectedReturn := decimal.NewFromFloat(7.0)  // 7%

	cost := Calculate10YearCost(initialValue, expenseRatio, expectedReturn)

	// Cost should be positive
	if !cost.IsPositive() {
		t.Error("10-year cost should be positive")
	}

	// Cost should be substantial (roughly 5-10% of initial value)
	minCost := initialValue.Mul(decimal.NewFromFloat(0.04))
	maxCost := initialValue.Mul(decimal.NewFromFloat(0.15))

	if cost.LessThan(minCost) || cost.GreaterThan(maxCost) {
		t.Errorf("10-year cost %s seems unreasonable (expected between %s and %s)",
			cost.String(), minCost.String(), maxCost.String())
	}
}

func TestKnownExpenseRatios(t *testing.T) {
	// Verify we have expense ratios for major ETFs
	expectedETFs := []string{"VOO", "VTI", "SPY", "QQQ", "BND", "AGG", "GLD"}

	for _, etf := range expectedETFs {
		if _, ok := KnownExpenseRatios[etf]; !ok {
			t.Errorf("Missing expense ratio for %s", etf)
		}
	}

	// Verify all ratios are reasonable (0% to 3%)
	for ticker, ratio := range KnownExpenseRatios {
		if ratio.IsNegative() {
			t.Errorf("Expense ratio for %s is negative: %s", ticker, ratio.String())
		}
		if ratio.GreaterThan(decimal.NewFromFloat(3.0)) {
			t.Errorf("Expense ratio for %s seems too high: %s", ticker, ratio.String())
		}
	}
}

func TestBenchmarkExpenseRatio(t *testing.T) {
	// Benchmark should be a reasonable target (0.10%)
	expected := decimal.NewFromFloat(0.10)
	if !BenchmarkExpenseRatio.Equal(expected) {
		t.Errorf("BenchmarkExpenseRatio = %s, want %s",
			BenchmarkExpenseRatio.String(), expected.String())
	}
}

func TestDefaultExpenseRatios(t *testing.T) {
	// Verify all asset classes have defaults
	for _, class := range AllAssetClasses() {
		if _, ok := DefaultExpenseRatios[class]; !ok {
			t.Errorf("Missing default expense ratio for %s", class)
		}
	}
}
