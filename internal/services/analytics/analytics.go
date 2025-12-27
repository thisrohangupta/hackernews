package analytics

import (
	"math"
	"sort"
	"time"

	"github.com/findosh/truenorth/internal/models"
	"github.com/shopspring/decimal"
)

// Service provides portfolio analytics calculations
type Service struct {
	// Historical data cache (in production, this would come from database/API)
	priceCache map[string][]models.PriceHistory
}

// NewService creates a new analytics service
func NewService() *Service {
	return &Service{
		priceCache: make(map[string][]models.PriceHistory),
	}
}

// CalculatePortfolioPerformance calculates performance for a portfolio over a period
func (s *Service) CalculatePortfolioPerformance(portfolio *models.Portfolio, period string) *models.PortfolioPerformance {
	if portfolio == nil || len(portfolio.Holdings) == 0 {
		return nil
	}

	// Use historical asset class returns for estimation
	totalValue := portfolio.TotalValue
	startValue := s.estimateHistoricalValue(portfolio, period)

	// Calculate total return
	totalReturn := decimal.Zero
	if !startValue.IsZero() {
		totalReturn = totalValue.Sub(startValue).Div(startValue).Mul(decimal.NewFromInt(100))
	}

	// Calculate weighted metrics across holdings
	var weightedVolatility decimal.Decimal
	var weightedReturn decimal.Decimal

	for _, h := range portfolio.Holdings {
		if totalValue.IsZero() {
			continue
		}
		weight := h.MarketValue.Div(totalValue)
		stats := models.AssetClassReturns[h.AssetClass]

		weightedReturn = weightedReturn.Add(weight.Mul(stats.Average))
		weightedVolatility = weightedVolatility.Add(weight.Mul(stats.Volatility))
	}

	// Calculate Sharpe ratio
	sharpeRatio := decimal.Zero
	if !weightedVolatility.IsZero() {
		excessReturn := weightedReturn.Sub(models.RiskFreeRate.Mul(decimal.NewFromInt(100)))
		sharpeRatio = excessReturn.Div(weightedVolatility).Round(2)
	}

	// Estimate max drawdown based on asset mix
	maxDrawdown := s.estimateMaxDrawdown(portfolio)

	// Calculate holding contributions
	holdingPerfs := make([]models.HoldingPerformance, 0, len(portfolio.Holdings))
	for _, h := range portfolio.Holdings {
		stats := models.AssetClassReturns[h.AssetClass]
		holdingReturn := stats.Average

		// Estimate contribution to portfolio return
		weight := decimal.Zero
		if !totalValue.IsZero() {
			weight = h.MarketValue.Div(totalValue).Mul(decimal.NewFromInt(100))
		}
		contribution := weight.Mul(holdingReturn).Div(decimal.NewFromInt(100))

		holdingPerfs = append(holdingPerfs, models.HoldingPerformance{
			Ticker:          h.Ticker,
			Name:            h.Name,
			StartValue:      h.MarketValue, // Simplified
			EndValue:        h.MarketValue,
			TotalReturn:     holdingReturn,
			ContributionPct: contribution.Round(2),
			Weight:          weight.Round(2),
		})
	}

	return &models.PortfolioPerformance{
		PortfolioID:      portfolio.ID.String(),
		Period:           period,
		StartValue:       startValue,
		EndValue:         totalValue,
		TotalReturn:      totalReturn.Round(2),
		AnnualizedReturn: weightedReturn.Round(2),
		Volatility:       weightedVolatility.Round(2),
		SharpeRatio:      sharpeRatio,
		MaxDrawdown:      maxDrawdown,
		Holdings:         holdingPerfs,
	}
}

// CalculateRiskRewardMatrix builds the full risk-reward analysis
func (s *Service) CalculateRiskRewardMatrix(portfolio *models.Portfolio) *models.RiskRewardMatrix {
	if portfolio == nil {
		return nil
	}

	matrix := &models.RiskRewardMatrix{
		PortfolioID:  portfolio.ID.String(),
		CalculatedAt: time.Now().Format(time.RFC3339),
		Period:       "5y",
		ByAssetClass: make(map[models.AssetClass]models.RiskRewardMetrics),
	}

	// Calculate portfolio-level metrics
	matrix.Portfolio = s.calculatePortfolioMetrics(portfolio)

	// Calculate by asset class
	assetClassHoldings := make(map[models.AssetClass][]models.Holding)
	for _, h := range portfolio.Holdings {
		assetClassHoldings[h.AssetClass] = append(assetClassHoldings[h.AssetClass], h)
	}

	for class, holdings := range assetClassHoldings {
		matrix.ByAssetClass[class] = s.calculateAssetClassMetrics(class, holdings, portfolio.TotalValue)
	}

	// Calculate individual holding metrics
	matrix.Holdings = s.calculateHoldingMetrics(portfolio)

	// Perform quadrant analysis
	matrix.Quadrants = s.analyzeQuadrants(matrix.Holdings)

	return matrix
}

