package ai

import (
	"fmt"
	"sort"
	"time"

	"github.com/findosh/truenorth/internal/models"
	"github.com/shopspring/decimal"
)

// TaxOptimizer provides tax-loss harvesting and optimization recommendations
type TaxOptimizer struct {
	// Wash sale window (30 days before and after)
	washSaleWindow time.Duration
	// Short-term vs long-term threshold
	longTermThreshold time.Duration
	// Minimum loss to consider harvesting
	minLossThreshold decimal.Decimal
}

// NewTaxOptimizer creates a new tax optimizer
func NewTaxOptimizer() *TaxOptimizer {
	return &TaxOptimizer{
		washSaleWindow:    30 * 24 * time.Hour,
		longTermThreshold: 365 * 24 * time.Hour,
		minLossThreshold:  decimal.NewFromInt(100), // $100 minimum loss
	}
}

// TaxLotType indicates short-term or long-term holding
type TaxLotType string

const (
	TaxLotShortTerm TaxLotType = "short_term"
	TaxLotLongTerm  TaxLotType = "long_term"
)

// HarvestOpportunity represents a tax-loss harvesting opportunity
type HarvestOpportunity struct {
	Ticker           string          `json:"ticker"`
	Name             string          `json:"name"`
	CurrentValue     decimal.Decimal `json:"current_value"`
	CostBasis        decimal.Decimal `json:"cost_basis"`
	UnrealizedLoss   decimal.Decimal `json:"unrealized_loss"`
	LossPercent      decimal.Decimal `json:"loss_percent"`
	LotType          TaxLotType      `json:"lot_type"`
	EstimatedSavings decimal.Decimal `json:"estimated_savings"`
	Alternatives     []string        `json:"alternatives"` // Similar holdings to avoid wash sale
	WashSaleRisk     bool            `json:"wash_sale_risk"`
	Notes            string          `json:"notes"`
}

// TaxSummary provides overall tax optimization summary
type TaxSummary struct {
	TotalUnrealizedGains  decimal.Decimal      `json:"total_unrealized_gains"`
	TotalUnrealizedLosses decimal.Decimal      `json:"total_unrealized_losses"`
	NetUnrealized         decimal.Decimal      `json:"net_unrealized"`
	ShortTermGains        decimal.Decimal      `json:"short_term_gains"`
	ShortTermLosses       decimal.Decimal      `json:"short_term_losses"`
	LongTermGains         decimal.Decimal      `json:"long_term_gains"`
	LongTermLosses        decimal.Decimal      `json:"long_term_losses"`
	HarvestableAmount     decimal.Decimal      `json:"harvestable_amount"`
	EstimatedTaxSavings   decimal.Decimal      `json:"estimated_tax_savings"`
	Opportunities         []HarvestOpportunity `json:"opportunities"`
	Recommendations       []string             `json:"recommendations"`
	Disclaimers           []string             `json:"disclaimers"`
}

// AnalyzeTaxOpportunities identifies tax-loss harvesting opportunities
func (t *TaxOptimizer) AnalyzeTaxOpportunities(portfolio *models.Portfolio) *TaxSummary {
	summary := &TaxSummary{
		Opportunities:   make([]HarvestOpportunity, 0),
		Recommendations: make([]string, 0),
		Disclaimers: []string{
			"This analysis is for educational purposes only and does not constitute tax advice.",
			"Consult a qualified tax professional before making tax-related decisions.",
			"Tax implications vary based on individual circumstances.",
			"Wash sale rules may affect the deductibility of losses.",
		},
	}

	if portfolio == nil || len(portfolio.Holdings) == 0 {
		return summary
	}

	// Analyze each holding
	for _, holding := range portfolio.Holdings {
		gainLoss := holding.GainLoss()

		if gainLoss.IsPositive() {
			// Track gains
			summary.TotalUnrealizedGains = summary.TotalUnrealizedGains.Add(gainLoss)
			// Assume long-term for simplicity (in production, would track purchase date)
			summary.LongTermGains = summary.LongTermGains.Add(gainLoss)
		} else if gainLoss.IsNegative() {
			// Track losses
			loss := gainLoss.Abs()
			summary.TotalUnrealizedLosses = summary.TotalUnrealizedLosses.Add(loss)
			summary.LongTermLosses = summary.LongTermLosses.Add(loss)

			// Check if harvestable
			if loss.GreaterThanOrEqual(t.minLossThreshold) {
				opp := t.createHarvestOpportunity(holding, loss, portfolio)
				summary.Opportunities = append(summary.Opportunities, opp)
				summary.HarvestableAmount = summary.HarvestableAmount.Add(loss)
			}
		}
	}

	// Calculate net position
	summary.NetUnrealized = summary.TotalUnrealizedGains.Sub(summary.TotalUnrealizedLosses)

	// Estimate tax savings (simplified: assume 24% marginal rate for short-term, 15% for long-term)
	shortTermRate := decimal.NewFromFloat(0.24)
	longTermRate := decimal.NewFromFloat(0.15)

	shortTermSavings := summary.ShortTermLosses.Mul(shortTermRate)
	longTermSavings := summary.LongTermLosses.Mul(longTermRate)
	summary.EstimatedTaxSavings = shortTermSavings.Add(longTermSavings).Round(2)

	// Sort opportunities by loss amount (largest first)
	sort.Slice(summary.Opportunities, func(i, j int) bool {
		return summary.Opportunities[i].UnrealizedLoss.GreaterThan(summary.Opportunities[j].UnrealizedLoss)
	})

	// Generate recommendations
	summary.Recommendations = t.generateRecommendations(summary)

	return summary
}

