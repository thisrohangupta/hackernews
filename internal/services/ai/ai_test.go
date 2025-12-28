package ai

import (
	"testing"
	"time"

	"github.com/findosh/truenorth/internal/models"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestNewService(t *testing.T) {
	svc := NewService(nil)
	if svc == nil {
		t.Fatal("Expected service to be created")
	}
}

func TestNewServiceWithConfig(t *testing.T) {
	cfg := DefaultConfig()
	cfg.APIKey = "test-key"
	svc := NewService(cfg)

	if svc == nil {
		t.Fatal("Expected service to be created")
	}
}

func TestModelRouter_ClassifyIntent(t *testing.T) {
	router := NewModelRouter(DefaultConfig())

	tests := []struct {
		query    string
		expected QueryIntent
	}{
		{"What is a stock?", IntentSimple},
		{"What is my portfolio allocation?", IntentAnalytical},
		{"How can I optimize my taxes?", IntentTax},
		{"What is my risk exposure?", IntentRisk},
		{"What will my retirement portfolio look like?", IntentProjection},
		{"Show me historical trends for my portfolio", IntentResearch},
		{"Compare VOO vs VTI", IntentComparison},
		{"Should I buy AAPL?", IntentUnsupported},
		{"Recommend me stocks to buy", IntentUnsupported},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			intent := router.ClassifyIntent(tt.query)
			if intent != tt.expected {
				t.Errorf("ClassifyIntent(%q) = %s, want %s", tt.query, intent, tt.expected)
			}
		})
	}
}

func TestModelRouter_SelectModel(t *testing.T) {
	cfg := DefaultConfig()
	router := NewModelRouter(cfg)

	tests := []struct {
		intent   QueryIntent
		query    string
		expected Model
	}{
		{IntentSimple, "short query", ModelHaiku},
		{IntentAnalytical, "short query", ModelSonnet},
		{IntentTax, "short query", ModelSonnet},
	}

	for _, tt := range tests {
		t.Run(string(tt.intent), func(t *testing.T) {
			model := router.SelectModel(tt.intent, tt.query)
			if model != tt.expected {
				t.Errorf("SelectModel(%s) = %s, want %s", tt.intent, model, tt.expected)
			}
		})
	}
}

func TestModelRouter_AnalyzeComplexity(t *testing.T) {
	router := NewModelRouter(DefaultConfig())

	tests := []struct {
		query      string
		complexity string
	}{
		{"what is VOO?", "simple"},
		{"How is my portfolio performing compared to the S&P 500 benchmark?", "moderate"},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			result := router.AnalyzeComplexity(tt.query)
			if result.Complexity != tt.complexity {
				t.Errorf("AnalyzeComplexity(%q).Complexity = %s, want %s",
					tt.query, result.Complexity, tt.complexity)
			}
		})
	}
}

func TestQueryCache(t *testing.T) {
	cache := NewQueryCache(1*time.Hour, 0.92)

	// Test set and get
	response := &Response{
		ID:   "test-id",
		Text: "Test response",
	}

	cache.Set("test query", "portfolio-1", response)

	// Should get exact match
	cached := cache.Get("test query", "portfolio-1")
	if cached == nil {
		t.Fatal("Expected cached response")
	}
	if cached.Text != response.Text {
		t.Errorf("Cached text = %s, want %s", cached.Text, response.Text)
	}

	// Should not get for different portfolio
	cached = cache.Get("test query", "portfolio-2")
	if cached != nil {
		t.Error("Should not get cached response for different portfolio")
	}
}

func TestQueryCache_Invalidate(t *testing.T) {
	cache := NewQueryCache(1*time.Hour, 0.92)

	response := &Response{ID: "test-id", Text: "Test response"}
	cache.Set("query 1", "portfolio-1", response)
	cache.Set("query 2", "portfolio-1", response)
	cache.Set("query 3", "portfolio-2", response)

	cache.Invalidate("portfolio-1")

	if cache.Get("query 1", "portfolio-1") != nil {
		t.Error("Cache should be invalidated for portfolio-1")
	}
	if cache.Get("query 3", "portfolio-2") == nil {
		t.Error("Cache should still exist for portfolio-2")
	}
}

func TestQueryCache_Stats(t *testing.T) {
	cache := NewQueryCache(1*time.Hour, 0.92)

	response := &Response{ID: "test-id", Text: "Test response"}
	cache.Set("query 1", "portfolio-1", response)
	cache.Set("query 2", "portfolio-1", response)

	stats := cache.Stats()
	if stats["entries"].(int) != 2 {
		t.Errorf("Expected 2 entries, got %v", stats["entries"])
	}
}

func TestAuditLogger(t *testing.T) {
	logger := NewAuditLogger(true)

	query := &Query{
		ID:     "query-1",
		UserID: "user-1",
		Text:   "Test query",
	}

	response := &Response{
		ID:         "response-1",
		QueryID:    "query-1",
		Intent:     IntentSimple,
		Model:      ModelHaiku,
		TokensUsed: TokenUsage{Input: 100, Output: 50, Total: 150},
	}

	logger.Log(query, response)

	entries := logger.GetEntries("user-1", time.Now().Add(-1*time.Hour), 10)
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}

	if entries[0].QueryID != "query-1" {
		t.Errorf("QueryID = %s, want query-1", entries[0].QueryID)
	}
}