// CalculateExpenses analyzes portfolio expenses
func (s *Service) CalculateExpenses(portfolio *models.Portfolio) *models.PortfolioExpenses {
	if portfolio == nil {
		return nil
	}

	expenses := &models.PortfolioExpenses{
		ByAssetClass: make(map[models.AssetClass]models.AssetClassExpenses),
	}

	totalWeightedRatio := decimal.Zero
	var highestExpenses []models.HoldingExpense

	// Group by asset class
	assetClassData := make(map[models.AssetClass]struct {
		totalValue   decimal.Decimal
		totalExpense decimal.Decimal
		count        int
	})

	for _, h := range portfolio.Holdings {
		expenseRatio := models.GetExpenseRatio(h.Ticker, h.AssetClass)
		annualCost := models.CalculateAnnualExpense(h.MarketValue, expenseRatio)

		// Add to totals
		expenses.TotalAnnualExpenses = expenses.TotalAnnualExpenses.Add(annualCost)

		// Weight for portfolio expense ratio
		if !portfolio.TotalValue.IsZero() {
			weight := h.MarketValue.Div(portfolio.TotalValue)
			totalWeightedRatio = totalWeightedRatio.Add(weight.Mul(expenseRatio))
		}

		// Track by asset class
		data := assetClassData[h.AssetClass]
		data.totalValue = data.totalValue.Add(h.MarketValue)
		data.totalExpense = data.totalExpense.Add(annualCost)
		data.count++
		assetClassData[h.AssetClass] = data

		// Track highest expense holdings
		highestExpenses = append(highestExpenses, models.HoldingExpense{
			Ticker:       h.Ticker,
			Name:         h.Name,
			ExpenseRatio: expenseRatio,
			MarketValue:  h.MarketValue,
			AnnualCost:   annualCost,
			AssetClass:   h.AssetClass,
		})
	}

	expenses.WeightedExpenseRatio = totalWeightedRatio.Round(4)

	// Build asset class breakdown
	for class, data := range assetClassData {
		avgRatio := decimal.Zero
		if data.count > 0 && !data.totalValue.IsZero() {
			avgRatio = data.totalExpense.Div(data.totalValue).Mul(decimal.NewFromInt(100))
		}

		expenses.ByAssetClass[class] = models.AssetClassExpenses{
			AvgExpenseRatio: avgRatio.Round(4),
			TotalValue:      data.totalValue,
			AnnualCost:      data.totalExpense,
			HoldingCount:    data.count,
		}
	}

	// Sort and keep top 10 most expensive
	sort.Slice(highestExpenses, func(i, j int) bool {
		return highestExpenses[i].ExpenseRatio.GreaterThan(highestExpenses[j].ExpenseRatio)
	})
	if len(highestExpenses) > 10 {
		highestExpenses = highestExpenses[:10]
	}
	expenses.HighestExpense = highestExpenses

	// Compare to benchmark
	expenses.VsBenchmark = expenses.WeightedExpenseRatio.Sub(models.BenchmarkExpenseRatio).Round(4)

	// Calculate 10-year cost
	expectedReturn := decimal.NewFromFloat(7.0) // 7% expected market return
	expenses.TenYearCost = models.Calculate10YearCost(
		portfolio.TotalValue,
		expenses.WeightedExpenseRatio,
		expectedReturn,
	)

	// Estimate potential savings (reducing to benchmark level)
	if expenses.WeightedExpenseRatio.GreaterThan(models.BenchmarkExpenseRatio) {
		savingsRatio := expenses.WeightedExpenseRatio.Sub(models.BenchmarkExpenseRatio)
		expenses.PotentialSavings = models.CalculateAnnualExpense(portfolio.TotalValue, savingsRatio)
	}

	return expenses
}

// Helper methods

func (s *Service) estimateHistoricalValue(portfolio *models.Portfolio, period string) decimal.Decimal {
	// Estimate historical value based on average returns
	// In production, this would use actual historical data

	duration := models.GetPeriodDuration(period)
	years := decimal.NewFromFloat(duration.Hours() / 24 / 365)

	if years.IsZero() {
		return portfolio.TotalValue
	}

	// Calculate weighted average return
	weightedReturn := decimal.Zero
	for _, h := range portfolio.Holdings {
		if portfolio.TotalValue.IsZero() {
			continue
		}
		weight := h.MarketValue.Div(portfolio.TotalValue)
		stats := models.AssetClassReturns[h.AssetClass]
		weightedReturn = weightedReturn.Add(weight.Mul(stats.Average))
	}

	// Back-calculate starting value
	// EndValue = StartValue * (1 + return)^years
	// StartValue = EndValue / (1 + return)^years
	annualReturn := weightedReturn.Div(decimal.NewFromInt(100))
	multiplier := math.Pow(1+annualReturn.InexactFloat64(), years.InexactFloat64())

	if multiplier == 0 {
		return portfolio.TotalValue
	}

	startValue := portfolio.TotalValue.Div(decimal.NewFromFloat(multiplier))
	return startValue.Round(2)
}

