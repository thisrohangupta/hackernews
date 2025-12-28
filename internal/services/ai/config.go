// Package ai provides AI-powered portfolio analysis using Claude
package ai

import (
	"time"
)

// Model represents available Claude models for routing
type Model string

const (
	// ModelHaiku is fast and cheap for simple queries
	ModelHaiku Model = "claude-3-haiku-20240307"
	// ModelSonnet is balanced for complex analysis
	ModelSonnet Model = "claude-sonnet-4-20250514"
	// ModelOpus is most capable for deep research
	ModelOpus Model = "claude-opus-4-20250514"
)

// TokenCost represents cost per million tokens
type TokenCost struct {
	Input  float64
	Output float64
}

// ModelCosts defines pricing for each model (per million tokens)
var ModelCosts = map[Model]TokenCost{
	ModelHaiku:  {Input: 0.25, Output: 1.25},
	ModelSonnet: {Input: 3.00, Output: 15.00},
	ModelOpus:   {Input: 15.00, Output: 75.00},
}

// Config holds AI service configuration
type Config struct {
	// API configuration
	APIKey     string
	BaseURL    string
	MaxRetries int
	Timeout    time.Duration

	// Model routing
	DefaultModel    Model
	ComplexModel    Model
	SimpleModel     Model
	MaxTokens       int
	Temperature     float64
	EnableStreaming bool

	// Cost management
	DailyTokenBudget    int
	QueryTokenLimit     int
	EnableCostTracking  bool
	CacheEnabled        bool
	CacheTTL            time.Duration
	SemanticCacheThresh float64 // Similarity threshold for cache hits

	// Compliance
	EnableAuditLog     bool
	EnableDisclaimers  bool
	BlockedTopics      []string
	RequireSourceCites bool
}

// DefaultConfig returns production-ready defaults
func DefaultConfig() *Config {
	return &Config{
		BaseURL:    "https://api.anthropic.com/v1",
		MaxRetries: 3,
		Timeout:    30 * time.Second,

		DefaultModel:    ModelSonnet,
		ComplexModel:    ModelSonnet,
		SimpleModel:     ModelHaiku,
		MaxTokens:       4096,
		Temperature:     0.2, // Low temperature for financial accuracy
		EnableStreaming: true,

		DailyTokenBudget:    1000000, // 1M tokens/day per user
		QueryTokenLimit:     8192,
		EnableCostTracking:  true,
		CacheEnabled:        true,
		CacheTTL:            1 * time.Hour,
		SemanticCacheThresh: 0.92, // High similarity for cache hits

		EnableAuditLog:     true,
		EnableDisclaimers:  true,
		RequireSourceCites: true,
		BlockedTopics: []string{
			"specific stock recommendations",
			"guaranteed returns",
			"market timing predictions",
			"insider information",
		},
	}
}

// QueryIntent classifies the type of user query
type QueryIntent string

const (
	IntentSimple      QueryIntent = "simple"      // FAQ, definitions
	IntentAnalytical  QueryIntent = "analytical"  // Portfolio analysis
	IntentTax         QueryIntent = "tax"         // Tax optimization
	IntentResearch    QueryIntent = "research"    // Deep research
	IntentComparison  QueryIntent = "comparison"  // Compare holdings
	IntentProjection  QueryIntent = "projection"  // Future scenarios
	IntentRisk        QueryIntent = "risk"        // Risk assessment
	IntentCompliance  QueryIntent = "compliance"  // Regulatory questions
	IntentUnsupported QueryIntent = "unsupported" // Blocked queries
)

// IntentModelRouting maps intents to appropriate models
var IntentModelRouting = map[QueryIntent]Model{
	IntentSimple:      ModelHaiku,
	IntentAnalytical:  ModelSonnet,
	IntentTax:         ModelSonnet,
	IntentResearch:    ModelSonnet,
	IntentComparison:  ModelHaiku,
	IntentProjection:  ModelSonnet,
	IntentRisk:        ModelSonnet,
	IntentCompliance:  ModelSonnet,
	IntentUnsupported: ModelHaiku, // Quick rejection
}