func TestAuditLogger_Stats(t *testing.T) {
	logger := NewAuditLogger(true)

	// Log some entries
	for i := 0; i < 5; i++ {
		query := &Query{ID: "q", UserID: "user-1", Text: "test"}
		response := &Response{
			ID:         "r",
			Intent:     IntentSimple,
			Model:      ModelHaiku,
			TokensUsed: TokenUsage{Total: 100},
		}
		logger.Log(query, response)
	}

	stats := logger.GetStats(time.Now().Add(-1 * time.Hour))

	if stats["total_queries"].(int) != 5 {
		t.Errorf("Expected 5 queries, got %v", stats["total_queries"])
	}

	if stats["total_tokens"].(int) != 500 {
		t.Errorf("Expected 500 tokens, got %v", stats["total_tokens"])
	}
}

func TestTaxOptimizer_AnalyzeTaxOpportunities(t *testing.T) {
	optimizer := NewTaxOptimizer()

	portfolio := &models.Portfolio{
		ID:         uuid.New(),
		TotalValue: decimal.NewFromInt(1000000),
		Holdings: []models.Holding{
			{
				Ticker:      "AAPL",
				Name:        "Apple Inc",
				MarketValue: decimal.NewFromInt(100000),
				CostBasis:   decimal.NewFromInt(80000), // Gain
				AssetClass:  models.AssetClassEquity,
			},
			{
				Ticker:      "TSLA",
				Name:        "Tesla Inc",
				MarketValue: decimal.NewFromInt(50000),
				CostBasis:   decimal.NewFromInt(70000), // Loss
				AssetClass:  models.AssetClassEquity,
			},
			{
				Ticker:      "NVDA",
				Name:        "NVIDIA Corp",
				MarketValue: decimal.NewFromInt(30000),
				CostBasis:   decimal.NewFromInt(45000), // Loss
				AssetClass:  models.AssetClassEquity,
			},
		},
	}

	summary := optimizer.AnalyzeTaxOpportunities(portfolio)

	if summary == nil {
		t.Fatal("Expected summary")
	}

	// Should detect gains
	if summary.TotalUnrealizedGains.IsZero() {
		t.Error("Expected unrealized gains")
	}

	// Should detect losses
	if summary.TotalUnrealizedLosses.IsZero() {
		t.Error("Expected unrealized losses")
	}

	// Should find harvest opportunities
	if len(summary.Opportunities) == 0 {
		t.Error("Expected harvest opportunities")
	}

	// Should have disclaimers
	if len(summary.Disclaimers) == 0 {
		t.Error("Expected disclaimers")
	}

	// Should have recommendations
	if len(summary.Recommendations) == 0 {
		t.Error("Expected recommendations")
	}
}

func TestTaxOptimizer_NilPortfolio(t *testing.T) {
	optimizer := NewTaxOptimizer()

	summary := optimizer.AnalyzeTaxOpportunities(nil)

	if summary == nil {
		t.Fatal("Expected summary even for nil portfolio")
	}

	if len(summary.Opportunities) != 0 {
		t.Error("Expected no opportunities for nil portfolio")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.DefaultModel != ModelSonnet {
		t.Errorf("DefaultModel = %s, want %s", cfg.DefaultModel, ModelSonnet)
	}

	if cfg.SimpleModel != ModelHaiku {
		t.Errorf("SimpleModel = %s, want %s", cfg.SimpleModel, ModelHaiku)
	}

	if cfg.Temperature > 0.5 {
		t.Error("Temperature should be low for financial accuracy")
	}

	if !cfg.EnableDisclaimers {
		t.Error("Disclaimers should be enabled by default")
	}

	if !cfg.EnableAuditLog {
		t.Error("Audit log should be enabled by default")
	}
}

func TestIntentModelRouting(t *testing.T) {
	// Verify all intents have routing
	intents := []QueryIntent{
		IntentSimple,
		IntentAnalytical,
		IntentTax,
		IntentResearch,
		IntentComparison,
		IntentProjection,
		IntentRisk,
		IntentCompliance,
		IntentUnsupported,
	}

	for _, intent := range intents {
		if _, ok := IntentModelRouting[intent]; !ok {
			t.Errorf("Missing model routing for intent: %s", intent)
		}
	}
}

func TestModelCosts(t *testing.T) {
	// Verify cost structure makes sense
	haikuCost := ModelCosts[ModelHaiku]
	sonnetCost := ModelCosts[ModelSonnet]
	opusCost := ModelCosts[ModelOpus]

	// Haiku should be cheapest
	if haikuCost.Input >= sonnetCost.Input {
		t.Error("Haiku should be cheaper than Sonnet")
	}

	// Opus should be most expensive
	if opusCost.Input <= sonnetCost.Input {
		t.Error("Opus should be more expensive than Sonnet")
	}
}
