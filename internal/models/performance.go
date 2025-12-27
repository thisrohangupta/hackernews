package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// PriceHistory stores historical price data for a ticker
type PriceHistory struct {
	Ticker    string          `json:"ticker"`
	Date      time.Time       `json:"date"`
	Open      decimal.Decimal `json:"open"`
	High      decimal.Decimal `json:"high"`
	Low       decimal.Decimal `json:"low"`
	Close     decimal.Decimal `json:"close"`
	AdjClose  decimal.Decimal `json:"adj_close"` // Adjusted for splits/dividends
	Volume    int64           `json:"volume"`
}

// PerformanceMetrics holds calculated performance data
type PerformanceMetrics struct {
	Ticker          string          `json:"ticker"`
	Period          string          `json:"period"` // 1d, 1w, 1m, 3m, 6m, 1y, 3y, 5y, ytd
	StartDate       time.Time       `json:"start_date"`
	EndDate         time.Time       `json:"end_date"`
	StartPrice      decimal.Decimal `json:"start_price"`
	EndPrice        decimal.Decimal `json:"end_price"`
	TotalReturn     decimal.Decimal `json:"total_return"`      // Percentage
	AnnualizedReturn decimal.Decimal `json:"annualized_return"` // CAGR
	Volatility      decimal.Decimal `json:"volatility"`        // Standard deviation of returns
	MaxDrawdown     decimal.Decimal `json:"max_drawdown"`      // Worst peak-to-trough
	SharpeRatio     decimal.Decimal `json:"sharpe_ratio"`      // Risk-adjusted return
	Beta            decimal.Decimal `json:"beta"`              // Market correlation
}

// PortfolioPerformance holds performance for entire portfolio
type PortfolioPerformance struct {
	PortfolioID      string               `json:"portfolio_id"`
	Period           string               `json:"period"`
	StartValue       decimal.Decimal      `json:"start_value"`
	EndValue         decimal.Decimal      `json:"end_value"`
	TotalReturn      decimal.Decimal      `json:"total_return"`
	AnnualizedReturn decimal.Decimal      `json:"annualized_return"`
	Volatility       decimal.Decimal      `json:"volatility"`
	SharpeRatio      decimal.Decimal      `json:"sharpe_ratio"`
	MaxDrawdown      decimal.Decimal      `json:"max_drawdown"`
	BestMonth        decimal.Decimal      `json:"best_month"`
	WorstMonth       decimal.Decimal      `json:"worst_month"`
	PositiveMonths   int                  `json:"positive_months"`
	NegativeMonths   int                  `json:"negative_months"`
	Holdings         []HoldingPerformance `json:"holdings,omitempty"`
}

// HoldingPerformance tracks individual holding performance
type HoldingPerformance struct {
	Ticker           string          `json:"ticker"`
	Name             string          `json:"name"`
	StartValue       decimal.Decimal `json:"start_value"`
	EndValue         decimal.Decimal `json:"end_value"`
	TotalReturn      decimal.Decimal `json:"total_return"`
	ContributionPct  decimal.Decimal `json:"contribution_pct"` // Contribution to portfolio return
	Weight           decimal.Decimal `json:"weight"`           // Current portfolio weight
}

// ValueSnapshot stores portfolio value at a point in time
type ValueSnapshot struct {
	PortfolioID string          `json:"portfolio_id"`
	Date        time.Time       `json:"date"`
	TotalValue  decimal.Decimal `json:"total_value"`
	CashValue   decimal.Decimal `json:"cash_value"`
}

// TimeSeriesPoint for charting
type TimeSeriesPoint struct {
	Date  time.Time       `json:"date"`
	Value decimal.Decimal `json:"value"`
}

// PerformancePeriod constants
const (
	Period1Day   = "1d"
	Period1Week  = "1w"
	Period1Month = "1m"
	Period3Month = "3m"
	Period6Month = "6m"
	Period1Year  = "1y"
	Period3Year  = "3y"
	Period5Year  = "5y"
	PeriodYTD    = "ytd"
	PeriodAll    = "all"
)

// GetPeriodDuration returns the duration for a period string
func GetPeriodDuration(period string) time.Duration {
	switch period {
	case Period1Day:
		return 24 * time.Hour
	case Period1Week:
		return 7 * 24 * time.Hour
	case Period1Month:
		return 30 * 24 * time.Hour
	case Period3Month:
		return 90 * 24 * time.Hour
	case Period6Month:
		return 180 * 24 * time.Hour
	case Period1Year:
		return 365 * 24 * time.Hour
	case Period3Year:
		return 3 * 365 * 24 * time.Hour
	case Period5Year:
		return 5 * 365 * 24 * time.Hour
	default:
		return 365 * 24 * time.Hour
	}
}

// GetPeriodStartDate calculates start date for a period
func GetPeriodStartDate(period string) time.Time {
	now := time.Now().UTC()

	switch period {
	case Period1Day:
		return now.AddDate(0, 0, -1)
	case Period1Week:
		return now.AddDate(0, 0, -7)
	case Period1Month:
		return now.AddDate(0, -1, 0)
	case Period3Month:
		return now.AddDate(0, -3, 0)
	case Period6Month:
		return now.AddDate(0, -6, 0)
	case Period1Year:
		return now.AddDate(-1, 0, 0)
	case Period3Year:
		return now.AddDate(-3, 0, 0)
	case Period5Year:
		return now.AddDate(-5, 0, 0)
	case PeriodYTD:
		return time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
	default:
		return now.AddDate(-1, 0, 0)
	}
}
