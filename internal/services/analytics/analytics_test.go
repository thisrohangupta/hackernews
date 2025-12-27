package analytics

import (
	"testing"

	"github.com/findosh/truenorth/internal/models"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestNewService(t *testing.T) {
	svc := NewService()
	if svc == nil {
		t.Fatal("Expected service to be created")
	}
}

func TestService_CalculatePortfolioPerformance_Empty(t *testing.T) {
	svc := NewService()

	// Nil portfolio
	perf := svc.CalculatePortfolioPerformance(nil, models.Period1Year)
	if perf != nil {
		t.Error("Expected nil for nil portfolio")
	}

	// Empty portfolio
	portfolio := &models.Portfolio{Holdings: []models.Holding{}}
	perf = svc.CalculatePortfolioPerformance(portfolio, models.Period1Year)
	if perf != nil {
		t.Error("Expected nil for empty portfolio")
	}
}

func TestService_CalculatePortfolioPerformance(t *testing.T) {
	svc := NewService()

	portfolio := createTestPortfolio()
	perf := svc.CalculatePortfolioPerformance(portfolio, models.Period1Year)

	if perf == nil {
		t.Fatal("Expected performance to be calculated")
	}

	if perf.PortfolioID != portfolio.ID.String() {
		t.Error("Portfolio ID should match")
	}

	if perf.Period != models.Period1Year {
		t.Errorf("Period should be %s, got %s", models.Period1Year, perf.Period)
	}

	if perf.EndValue.IsZero() {
		t.Error("End value should not be zero")
	}
}

func TestService_CalculateRiskRewardMatrix(t *testing.T) {
	svc := NewService()

	portfolio := createTestPortfolio()
	matrix := svc.CalculateRiskRewardMatrix(portfolio)

	if matrix == nil {
		t.Fatal("Expected risk-reward matrix to be calculated")
	}

	if matrix.PortfolioID != portfolio.ID.String() {
		t.Error("Portfolio ID should match")
	}

	if len(matrix.ByAssetClass) == 0 {
		t.Error("Expected asset class breakdown")
	}

	if len(matrix.Holdings) == 0 {
		t.Error("Expected holdings breakdown")
	}
}

func TestService_CalculateRiskRewardMatrix_Nil(t *testing.T) {
	svc := NewService()

	matrix := svc.CalculateRiskRewardMatrix(nil)
	if matrix != nil {
		t.Error("Expected nil for nil portfolio")
	}
}

func TestService_CalculateExpenses(t *testing.T) {
	svc := NewService()

	portfolio := createTestPortfolio()
	expenses := svc.CalculateExpenses(portfolio)

	if expenses == nil {
		t.Fatal("Expected expenses to be calculated")
	}

	if expenses.WeightedExpenseRatio.IsZero() {
		t.Error("Expected weighted expense ratio to be calculated")
	}

	if len(expenses.ByAssetClass) == 0 {
		t.Error("Expected asset class expense breakdown")
	}
}

func TestService_CalculateExpenses_Nil(t *testing.T) {
	svc := NewService()

	expenses := svc.CalculateExpenses(nil)
	if expenses != nil {
		t.Error("Expected nil for nil portfolio")
	}
}

func TestService_GenerateTimeSeries(t *testing.T) {
	svc := NewService()

	portfolio := createTestPortfolio()
	series := svc.GenerateTimeSeries(portfolio, models.Period1Year)

	if series == nil {
		t.Fatal("Expected time series to be generated")
	}

	if len(series) == 0 {
		t.Error("Expected time series to have points")
	}

	// Check that last point matches portfolio value
	lastPoint := series[len(series)-1]
	if !lastPoint.Value.Equal(portfolio.TotalValue) {
		t.Errorf("Last point should equal portfolio value: got %s, want %s",
			lastPoint.Value.String(), portfolio.TotalValue.String())
	}
}

func TestService_GenerateTimeSeries_Nil(t *testing.T) {
	svc := NewService()

	series := svc.GenerateTimeSeries(nil, models.Period1Year)
	if series != nil {
		t.Error("Expected nil for nil portfolio")
	}
}

func TestService_AnalyzeQuadrants(t *testing.T) {
	svc := NewService()

	portfolio := createTestPortfolio()
	matrix := svc.CalculateRiskRewardMatrix(portfolio)

	if matrix == nil {
		t.Fatal("Expected matrix")
	}

	// Check that all holdings are categorized
	totalCategorized := len(matrix.Quadrants.Optimal) +
		len(matrix.Quadrants.Aggressive) +
		len(matrix.Quadrants.Conservative) +
		len(matrix.Quadrants.Avoid)

	if totalCategorized == 0 {
		t.Error("Expected at least some holdings to be categorized")
	}
}

// Helper to create a test portfolio
func createTestPortfolio() *models.Portfolio {
	portfolioID := uuid.New()

	portfolio := &models.Portfolio{
		ID:         portfolioID,
		UserID:     uuid.New(),
		Name:       "Test Portfolio",
		TotalValue: decimal.NewFromInt(1000000),
		Holdings: []models.Holding{
			{
				ID:          uuid.New(),
				PortfolioID: portfolioID,
				Ticker:      "VOO",
				Name:        "Vanguard S&P 500 ETF",
				Quantity:    decimal.NewFromInt(100),
				MarketValue: decimal.NewFromInt(500000),
				AssetClass:  models.AssetClassEquity,
				Sector:      "Diversified",
				Geography:   "US",
			},
			{
				ID:          uuid.New(),
				PortfolioID: portfolioID,
				Ticker:      "BND",
				Name:        "Vanguard Total Bond Market ETF",
				Quantity:    decimal.NewFromInt(200),
				MarketValue: decimal.NewFromInt(300000),
				AssetClass:  models.AssetClassFixedIncome,
				Sector:      "Bonds",
				Geography:   "US",
			},
			{
				ID:          uuid.New(),
				PortfolioID: portfolioID,
				Ticker:      "SPAXX",
				Name:        "Fidelity Money Market",
				Quantity:    decimal.NewFromInt(200000),
				MarketValue: decimal.NewFromInt(200000),
				AssetClass:  models.AssetClassCash,
				Sector:      "Cash",
				Geography:   "US",
			},
		},
	}

	return portfolio
}