// createHarvestOpportunity builds a harvest opportunity for a holding
func (t *TaxOptimizer) createHarvestOpportunity(holding models.Holding, loss decimal.Decimal, portfolio *models.Portfolio) HarvestOpportunity {
	lossPercent := decimal.Zero
	if !holding.CostBasis.IsZero() {
		lossPercent = loss.Div(holding.CostBasis).Mul(decimal.NewFromInt(100)).Round(2)
	}

	// Estimate savings (assume 20% blended rate)
	estimatedSavings := loss.Mul(decimal.NewFromFloat(0.20)).Round(2)

	// Find similar alternatives to avoid wash sale
	alternatives := t.findAlternatives(holding, portfolio)

	// Check wash sale risk
	washSaleRisk := len(alternatives) > 0 && t.hasRelatedHoldings(holding, portfolio)

	notes := t.generateNotes(holding, loss, lossPercent, washSaleRisk)

	return HarvestOpportunity{
		Ticker:           holding.Ticker,
		Name:             holding.Name,
		CurrentValue:     holding.MarketValue,
		CostBasis:        holding.CostBasis,
		UnrealizedLoss:   loss,
		LossPercent:      lossPercent,
		LotType:          TaxLotLongTerm, // Simplified assumption
		EstimatedSavings: estimatedSavings,
		Alternatives:     alternatives,
		WashSaleRisk:     washSaleRisk,
		Notes:            notes,
	}
}

// findAlternatives suggests similar investments to maintain exposure
func (t *TaxOptimizer) findAlternatives(holding models.Holding, portfolio *models.Portfolio) []string {
	alternatives := make([]string, 0)

	// Suggest alternatives based on asset class
	switch holding.AssetClass {
	case models.AssetClassEquity:
		if holding.Geography == "US" {
			// Suggest different but similar US equity ETFs
			alternatives = append(alternatives,
				"Consider a different S&P 500 ETF (VOO → IVV or SPY)",
				"Total market ETF as alternative (VTI, ITOT)",
			)
		} else {
			alternatives = append(alternatives,
				"Consider equivalent international ETF from different provider",
			)
		}
	case models.AssetClassFixedIncome:
		alternatives = append(alternatives,
			"Consider bond ETF from different provider",
			"Treasury ETF as alternative to corporate bonds",
		)
	default:
		alternatives = append(alternatives,
			"Consult advisor for suitable alternatives",
		)
	}

	return alternatives
}

// hasRelatedHoldings checks if portfolio has substantially identical securities
func (t *TaxOptimizer) hasRelatedHoldings(holding models.Holding, portfolio *models.Portfolio) bool {
	// Check for same asset class + sector + geography
	for _, h := range portfolio.Holdings {
		if h.Ticker == holding.Ticker {
			continue
		}
		if h.AssetClass == holding.AssetClass &&
			h.Sector == holding.Sector &&
			h.Geography == holding.Geography {
			return true
		}
	}
	return false
}

// generateNotes creates explanatory notes for the opportunity
func (t *TaxOptimizer) generateNotes(holding models.Holding, loss, lossPercent decimal.Decimal, washSaleRisk bool) string {
	notes := fmt.Sprintf("%s is down %s%% from cost basis. ", holding.Ticker, lossPercent.StringFixed(1))

	if lossPercent.GreaterThan(decimal.NewFromInt(20)) {
		notes += "Significant loss may warrant harvesting. "
	}

	if washSaleRisk {
		notes += "CAUTION: Wash sale risk if you have similar holdings or plan to repurchase within 30 days."
	} else {
		notes += "Consider replacing with similar but not substantially identical investment."
	}

	return notes
}

// generateRecommendations creates actionable recommendations
func (t *TaxOptimizer) generateRecommendations(summary *TaxSummary) []string {
	recs := make([]string, 0)

	// Check for significant harvestable losses
	threshold := decimal.NewFromInt(1000)
	if summary.HarvestableAmount.GreaterThan(threshold) {
		recs = append(recs, fmt.Sprintf(
			"You have approximately $%s in harvestable losses that could offset gains or income.",
			summary.HarvestableAmount.StringFixed(0),
		))
	}

	// Check for gain/loss offset opportunity
	if summary.TotalUnrealizedGains.GreaterThan(decimal.Zero) &&
		summary.TotalUnrealizedLosses.GreaterThan(decimal.Zero) {
		recs = append(recs,
			"Consider pairing loss harvesting with gain realization to optimize tax impact.",
		)
	}

	// Check for concentrated losses
	if len(summary.Opportunities) > 0 {
		topOpp := summary.Opportunities[0]
		if topOpp.LossPercent.GreaterThan(decimal.NewFromInt(30)) {
			recs = append(recs, fmt.Sprintf(
				"%s has a significant loss (%s%%). Review if this aligns with your investment thesis.",
				topOpp.Ticker, topOpp.LossPercent.StringFixed(1),
			))
		}
	}

	// Year-end reminder
	now := time.Now()
	if now.Month() >= 10 { // October or later
		recs = append(recs,
			"Year-end is approaching. Consider tax-loss harvesting before December 31 for current tax year benefits.",
		)
	}

	// Annual loss limit reminder
	recs = append(recs,
		"Remember: Capital losses can offset capital gains, plus up to $3,000 of ordinary income annually.",
	)

	if len(recs) == 0 {
		recs = append(recs,
			"No significant tax optimization opportunities identified at this time.",
		)
	}

	return recs
}

// CalculateWashSaleWindow returns the wash sale exclusion period
func (t *TaxOptimizer) CalculateWashSaleWindow(saleDate time.Time) (time.Time, time.Time) {
	start := saleDate.Add(-30 * 24 * time.Hour)
	end := saleDate.Add(30 * 24 * time.Hour)
	return start, end
}