func (s *Service) estimateMaxDrawdown(portfolio *models.Portfolio) decimal.Decimal {
	// Estimate max drawdown based on worst year of each asset class
	weightedDrawdown := decimal.Zero

	for _, h := range portfolio.Holdings {
		if portfolio.TotalValue.IsZero() {
			continue
		}
		weight := h.MarketValue.Div(portfolio.TotalValue)
		stats := models.AssetClassReturns[h.AssetClass]
		weightedDrawdown = weightedDrawdown.Add(weight.Mul(stats.WorstYear))
	}

	return weightedDrawdown.Round(2)
}

func (s *Service) calculatePortfolioMetrics(portfolio *models.Portfolio) models.RiskRewardMetrics {
	metrics := models.RiskRewardMetrics{}

	if portfolio.TotalValue.IsZero() || len(portfolio.Holdings) == 0 {
		return metrics
	}

	// Calculate weighted averages
	var totalReturn, volatility, maxDrawdown, beta decimal.Decimal

	for _, h := range portfolio.Holdings {
		weight := h.MarketValue.Div(portfolio.TotalValue)
		stats := models.AssetClassReturns[h.AssetClass]

		totalReturn = totalReturn.Add(weight.Mul(stats.Average))
		volatility = volatility.Add(weight.Mul(stats.Volatility))
		maxDrawdown = maxDrawdown.Add(weight.Mul(stats.WorstYear))
	}

	// Estimate beta (simplified - equity-weighted)
	equityWeight := decimal.Zero
	for _, h := range portfolio.Holdings {
		if h.AssetClass == models.AssetClassEquity {
			equityWeight = equityWeight.Add(h.MarketValue.Div(portfolio.TotalValue))
		}
	}
	beta = equityWeight // Beta roughly equals equity exposure for diversified portfolio

	metrics.ExpectedReturn = totalReturn.Round(2)
	metrics.AnnualizedReturn = totalReturn.Round(2)
	metrics.Volatility = volatility.Round(2)
	metrics.MaxDrawdown = maxDrawdown.Round(2)
	metrics.Beta = beta.Round(2)

	// Sharpe ratio
	if !volatility.IsZero() {
		excessReturn := totalReturn.Sub(models.RiskFreeRate.Mul(decimal.NewFromInt(100)))
		metrics.SharpeRatio = excessReturn.Div(volatility).Round(2)
	}

	// Sortino ratio (using downside deviation estimate)
	downsideVol := volatility.Mul(decimal.NewFromFloat(0.7)) // Approximate
	metrics.DownsideDeviation = downsideVol.Round(2)
	if !downsideVol.IsZero() {
		excessReturn := totalReturn.Sub(models.RiskFreeRate.Mul(decimal.NewFromInt(100)))
		metrics.SortinoRatio = excessReturn.Div(downsideVol).Round(2)
	}

	// Calmar ratio
	if !maxDrawdown.IsZero() {
		metrics.CalmarRatio = totalReturn.Div(maxDrawdown.Abs()).Round(2)
	}

	// Treynor ratio
	if !beta.IsZero() {
		excessReturn := totalReturn.Sub(models.RiskFreeRate.Mul(decimal.NewFromInt(100)))
		metrics.TreynorRatio = excessReturn.Div(beta).Round(2)
	}

	return metrics
}

func (s *Service) calculateAssetClassMetrics(class models.AssetClass, holdings []models.Holding, totalPortfolioValue decimal.Decimal) models.RiskRewardMetrics {
	stats := models.AssetClassReturns[class]

	metrics := models.RiskRewardMetrics{
		ExpectedReturn:   stats.Average,
		AnnualizedReturn: stats.Average,
		Volatility:       stats.Volatility,
		MaxDrawdown:      stats.WorstYear,
	}

	// Sharpe ratio
	if !stats.Volatility.IsZero() {
		excessReturn := stats.Average.Sub(models.RiskFreeRate.Mul(decimal.NewFromInt(100)))
		metrics.SharpeRatio = excessReturn.Div(stats.Volatility).Round(2)
	}

	return metrics
}

