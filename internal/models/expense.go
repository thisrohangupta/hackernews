package models

import (
	"github.com/shopspring/decimal"
)

// ExpenseInfo holds expense data for a holding
type ExpenseInfo struct {
	ExpenseRatio     decimal.Decimal `json:"expense_ratio"`      // Annual expense ratio (e.g., 0.03 = 0.03%)
	ManagementFee    decimal.Decimal `json:"management_fee"`     // Management fee if separate
	TradingCosts     decimal.Decimal `json:"trading_costs"`      // Estimated trading costs
	TotalExpense     decimal.Decimal `json:"total_expense"`      // Sum of all expenses
	ExpenseDollars   decimal.Decimal `json:"expense_dollars"`    // Annual cost in dollars
}

// PortfolioExpenses aggregates expense data across portfolio
type PortfolioExpenses struct {
	// Weighted average expense ratio
	WeightedExpenseRatio decimal.Decimal `json:"weighted_expense_ratio"`

	// Total annual expenses in dollars
	TotalAnnualExpenses  decimal.Decimal `json:"total_annual_expenses"`

	// Breakdown by asset class
	ByAssetClass         map[AssetClass]AssetClassExpenses `json:"by_asset_class"`

	// Top expensive holdings
	HighestExpense       []HoldingExpense `json:"highest_expense"`

	// Comparison to benchmark
	VsBenchmark          decimal.Decimal `json:"vs_benchmark"` // Difference from typical

	// 10-year cost projection
	TenYearCost          decimal.Decimal `json:"ten_year_cost"`

	// Potential savings with lower-cost alternatives
	PotentialSavings     decimal.Decimal `json:"potential_savings"`
}

// AssetClassExpenses holds expenses for an asset class
type AssetClassExpenses struct {
	AvgExpenseRatio  decimal.Decimal `json:"avg_expense_ratio"`
	TotalValue       decimal.Decimal `json:"total_value"`
	AnnualCost       decimal.Decimal `json:"annual_cost"`
	HoldingCount     int             `json:"holding_count"`
}

// HoldingExpense shows expense for a single holding
type HoldingExpense struct {
	Ticker          string          `json:"ticker"`
	Name            string          `json:"name"`
	ExpenseRatio    decimal.Decimal `json:"expense_ratio"`
	MarketValue     decimal.Decimal `json:"market_value"`
	AnnualCost      decimal.Decimal `json:"annual_cost"`
	AssetClass      AssetClass      `json:"asset_class"`
}

// Known ETF/Fund expense ratios (in percentage, e.g., 0.03 = 0.03%)
var KnownExpenseRatios = map[string]decimal.Decimal{
	// Vanguard ETFs (very low cost)
	"VOO":  decimal.NewFromFloat(0.03),
	"VTI":  decimal.NewFromFloat(0.03),
	"VEA":  decimal.NewFromFloat(0.05),
	"VWO":  decimal.NewFromFloat(0.08),
	"VXUS": decimal.NewFromFloat(0.07),
	"BND":  decimal.NewFromFloat(0.03),
	"VNQ":  decimal.NewFromFloat(0.12),

	// iShares ETFs
	"IVV":  decimal.NewFromFloat(0.03),
	"AGG":  decimal.NewFromFloat(0.03),
	"TLT":  decimal.NewFromFloat(0.15),
	"IEF":  decimal.NewFromFloat(0.15),
	"SHY":  decimal.NewFromFloat(0.15),
	"TIP":  decimal.NewFromFloat(0.19),

	// SPDR ETFs
	"SPY":  decimal.NewFromFloat(0.09),
	"GLD":  decimal.NewFromFloat(0.40),

	// Invesco
	"QQQ":  decimal.NewFromFloat(0.20),

	// Schwab (ultra-low cost)
	"SCHB": decimal.NewFromFloat(0.03),
	"SCHX": decimal.NewFromFloat(0.03),
	"SCHF": decimal.NewFromFloat(0.06),

	// Fidelity (zero expense)
	"FZROX": decimal.NewFromFloat(0.00),
	"FZILX": decimal.NewFromFloat(0.00),

	// Money market (typically 0.4-0.5%)
	"SPAXX": decimal.NewFromFloat(0.42),
	"FDRXX": decimal.NewFromFloat(0.42),
	"VMFXX": decimal.NewFromFloat(0.11),
	"SWVXX": decimal.NewFromFloat(0.34),

	// Crypto (higher fees)
	"GBTC": decimal.NewFromFloat(1.50),
	"ETHE": decimal.NewFromFloat(2.50),
	"BITO": decimal.NewFromFloat(0.95),
}

// DefaultExpenseRatios by asset class for unknowns
var DefaultExpenseRatios = map[AssetClass]decimal.Decimal{
	AssetClassEquity:      decimal.NewFromFloat(0.10), // 0.10% for stocks/ETFs
	AssetClassFixedIncome: decimal.NewFromFloat(0.15), // 0.15% for bond funds
	AssetClassAlternative: decimal.NewFromFloat(0.75), // 0.75% for alternatives
	AssetClassCrypto:      decimal.NewFromFloat(1.00), // 1.00% for crypto products
	AssetClassCash:        decimal.NewFromFloat(0.40), // 0.40% for money market
	AssetClassOther:       decimal.NewFromFloat(0.50), // 0.50% default
}

// BenchmarkExpenseRatio is the target for a well-optimized portfolio
var BenchmarkExpenseRatio = decimal.NewFromFloat(0.10) // 0.10%

// GetExpenseRatio returns the expense ratio for a ticker
func GetExpenseRatio(ticker string, assetClass AssetClass) decimal.Decimal {
	// Check known ratios first
	if ratio, ok := KnownExpenseRatios[ticker]; ok {
		return ratio
	}

	// Individual stocks have no expense ratio
	if assetClass == AssetClassEquity && len(ticker) <= 5 {
		// Likely a stock, not an ETF
		return decimal.Zero
	}

	// Use default for asset class
	if ratio, ok := DefaultExpenseRatios[assetClass]; ok {
		return ratio
	}

	return decimal.NewFromFloat(0.25) // Conservative default
}

// CalculateAnnualExpense calculates annual expense in dollars
func CalculateAnnualExpense(marketValue, expenseRatio decimal.Decimal) decimal.Decimal {
	// Convert ratio from percentage (0.03) to decimal (0.0003)
	ratioDecimal := expenseRatio.Div(decimal.NewFromInt(100))
	return marketValue.Mul(ratioDecimal).Round(2)
}

// Calculate10YearCost projects expense impact over 10 years with compounding
func Calculate10YearCost(initialValue, expenseRatio, expectedReturn decimal.Decimal) decimal.Decimal {
	// Compare growth with and without expenses
	years := 10

	// Net return after expenses
	netReturn := expectedReturn.Sub(expenseRatio)

	// Value with full return
	fullGrowth := initialValue
	// Value with expenses
	netGrowth := initialValue

	one := decimal.NewFromInt(1)
	hundred := decimal.NewFromInt(100)

	for i := 0; i < years; i++ {
		fullGrowth = fullGrowth.Mul(one.Add(expectedReturn.Div(hundred)))
		netGrowth = netGrowth.Mul(one.Add(netReturn.Div(hundred)))
	}

	// Cost is the difference
	return fullGrowth.Sub(netGrowth).Round(2)
}
