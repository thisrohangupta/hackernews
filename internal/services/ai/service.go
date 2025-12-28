package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/findosh/truenorth/internal/models"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Service provides AI-powered portfolio analysis
type Service struct {
	cfg        *Config
	httpClient *http.Client
	cache      *QueryCache
	router     *ModelRouter
	auditor    *AuditLogger
	mu         sync.RWMutex

	// Usage tracking
	dailyTokens map[string]int // userID -> token count
	lastReset   time.Time
}

// NewService creates a new AI service
func NewService(cfg *Config) *Service {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	return &Service{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		cache:       NewQueryCache(cfg.CacheTTL, cfg.SemanticCacheThresh),
		router:      NewModelRouter(cfg),
		auditor:     NewAuditLogger(cfg.EnableAuditLog),
		dailyTokens: make(map[string]int),
		lastReset:   time.Now(),
	}
}

// Query represents a user query with context
type Query struct {
	ID          string            `json:"id"`
	UserID      string            `json:"user_id"`
	Text        string            `json:"text"`
	PortfolioID string            `json:"portfolio_id,omitempty"`
	Context     map[string]string `json:"context,omitempty"`
	Timestamp   time.Time         `json:"timestamp"`
}

// Response represents an AI response
type Response struct {
	ID           string       `json:"id"`
	QueryID      string       `json:"query_id"`
	Text         string       `json:"text"`
	Sources      []Source     `json:"sources,omitempty"`
	Disclaimers  []string     `json:"disclaimers,omitempty"`
	Model        Model        `json:"model"`
	Intent       QueryIntent  `json:"intent"`
	TokensUsed   TokenUsage   `json:"tokens_used"`
	Cached       bool         `json:"cached"`
	ProcessingMs int64        `json:"processing_ms"`
	Timestamp    time.Time    `json:"timestamp"`
}

// Source represents a data source citation
type Source struct {
	Type        string `json:"type"` // "holding", "document", "market_data"
	Reference   string `json:"reference"`
	Description string `json:"description"`
	URL         string `json:"url,omitempty"`
}

// TokenUsage tracks token consumption
type TokenUsage struct {
	Input  int `json:"input"`
	Output int `json:"output"`
	Total  int `json:"total"`
}

// Ask processes a user query and returns an AI response
func (s *Service) Ask(ctx context.Context, query *Query, portfolio *models.Portfolio) (*Response, error) {
	startTime := time.Now()

	// Generate IDs
	if query.ID == "" {
		query.ID = uuid.New().String()
	}
	query.Timestamp = time.Now()

	// Check rate limits
	if err := s.checkRateLimits(query.UserID); err != nil {
		return nil, err
	}

	// Classify intent
	intent := s.router.ClassifyIntent(query.Text)

	// Check for blocked content
	if intent == IntentUnsupported {
		return s.blockedResponse(query, intent, startTime), nil
	}

	// Check cache
	if s.cfg.CacheEnabled {
		if cached := s.cache.Get(query.Text, query.PortfolioID); cached != nil {
			cached.Cached = true
			cached.QueryID = query.ID
			cached.ProcessingMs = time.Since(startTime).Milliseconds()
			return cached, nil
		}
	}

	// Route to appropriate model
	model := s.router.SelectModel(intent, query.Text)

	// Build prompt with portfolio context
	systemPrompt := s.buildSystemPrompt(portfolio)
	userPrompt := s.buildUserPrompt(query, portfolio)

	// Call Claude API
	response, usage, err := s.callClaude(ctx, model, systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("claude API error: %w", err)
	}

	// Track usage
	s.trackUsage(query.UserID, usage)

	// Build response
	resp := &Response{
		ID:           uuid.New().String(),
		QueryID:      query.ID,
		Text:         response,
		Model:        model,
		Intent:       intent,
		TokensUsed:   usage,
		Cached:       false,
		ProcessingMs: time.Since(startTime).Milliseconds(),
		Timestamp:    time.Now(),
	}

	// Add disclaimers
	if s.cfg.EnableDisclaimers {
		resp.Disclaimers = s.getDisclaimers(intent)
	}

	// Extract sources from response
	resp.Sources = s.extractSources(response, portfolio)

	// Cache response
	if s.cfg.CacheEnabled {
		s.cache.Set(query.Text, query.PortfolioID, resp)
	}

	// Audit log
	s.auditor.Log(query, resp)

	return resp, nil
}

