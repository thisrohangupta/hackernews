package models

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestCalculateQuadrant(t *testing.T) {
	avgReturn := decimal.NewFromInt(10)
	avgVolatility := decimal.NewFromInt(15)

	tests := []struct {
		name       string
		returnPct  decimal.Decimal
		volatility decimal.Decimal
		expected   string
	}{
		{
			name:       "Optimal - high return, low risk",
			returnPct:  decimal.NewFromInt(15),
			volatility: decimal.NewFromInt(10),
			expected:   "optimal",
		},
		{
			name:       "Aggressive - high return, high risk",
			returnPct:  decimal.NewFromInt(15),
			volatility: decimal.NewFromInt(20),
			expected:   "aggressive",
		},
		{
			name:       "Conservative - low return, low risk",
			returnPct:  decimal.NewFromInt(5),
			volatility: decimal.NewFromInt(10),
			expected:   "conservative",
		},
		{
			name:       "Avoid - low return, high risk",
			returnPct:  decimal.NewFromInt(5),
			volatility: decimal.NewFromInt(20),
			expected:   "avoid",
		},
		{
			name:       "Edge case - exactly at average",
			returnPct:  decimal.NewFromInt(10),
			volatility: decimal.NewFromInt(15),
			expected:   "optimal", // Return >= avg, volatility not > avg
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateQuadrant(tt.returnPct, tt.volatility, avgReturn, avgVolatility)
			if result != tt.expected {
				t.Errorf("CalculateQuadrant() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestRiskFreeRate(t *testing.T) {
	// Risk-free rate should be reasonable (between 0% and 10%)
	if RiskFreeRate.IsNegative() {
		t.Error("Risk-free rate should not be negative")
	}
	if RiskFreeRate.GreaterThan(decimal.NewFromFloat(0.10)) {
		t.Error("Risk-free rate seems too high")
	}
}

func TestDefaultRiskMetrics(t *testing.T) {
	// Test that all asset classes return valid metrics
	for _, class := range AllAssetClasses() {
		t.Run(string(class), func(t *testing.T) {
			metrics := DefaultRiskMetrics(class)

			// Expected return should be set
			if metrics.ExpectedReturn.IsZero() && class != AssetClassCash {
				t.Errorf("Expected return for %s should not be zero", class)
			}

			// Volatility should be set
			if metrics.Volatility.IsZero() && class != AssetClassCash {
				t.Errorf("Volatility for %s should not be zero", class)
			}
		})
	}
}

func TestBenchmarkReturns(t *testing.T) {
	// Test that SPY benchmark exists and has reasonable values
	spy, ok := BenchmarkReturns["SPY"]
	if !ok {
		t.Fatal("SPY benchmark should exist")
	}

	if spy.AnnualizedReturn.IsZero() {
		t.Error("SPY annualized return should not be zero")
	}

	if spy.Volatility.IsZero() {
		t.Error("SPY volatility should not be zero")
	}

	if !spy.Beta.Equal(decimal.NewFromInt(1)) {
		t.Errorf("SPY beta should be 1, got %s", spy.Beta.String())
	}

	// AGG benchmark
	agg, ok := BenchmarkReturns["AGG"]
	if !ok {
		t.Fatal("AGG benchmark should exist")
	}

	if agg.Volatility.GreaterThan(spy.Volatility) {
		t.Error("Bond index should have lower volatility than stock index")
	}
}
