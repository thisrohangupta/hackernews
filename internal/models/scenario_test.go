package models

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestNewScenario(t *testing.T) {
	portfolioID := uuid.New()
	s := NewScenario(portfolioID, "Test Scenario")

	if s.ID == uuid.Nil {
		t.Error("Expected scenario ID to be generated")
	}
	if s.PortfolioID != portfolioID {
		t.Errorf("Expected portfolio ID %v, got %v", portfolioID, s.PortfolioID)
	}
	if s.Name != "Test Scenario" {
		t.Errorf("Expected name 'Test Scenario', got '%s'", s.Name)
	}
	if s.Allocations == nil {
		t.Error("Expected allocations map to be initialized")
	}
}

func TestScenario_SetAllocation(t *testing.T) {
	s := NewScenario(uuid.New(), "Test")

	s.SetAllocation(AssetClassEquity, decimal.NewFromInt(60))
	s.SetAllocation(AssetClassFixedIncome, decimal.NewFromInt(30))
	s.SetAllocation(AssetClassCash, decimal.NewFromInt(10))

	if !s.Allocations[AssetClassEquity].Equal(decimal.NewFromInt(60)) {
		t.Errorf("Expected equity allocation 60, got %s", s.Allocations[AssetClassEquity])
	}
	if !s.Allocations[AssetClassFixedIncome].Equal(decimal.NewFromInt(30)) {
		t.Errorf("Expected fixed income allocation 30, got %s", s.Allocations[AssetClassFixedIncome])
	}
	if !s.Allocations[AssetClassCash].Equal(decimal.NewFromInt(10)) {
		t.Errorf("Expected cash allocation 10, got %s", s.Allocations[AssetClassCash])
	}
}

func TestScenario_TotalAllocation(t *testing.T) {
	s := NewScenario(uuid.New(), "Test")
	s.SetAllocation(AssetClassEquity, decimal.NewFromInt(60))
	s.SetAllocation(AssetClassFixedIncome, decimal.NewFromInt(30))
	s.SetAllocation(AssetClassCash, decimal.NewFromInt(10))

	total := s.TotalAllocation()

	if !total.Equal(decimal.NewFromInt(100)) {
		t.Errorf("Expected total allocation 100, got %s", total)
	}
}

func TestScenario_IsValid(t *testing.T) {
	tests := []struct {
		name        string
		allocations map[AssetClass]decimal.Decimal
		expected    bool
	}{
		{
			name: "Valid - sums to 100",
			allocations: map[AssetClass]decimal.Decimal{
				AssetClassEquity:      decimal.NewFromInt(60),
				AssetClassFixedIncome: decimal.NewFromInt(30),
				AssetClassCash:        decimal.NewFromInt(10),
			},
			expected: true,
		},
		{
			name: "Invalid - sums to 90",
			allocations: map[AssetClass]decimal.Decimal{
				AssetClassEquity:      decimal.NewFromInt(50),
				AssetClassFixedIncome: decimal.NewFromInt(30),
				AssetClassCash:        decimal.NewFromInt(10),
			},
			expected: false,
		},
		{
			name: "Invalid - sums to 110",
			allocations: map[AssetClass]decimal.Decimal{
				AssetClassEquity:      decimal.NewFromInt(70),
				AssetClassFixedIncome: decimal.NewFromInt(30),
				AssetClassCash:        decimal.NewFromInt(10),
			},
			expected: false,
		},
		{
			name:        "Invalid - empty",
			allocations: map[AssetClass]decimal.Decimal{},
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Scenario{Allocations: tt.allocations}
			if s.IsValid() != tt.expected {
				t.Errorf("IsValid() = %v, expected %v", s.IsValid(), tt.expected)
			}
		})
	}
}