// buildSystemPrompt creates the system prompt with compliance guardrails
func (s *Service) buildSystemPrompt(portfolio *models.Portfolio) string {
	var sb strings.Builder

	sb.WriteString(`You are TrueNorth AI, a sophisticated portfolio analysis assistant for high-net-worth DIY investors.

## Your Role
- Provide factual, data-driven portfolio analysis
- Help users understand their investments, risks, and opportunities
- Explain financial concepts clearly
- Cite specific holdings and data when answering

## Critical Rules
1. NEVER provide specific buy/sell recommendations for individual securities
2. NEVER guarantee returns or predict specific price movements
3. NEVER provide tax advice - only educational information about tax concepts
4. ALWAYS include relevant disclaimers
5. ALWAYS cite sources for numerical claims
6. If you don't have data to answer accurately, say so clearly

## Response Format
- Be concise but thorough
- Use bullet points for clarity
- Include specific numbers from the portfolio when relevant
- End with actionable next steps when appropriate

`)

	// Add portfolio context if available
	if portfolio != nil && len(portfolio.Holdings) > 0 {
		sb.WriteString("\n## Current Portfolio Summary\n")
		sb.WriteString(fmt.Sprintf("- Total Value: $%s\n", portfolio.TotalValue.StringFixed(2)))
		sb.WriteString(fmt.Sprintf("- Number of Holdings: %d\n", len(portfolio.Holdings)))

		// Top holdings
		sb.WriteString("\n### Top Holdings:\n")
		count := 0
		for _, h := range portfolio.Holdings {
			if count >= 10 {
				break
			}
			pct := "0.00"
			if !portfolio.TotalValue.IsZero() {
				pct = h.MarketValue.Div(portfolio.TotalValue).Mul(decimal.NewFromInt(100)).StringFixed(2)
			}
			sb.WriteString(fmt.Sprintf("- %s (%s): $%s (%s%%)\n",
				h.Ticker, h.AssetClass.DisplayName(),
				h.MarketValue.StringFixed(2), pct))
			count++
		}
	}

	return sb.String()
}

// buildUserPrompt formats the user query with context
func (s *Service) buildUserPrompt(query *Query, portfolio *models.Portfolio) string {
	var sb strings.Builder

	sb.WriteString(query.Text)

	// Add any additional context
	if len(query.Context) > 0 {
		sb.WriteString("\n\nAdditional context:\n")
		for k, v := range query.Context {
			sb.WriteString(fmt.Sprintf("- %s: %s\n", k, v))
		}
	}

	return sb.String()
}