func (s *Service) calculateHoldingMetrics(portfolio *models.Portfolio) []models.HoldingRiskReward {
	holdings := make([]models.HoldingRiskReward, 0, len(portfolio.Holdings))

	// Calculate averages for quadrant analysis
	avgReturn := decimal.Zero
	avgVol := decimal.Zero
	count := decimal.NewFromInt(int64(len(portfolio.Holdings)))

	for _, h := range portfolio.Holdings {
		stats := models.AssetClassReturns[h.AssetClass]
		avgReturn = avgReturn.Add(stats.Average)
		avgVol = avgVol.Add(stats.Volatility)
	}

	if !count.IsZero() {
		avgReturn = avgReturn.Div(count)
		avgVol = avgVol.Div(count)
	}

	for _, h := range portfolio.Holdings {
		stats := models.AssetClassReturns[h.AssetClass]

		weight := decimal.Zero
		if !portfolio.TotalValue.IsZero() {
			weight = h.MarketValue.Div(portfolio.TotalValue).Mul(decimal.NewFromInt(100))
		}

		metrics := models.DefaultRiskMetrics(h.AssetClass)
		quadrant := models.CalculateQuadrant(stats.Average, stats.Volatility, avgReturn, avgVol)

		// Calculate risk contribution (simplified)
		riskContrib := weight.Mul(stats.Volatility).Div(decimal.NewFromInt(100))

		holdings = append(holdings, models.HoldingRiskReward{
			Ticker:           h.Ticker,
			Name:             h.Name,
			AssetClass:       h.AssetClass,
			Weight:           weight.Round(2),
			Metrics:          metrics,
			Quadrant:         quadrant,
			RiskContribution: riskContrib.Round(2),
		})
	}

	// Sort by weight descending
	sort.Slice(holdings, func(i, j int) bool {
		return holdings[i].Weight.GreaterThan(holdings[j].Weight)
	})

	// Limit to top 20
	if len(holdings) > 20 {
		holdings = holdings[:20]
	}

	return holdings
}

func (s *Service) analyzeQuadrants(holdings []models.HoldingRiskReward) models.QuadrantAnalysis {
	analysis := models.QuadrantAnalysis{
		Optimal:      make([]string, 0),
		Aggressive:   make([]string, 0),
		Conservative: make([]string, 0),
		Avoid:        make([]string, 0),
	}

	for _, h := range holdings {
		switch h.Quadrant {
		case "optimal":
			analysis.Optimal = append(analysis.Optimal, h.Ticker)
			analysis.OptimalValue = analysis.OptimalValue.Add(h.Weight)
		case "aggressive":
			analysis.Aggressive = append(analysis.Aggressive, h.Ticker)
			analysis.AggressiveValue = analysis.AggressiveValue.Add(h.Weight)
		case "conservative":
			analysis.Conservative = append(analysis.Conservative, h.Ticker)
			analysis.ConservativeValue = analysis.ConservativeValue.Add(h.Weight)
		case "avoid":
			analysis.Avoid = append(analysis.Avoid, h.Ticker)
			analysis.AvoidValue = analysis.AvoidValue.Add(h.Weight)
		}
	}

	return analysis
}

// GenerateTimeSeries creates a historical value time series for charting
func (s *Service) GenerateTimeSeries(portfolio *models.Portfolio, period string) []models.TimeSeriesPoint {
	if portfolio == nil {
		return nil
	}

	endDate := time.Now().UTC()
	startDate := models.GetPeriodStartDate(period)

	// Calculate weighted return for the portfolio
	weightedReturn := decimal.Zero
	for _, h := range portfolio.Holdings {
		if portfolio.TotalValue.IsZero() {
			continue
		}
		weight := h.MarketValue.Div(portfolio.TotalValue)
		stats := models.AssetClassReturns[h.AssetClass]
		weightedReturn = weightedReturn.Add(weight.Mul(stats.Average))
	}

	// Generate monthly points
	points := make([]models.TimeSeriesPoint, 0)
	current := startDate
	totalDays := endDate.Sub(startDate).Hours() / 24

	// Daily return
	dailyReturn := weightedReturn.Div(decimal.NewFromFloat(365)).Div(decimal.NewFromInt(100))

	// Back-calculate starting value
	startValue := s.estimateHistoricalValue(portfolio, period)
	currentValue := startValue

	for current.Before(endDate) {
		points = append(points, models.TimeSeriesPoint{
			Date:  current,
			Value: currentValue.Round(2),
		})

		// Move forward (weekly for long periods, daily for short)
		if totalDays > 365 {
			current = current.AddDate(0, 0, 7)
			weeklyReturn := dailyReturn.Mul(decimal.NewFromInt(7))
			currentValue = currentValue.Mul(decimal.NewFromInt(1).Add(weeklyReturn))
		} else {
			current = current.AddDate(0, 0, 1)
			currentValue = currentValue.Mul(decimal.NewFromInt(1).Add(dailyReturn))
		}
	}

	// Add final point at current value
	points = append(points, models.TimeSeriesPoint{
		Date:  endDate,
		Value: portfolio.TotalValue,
	})

	return points
}
