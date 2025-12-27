package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Scenario represents a what-if allocation model
type Scenario struct {
	ID          uuid.UUID                    `json:"id"`
	PortfolioID uuid.UUID                    `json:"portfolio_id"`
	Name        string                       `json:"name"`
	Allocations map[AssetClass]decimal.Decimal `json:"allocations"` // Target percentages
	Projections ScenarioProjections          `json:"projections"`
	CreatedAt   time.Time                    `json:"created_at"`
}

// ScenarioProjections contains projected returns for different market conditions
type ScenarioProjections struct {
	BestCase     decimal.Decimal `json:"best_case"`     // Best year return %
	WorstCase    decimal.Decimal `json:"worst_case"`    // Worst year return %
	AverageCase  decimal.Decimal `json:"average_case"`  // Average annual return %
	MaxDrawdown  decimal.Decimal `json:"max_drawdown"`  // Maximum drawdown %
	ExpectedValue decimal.Decimal `json:"expected_value"` // Projected value after 1 year
}

// NewScenario creates a new scenario with default allocations
func NewScenario(portfolioID uuid.UUID, name string) *Scenario {
	return &Scenario{
		ID:          uuid.New(),
		PortfolioID: portfolioID,
		Name:        name,
		Allocations: make(map[AssetClass]decimal.Decimal),
		CreatedAt:   time.Now().UTC(),
	}
}

// SetAllocation sets the target allocation for an asset class
func (s *Scenario) SetAllocation(class AssetClass, percentage decimal.Decimal) {
	s.Allocations[class] = percentage
}

// TotalAllocation returns the sum of all allocations (should be 100)
func (s *Scenario) TotalAllocation() decimal.Decimal {
	total := decimal.Zero
	for _, pct := range s.Allocations {
		total = total.Add(pct)
	}
	return total
}

// IsValid checks if allocations sum to 100%
func (s *Scenario) IsValid() bool {
	return s.TotalAllocation().Equal(decimal.NewFromInt(100))
}

// Historical return assumptions by asset class (annualized)
// Based on long-term historical averages
var AssetClassReturns = map[AssetClass]AssetClassStats{
	AssetClassEquity: {
		BestYear:    decimal.NewFromFloat(32.4),  // Best year
		WorstYear:   decimal.NewFromFloat(-37.0), // Worst year (2008)
		Average:     decimal.NewFromFloat(10.5),  // Long-term average
		Volatility:  decimal.NewFromFloat(15.0),  // Standard deviation
	},
	AssetClassFixedIncome: {
		BestYear:    decimal.NewFromFloat(18.5),
		WorstYear:   decimal.NewFromFloat(-13.0),
		Average:     decimal.NewFromFloat(5.0),
		Volatility:  decimal.NewFromFloat(6.0),
	},
	AssetClassAlternative: {
		BestYear:    decimal.NewFromFloat(25.0),
		WorstYear:   decimal.NewFromFloat(-20.0),
		Average:     decimal.NewFromFloat(8.0),
		Volatility:  decimal.NewFromFloat(12.0),
	},
	AssetClassCrypto: {
		BestYear:    decimal.NewFromFloat(300.0),
		WorstYear:   decimal.NewFromFloat(-75.0),
		Average:     decimal.NewFromFloat(50.0),
		Volatility:  decimal.NewFromFloat(80.0),
	},
	AssetClassCash: {
		BestYear:    decimal.NewFromFloat(5.0),
		WorstYear:   decimal.NewFromFloat(0.0),
		Average:     decimal.NewFromFloat(2.0),
		Volatility:  decimal.NewFromFloat(0.5),
	},
	AssetClassOther: {
		BestYear:    decimal.NewFromFloat(10.0),
		WorstYear:   decimal.NewFromFloat(-10.0),
		Average:     decimal.NewFromFloat(5.0),
		Volatility:  decimal.NewFromFloat(10.0),
	},
}

// AssetClassStats holds historical return statistics
type AssetClassStats struct {
	BestYear   decimal.Decimal `json:"best_year"`
	WorstYear  decimal.Decimal `json:"worst_year"`
	Average    decimal.Decimal `json:"average"`
	Volatility decimal.Decimal `json:"volatility"`
}

// CalculateProjections computes projected returns based on allocations
func (s *Scenario) CalculateProjections(currentValue decimal.Decimal) {
	hundred := decimal.NewFromInt(100)
	bestCase := decimal.Zero
	worstCase := decimal.Zero
	avgCase := decimal.Zero
	volatility := decimal.Zero

	for class, allocation := range s.Allocations {
		weight := allocation.Div(hundred)
		stats := AssetClassReturns[class]

		bestCase = bestCase.Add(stats.BestYear.Mul(weight))
		worstCase = worstCase.Add(stats.WorstYear.Mul(weight))
		avgCase = avgCase.Add(stats.Average.Mul(weight))
		// Simplified volatility calculation (not accounting for correlation)
		volatility = volatility.Add(stats.Volatility.Mul(weight))
	}

	s.Projections = ScenarioProjections{
		BestCase:      bestCase.Round(2),
		WorstCase:     worstCase.Round(2),
		AverageCase:   avgCase.Round(2),
		MaxDrawdown:   worstCase.Round(2), // Simplified: use worst year as proxy
		ExpectedValue: currentValue.Mul(decimal.NewFromInt(1).Add(avgCase.Div(hundred))).Round(2),
	}
}

// ScenarioComparison compares current allocation to a target scenario
type ScenarioComparison struct {
	Current   map[AssetClass]decimal.Decimal `json:"current"`
	Target    map[AssetClass]decimal.Decimal `json:"target"`
	Changes   map[AssetClass]decimal.Decimal `json:"changes"` // Difference
	Rebalance map[AssetClass]decimal.Decimal `json:"rebalance"` // Dollar amounts to move
}

// Compare creates a comparison between current allocation and scenario target
func (s *Scenario) Compare(current map[AssetClass]decimal.Decimal, totalValue decimal.Decimal) *ScenarioComparison {
	comparison := &ScenarioComparison{
		Current:   current,
		Target:    s.Allocations,
		Changes:   make(map[AssetClass]decimal.Decimal),
		Rebalance: make(map[AssetClass]decimal.Decimal),
	}

	hundred := decimal.NewFromInt(100)

	for _, class := range AllAssetClasses() {
		currentPct := current[class]
		targetPct := s.Allocations[class]
		diff := targetPct.Sub(currentPct)
		comparison.Changes[class] = diff.Round(2)
		comparison.Rebalance[class] = totalValue.Mul(diff).Div(hundred).Round(2)
	}

	return comparison
}