// callClaude makes a request to the Claude API
func (s *Service) callClaude(ctx context.Context, model Model, systemPrompt, userPrompt string) (string, TokenUsage, error) {
	// Build request body
	reqBody := map[string]interface{}{
		"model":      string(model),
		"max_tokens": s.cfg.MaxTokens,
		"messages": []map[string]string{
			{"role": "user", "content": userPrompt},
		},
		"system":      systemPrompt,
		"temperature": s.cfg.Temperature,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", TokenUsage{}, err
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", s.cfg.BaseURL+"/messages", bytes.NewReader(jsonBody))
	if err != nil {
		return "", TokenUsage{}, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", s.cfg.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	// Make request with retries
	var resp *http.Response
	for i := 0; i <= s.cfg.MaxRetries; i++ {
		resp, err = s.httpClient.Do(req)
		if err == nil && resp.StatusCode < 500 {
			break
		}
		if i < s.cfg.MaxRetries {
			time.Sleep(time.Duration(i+1) * time.Second)
		}
	}

	if err != nil {
		return "", TokenUsage{}, err
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", TokenUsage{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return "", TokenUsage{}, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", TokenUsage{}, err
	}

	if len(result.Content) == 0 {
		return "", TokenUsage{}, fmt.Errorf("empty response from API")
	}

	usage := TokenUsage{
		Input:  result.Usage.InputTokens,
		Output: result.Usage.OutputTokens,
		Total:  result.Usage.InputTokens + result.Usage.OutputTokens,
	}

	return result.Content[0].Text, usage, nil
}

// checkRateLimits verifies user hasn't exceeded token budget
func (s *Service) checkRateLimits(userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Reset daily counters if needed
	if time.Since(s.lastReset) > 24*time.Hour {
		s.dailyTokens = make(map[string]int)
		s.lastReset = time.Now()
	}

	if s.dailyTokens[userID] >= s.cfg.DailyTokenBudget {
		return fmt.Errorf("daily token limit exceeded")
	}

	return nil
}

// trackUsage records token usage
func (s *Service) trackUsage(userID string, usage TokenUsage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dailyTokens[userID] += usage.Total
}

// blockedResponse returns a response for blocked queries
func (s *Service) blockedResponse(query *Query, intent QueryIntent, startTime time.Time) *Response {
	return &Response{
		ID:      uuid.New().String(),
		QueryID: query.ID,
		Text: `I can't provide specific investment recommendations, guaranteed return predictions, or personalized tax advice.

However, I can help you with:
- Understanding your current portfolio allocation and risk exposure
- Explaining financial concepts and investment strategies
- Analyzing historical performance and metrics
- Comparing different asset classes and their characteristics

Please rephrase your question, and I'll do my best to provide educational information.`,
		Model:        s.cfg.SimpleModel,
		Intent:       intent,
		TokensUsed:   TokenUsage{},
		Cached:       false,
		ProcessingMs: time.Since(startTime).Milliseconds(),
		Timestamp:    time.Now(),
		Disclaimers: []string{
			"This response was generated because your query appeared to request specific investment advice, which we cannot provide.",
		},
	}
}

// getDisclaimers returns appropriate disclaimers for the query type
func (s *Service) getDisclaimers(intent QueryIntent) []string {
	base := []string{
		"This information is AI-generated and for educational purposes only.",
		"This does not constitute investment, tax, or legal advice.",
		"Past performance does not guarantee future results.",
	}

	switch intent {
	case IntentTax:
		return append(base, "Consult a qualified tax professional for personalized tax advice.")
	case IntentRisk:
		return append(base, "Risk assessments are based on historical data and may not reflect future conditions.")
	case IntentProjection:
		return append(base, "Projections are hypothetical and based on assumptions that may not materialize.")
	default:
		return base
	}
}

// extractSources identifies data sources referenced in the response
func (s *Service) extractSources(response string, portfolio *models.Portfolio) []Source {
	sources := make([]Source, 0)

	if portfolio == nil {
		return sources
	}

	// Check if holdings are mentioned
	for _, h := range portfolio.Holdings {
		if strings.Contains(response, h.Ticker) {
			sources = append(sources, Source{
				Type:        "holding",
				Reference:   h.Ticker,
				Description: h.Name,
			})
		}
	}

	// Deduplicate
	seen := make(map[string]bool)
	unique := make([]Source, 0)
	for _, src := range sources {
		if !seen[src.Reference] {
			seen[src.Reference] = true
			unique = append(unique, src)
		}
	}

	return unique
}

// GetUsageStats returns usage statistics for a user
func (s *Service) GetUsageStats(userID string) map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tokensUsed := s.dailyTokens[userID]
	remaining := s.cfg.DailyTokenBudget - tokensUsed

	return map[string]interface{}{
		"tokens_used_today": tokensUsed,
		"tokens_remaining":  remaining,
		"daily_limit":       s.cfg.DailyTokenBudget,
		"reset_time":        s.lastReset.Add(24 * time.Hour),
	}
}

// GetAuditEntries returns audit entries for a user
func (s *Service) GetAuditEntries(userID string, since time.Time, limit int) []AuditEntry {
	return s.auditor.GetEntries(userID, since, limit)
}

// GetCacheStats returns cache statistics
func (s *Service) GetCacheStats() map[string]interface{} {
	return s.cache.Stats()
}

// InvalidateCache invalidates cache for a portfolio
func (s *Service) InvalidateCache(portfolioID string) {
	s.cache.Invalidate(portfolioID)
}
