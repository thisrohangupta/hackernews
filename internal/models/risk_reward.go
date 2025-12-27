package models

import (
	"github.com/shopspring/decimal"
)

// RiskRewardMatrix provides comprehensive risk-reward analysis
type RiskRewardMatrix struct {
	PortfolioID    string                       `json:"portfolio_id"`
	CalculatedAt   string                       `json:"calculated_at"`
	Period         string                       `json:"period"`

	// Portfolio-level metrics
	Portfolio      RiskRewardMetrics            `json:"portfolio"`

	// Asset class breakdown
	ByAssetClass   map[AssetClass]RiskRewardMetrics `json:"by_asset_class"`

	// Individual holdings (top 20)
	Holdings       []HoldingRiskReward          `json:"holdings"`

	// Quadrant classification
	Quadrants      QuadrantAnalysis             `json:"quadrants"`

	// Correlation matrix
	Correlations   []CorrelationPair            `json:"correlations,omitempty"`
}

// RiskRewardMetrics contains risk-reward calculations
type RiskRewardMetrics struct {
	// Return metrics
	TotalReturn       decimal.Decimal `json:"total_return"`
	AnnualizedReturn  decimal.Decimal `json:"annualized_return"`  // CAGR
	ExpectedReturn    decimal.Decimal `json:"expected_return"`    // Forward-looking

	// Risk metrics
	Volatility        decimal.Decimal `json:"volatility"`         // Annualized std dev
	DownsideDeviation decimal.Decimal `json:"downside_deviation"` // Downside volatility
	MaxDrawdown       decimal.Decimal `json:"max_drawdown"`
	VaR95             decimal.Decimal `json:"var_95"`             // Value at Risk 95%

	// Risk-adjusted metrics
	SharpeRatio       decimal.Decimal `json:"sharpe_ratio"`       // (Return - RiskFree) / Volatility
	SortinoRatio      decimal.Decimal `json:"sortino_ratio"`      // (Return - RiskFree) / DownsideDeviation
	CalmarRatio       decimal.Decimal `json:"calmar_ratio"`       // Return / MaxDrawdown

	// Market metrics
	Beta              decimal.Decimal `json:"beta"`               // Market sensitivity
	Alpha             decimal.Decimal `json:"alpha"`              // Excess return vs benchmark
	RSquared          decimal.Decimal `json:"r_squared"`          // Correlation with market

	// Treynor ratio = (Return - RiskFree) / Beta
	TreynorRatio      decimal.Decimal `json:"treynor_ratio"`
}

// HoldingRiskReward contains risk-reward for a single holding
type HoldingRiskReward struct {
	Ticker            string              `json:"ticker"`
	Name              string              `json:"name"`
	AssetClass        AssetClass          `json:"asset_class"`
	Weight            decimal.Decimal     `json:"weight"`
	Metrics           RiskRewardMetrics   `json:"metrics"`
	Quadrant          string              `json:"quadrant"` // "optimal", "aggressive", "conservative", "avoid"
	RiskContribution  decimal.Decimal     `json:"risk_contribution"`  // % of portfolio risk
}

// QuadrantAnalysis categorizes holdings into risk-reward quadrants
type QuadrantAnalysis struct {
	// High Return, Low Risk - Best performers
	Optimal       []string `json:"optimal"`
	OptimalValue  decimal.Decimal `json:"optimal_value"`

	// High Return, High Risk - Growth/aggressive
	Aggressive    []string `json:"aggressive"`
	AggressiveValue decimal.Decimal `json:"aggressive_value"`

	// Low Return, Low Risk - Conservative/stable
	Conservative  []string `json:"conservative"`
	ConservativeValue decimal.Decimal `json:"conservative_value"`

	// Low Return, High Risk - Underperformers
	Avoid         []string `json:"avoid"`
	AvoidValue    decimal.Decimal `json:"avoid_value"`
}

// CorrelationPair shows correlation between two assets
type CorrelationPair struct {
	Ticker1     string          `json:"ticker1"`
	Ticker2     string          `json:"ticker2"`
	Correlation decimal.Decimal `json:"correlation"` // -1 to 1
}

// RiskFreeRate is the assumed risk-free rate (10-year Treasury)
var RiskFreeRate = decimal.NewFromFloat(0.045) // 4.5% as of late 2024

// CalculateQuadrant determines which quadrant a holding belongs to
func CalculateQuadrant(returnPct, volatility decimal.Decimal, avgReturn, avgVolatility decimal.Decimal) string {
	highReturn := returnPct.GreaterThanOrEqual(avgReturn)
	highRisk := volatility.GreaterThan(avgVolatility)

	switch {
	case highReturn && !highRisk:
		return "optimal"
	case highReturn && highRisk:
		return "aggressive"
	case !highReturn && !highRisk:
		return "conservative"
	default:
		return "avoid"
	}
}

// BenchmarkReturns provides market benchmark data
var BenchmarkReturns = map[string]RiskRewardMetrics{
	"SPY": { // S&P 500
		AnnualizedReturn: decimal.NewFromFloat(10.5),
		Volatility:       decimal.NewFromFloat(15.0),
		SharpeRatio:      decimal.NewFromFloat(0.40),
		MaxDrawdown:      decimal.NewFromFloat(-33.9),
		Beta:             decimal.NewFromInt(1),
	},
	"AGG": { // Aggregate Bond
		AnnualizedReturn: decimal.NewFromFloat(4.5),
		Volatility:       decimal.NewFromFloat(5.5),
		SharpeRatio:      decimal.NewFromFloat(0.00),
		MaxDrawdown:      decimal.NewFromFloat(-17.0),
		Beta:             decimal.NewFromFloat(0.1),
	},
}

// DefaultRiskMetrics returns default metrics for an asset class
func DefaultRiskMetrics(class AssetClass) RiskRewardMetrics {
	stats := AssetClassReturns[class]

	sharpe := decimal.Zero
	if !stats.Volatility.IsZero() {
		excessReturn := stats.Average.Sub(RiskFreeRate.Mul(decimal.NewFromInt(100)))
		sharpe = excessReturn.Div(stats.Volatility).Round(2)
	}

	return RiskRewardMetrics{
		ExpectedReturn:   stats.Average,
		Volatility:       stats.Volatility,
		MaxDrawdown:      stats.WorstYear,
		SharpeRatio:      sharpe,
	}
}
