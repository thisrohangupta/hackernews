package ai

import (
	"strings"
)

// ModelRouter handles intent classification and model selection
type ModelRouter struct {
	cfg *Config
}

// NewModelRouter creates a new model router
func NewModelRouter(cfg *Config) *ModelRouter {
	return &ModelRouter{cfg: cfg}
}

// ClassifyIntent determines the type of query
func (r *ModelRouter) ClassifyIntent(query string) QueryIntent {
	query = strings.ToLower(query)

	// Check for blocked content first
	if r.isBlockedQuery(query) {
		return IntentUnsupported
	}

	// Tax-related queries
	if containsAny(query, []string{
		"tax", "taxes", "tax-loss", "harvest", "wash sale",
		"capital gain", "capital loss", "1099", "cost basis",
		"short-term", "long-term gain",
	}) {
		return IntentTax
	}

	// Risk assessment
	if containsAny(query, []string{
		"risk", "volatility", "drawdown", "beta", "sharpe",
		"sortino", "var", "value at risk", "exposure",
		"concentrated", "diversif",
	}) {
		return IntentRisk
	}

	// Projections and scenarios
	if containsAny(query, []string{
		"project", "forecast", "predict", "future", "scenario",
		"what if", "monte carlo", "retirement", "goal",
		"will i have", "can i afford",
	}) {
		return IntentProjection
	}

	// Research queries
	if containsAny(query, []string{
		"research", "analyze", "deep dive", "explain why",
		"compare to market", "versus benchmark", "historical",
		"trend", "pattern",
	}) {
		return IntentResearch
	}

	// Comparison queries
	if containsAny(query, []string{
		"compare", "versus", "vs", "better than", "difference between",
		"which is", "should i choose",
	}) {
		return IntentComparison
	}

	// Analytical queries (portfolio-specific)
	if containsAny(query, []string{
		"portfolio", "allocation", "holdings", "position",
		"performance", "return", "my", "how am i",
		"rebalance", "weight",
	}) {
		return IntentAnalytical
	}

	// Simple queries (definitions, basic info)
	if containsAny(query, []string{
		"what is", "what are", "define", "explain", "how does",
		"tell me about", "meaning of",
	}) {
		return IntentSimple
	}

	// Default to analytical for portfolio context
	return IntentAnalytical
}

// isBlockedQuery checks if the query requests blocked content
func (r *ModelRouter) isBlockedQuery(query string) bool {
	blockedPatterns := []string{
		// Specific recommendations
		"should i buy", "should i sell", "buy or sell",
		"is it a good time to", "when should i",
		"recommend me", "what should i invest in",
		"pick stocks for me", "best stocks to buy",

		// Guaranteed returns
		"guaranteed", "risk-free return", "can't lose",
		"will definitely", "100% certain",

		// Market timing
		"when will the market", "will the stock go up",
		"price target", "where will",

		// Insider information
		"insider", "non-public", "confidential information",
	}

	for _, pattern := range blockedPatterns {
		if strings.Contains(query, pattern) {
			return true
		}
	}

	return false
}

// SelectModel chooses the appropriate model for a query
func (r *ModelRouter) SelectModel(intent QueryIntent, query string) Model {
	// Check if query is complex (long, multiple questions)
	isComplex := len(query) > 500 || strings.Count(query, "?") > 1

	// Route based on intent
	model, ok := IntentModelRouting[intent]
	if !ok {
		model = r.cfg.DefaultModel
	}

	// Upgrade to complex model if needed
	if isComplex && model == ModelHaiku {
		model = r.cfg.ComplexModel
	}

	return model
}

// EstimateTokens provides a rough token count estimate
func (r *ModelRouter) EstimateTokens(text string) int {
	// Rough estimate: ~4 characters per token for English
	return len(text) / 4
}

// EstimateCost calculates approximate cost for a query
func (r *ModelRouter) EstimateCost(model Model, inputTokens, outputTokens int) float64 {
	costs, ok := ModelCosts[model]
	if !ok {
		return 0
	}

	inputCost := float64(inputTokens) / 1_000_000 * costs.Input
	outputCost := float64(outputTokens) / 1_000_000 * costs.Output

	return inputCost + outputCost
}

// containsAny checks if text contains any of the patterns
func containsAny(text string, patterns []string) bool {
	for _, p := range patterns {
		if strings.Contains(text, p) {
			return true
		}
	}
	return false
}

// QueryComplexity provides additional complexity analysis
type QueryComplexity struct {
	TokenEstimate int
	QuestionCount int
	HasNumbers    bool
	HasTickers    bool
	Complexity    string // "simple", "moderate", "complex"
}

// AnalyzeComplexity provides detailed query complexity analysis
func (r *ModelRouter) AnalyzeComplexity(query string) QueryComplexity {
	qc := QueryComplexity{
		TokenEstimate: r.EstimateTokens(query),
		QuestionCount: strings.Count(query, "?"),
	}

	// Check for numbers
	for _, c := range query {
		if c >= '0' && c <= '9' {
			qc.HasNumbers = true
			break
		}
	}

	// Check for potential tickers (uppercase words 2-5 chars)
	words := strings.Fields(query)
	for _, w := range words {
		if len(w) >= 2 && len(w) <= 5 && w == strings.ToUpper(w) {
			qc.HasTickers = true
			break
		}
	}

	// Determine complexity
	if qc.TokenEstimate > 200 || qc.QuestionCount > 2 {
		qc.Complexity = "complex"
	} else if qc.TokenEstimate > 50 || qc.QuestionCount > 1 || qc.HasNumbers {
		qc.Complexity = "moderate"
	} else {
		qc.Complexity = "simple"
	}

	return qc
}