func TestScenario_CalculateProjections(t *testing.T) {
	s := NewScenario(uuid.New(), "Balanced")
	s.SetAllocation(AssetClassEquity, decimal.NewFromInt(60))
	s.SetAllocation(AssetClassFixedIncome, decimal.NewFromInt(30))
	s.SetAllocation(AssetClassCash, decimal.NewFromInt(10))

	currentValue := decimal.NewFromFloat(1000000.00)
	s.CalculateProjections(currentValue)

	// Check projections are calculated
	if s.Projections.BestCase.IsZero() {
		t.Error("Expected non-zero best case projection")
	}
	if s.Projections.WorstCase.IsZero() {
		t.Error("Expected non-zero worst case projection")
	}
	if s.Projections.AverageCase.IsZero() {
		t.Error("Expected non-zero average case projection")
	}
	if s.Projections.ExpectedValue.IsZero() {
		t.Error("Expected non-zero expected value")
	}

	// Expected value should be reasonable (around current value +/- expected return)
	// With 60% equity (avg ~10.5%), 30% bonds (avg ~5%), 10% cash (avg ~2%)
	// Weighted avg return ≈ 7.7%
	// Expected value ≈ $1,077,000
	minExpected := decimal.NewFromFloat(1000000.00)
	maxExpected := decimal.NewFromFloat(1200000.00)

	if s.Projections.ExpectedValue.LessThan(minExpected) || s.Projections.ExpectedValue.GreaterThan(maxExpected) {
		t.Errorf("Expected value %s outside reasonable range [%s, %s]",
			s.Projections.ExpectedValue, minExpected, maxExpected)
	}
}

func TestScenario_Compare(t *testing.T) {
	s := NewScenario(uuid.New(), "Target")
	s.SetAllocation(AssetClassEquity, decimal.NewFromInt(70))
	s.SetAllocation(AssetClassFixedIncome, decimal.NewFromInt(20))
	s.SetAllocation(AssetClassCash, decimal.NewFromInt(10))

	current := map[AssetClass]decimal.Decimal{
		AssetClassEquity:      decimal.NewFromInt(60),
		AssetClassFixedIncome: decimal.NewFromInt(30),
		AssetClassCash:        decimal.NewFromInt(10),
	}

	totalValue := decimal.NewFromFloat(1000000.00)
	comparison := s.Compare(current, totalValue)

	// Check equity change: 70 - 60 = +10%
	equityChange := comparison.Changes[AssetClassEquity]
	if !equityChange.Equal(decimal.NewFromInt(10)) {
		t.Errorf("Expected equity change +10, got %s", equityChange)
	}

	// Check rebalance amount: 10% of $1M = $100,000
	equityRebalance := comparison.Rebalance[AssetClassEquity]
	if !equityRebalance.Equal(decimal.NewFromFloat(100000.00)) {
		t.Errorf("Expected equity rebalance $100,000, got %s", equityRebalance)
	}

	// Check fixed income change: 20 - 30 = -10%
	bondChange := comparison.Changes[AssetClassFixedIncome]
	if !bondChange.Equal(decimal.NewFromInt(-10)) {
		t.Errorf("Expected fixed income change -10, got %s", bondChange)
	}

	// Cash should be unchanged: 10 - 10 = 0
	cashChange := comparison.Changes[AssetClassCash]
	if !cashChange.IsZero() {
		t.Errorf("Expected cash change 0, got %s", cashChange)
	}
}

func TestAssetClassReturns(t *testing.T) {
	// Verify all asset classes have return statistics
	for _, class := range AllAssetClasses() {
		stats, ok := AssetClassReturns[class]
		if !ok {
			t.Errorf("Missing return statistics for asset class %s", class)
			continue
		}

		// Verify stats are reasonable
		if stats.BestYear.LessThanOrEqual(stats.WorstYear) {
			t.Errorf("Asset class %s: best year %s <= worst year %s",
				class, stats.BestYear, stats.WorstYear)
		}
		if stats.Average.LessThan(stats.WorstYear) || stats.Average.GreaterThan(stats.BestYear) {
			t.Errorf("Asset class %s: average %s outside range [%s, %s]",
				class, stats.Average, stats.WorstYear, stats.BestYear)
		}
		if stats.Volatility.IsNegative() {
			t.Errorf("Asset class %s: negative volatility %s", class, stats.Volatility)
		}
	}
}
