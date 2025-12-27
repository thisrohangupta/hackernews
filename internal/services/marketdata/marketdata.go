package marketdata

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/findosh/truenorth/internal/models"
	"github.com/shopspring/decimal"
)

// Provider represents a market data provider
type Provider string

const (
	ProviderMock   Provider = "mock"
	ProviderYahoo  Provider = "yahoo"
	ProviderAlpha  Provider = "alphavantage"
)

// Quote represents a stock/ETF quote
type Quote struct {
	Ticker        string          `json:"ticker"`
	Price         decimal.Decimal `json:"price"`
	Change        decimal.Decimal `json:"change"`
	ChangePercent decimal.Decimal `json:"change_percent"`
	Open          decimal.Decimal `json:"open"`
	High          decimal.Decimal `json:"high"`
	Low           decimal.Decimal `json:"low"`
	Volume        int64           `json:"volume"`
	MarketCap     decimal.Decimal `json:"market_cap,omitempty"`
	PE            decimal.Decimal `json:"pe,omitempty"`
	Dividend      decimal.Decimal `json:"dividend,omitempty"`
	LastUpdated   time.Time       `json:"last_updated"`
	IsMarketOpen  bool            `json:"is_market_open"`
}

// Service provides market data functionality
type Service struct {
	provider   Provider
	apiKey     string
	cache      map[string]*Quote
	cacheTTL   time.Duration
	mu         sync.RWMutex
	httpClient *http.Client
}

// Config holds service configuration
type Config struct {
	Provider Provider
	APIKey   string
	CacheTTL time.Duration
}

// NewService creates a new market data service
func NewService(cfg Config) *Service {
	if cfg.CacheTTL == 0 {
		cfg.CacheTTL = 5 * time.Minute
	}

	return &Service{
		provider: cfg.Provider,
		apiKey:   cfg.APIKey,
		cache:    make(map[string]*Quote),
		cacheTTL: cfg.CacheTTL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetQuote fetches a quote for a single ticker
func (s *Service) GetQuote(ticker string) (*Quote, error) {
	// Check cache first
	s.mu.RLock()
	if cached, ok := s.cache[ticker]; ok {
		if time.Since(cached.LastUpdated) < s.cacheTTL {
			s.mu.RUnlock()
			return cached, nil
		}
	}
	s.mu.RUnlock()

	// Fetch from provider
	var quote *Quote
	var err error

	switch s.provider {
	case ProviderYahoo:
		quote, err = s.fetchYahooQuote(ticker)
	case ProviderAlpha:
		quote, err = s.fetchAlphaVantageQuote(ticker)
	default:
		quote = s.getMockQuote(ticker)
	}

	if err != nil {
		return nil, err
	}

	// Update cache
	s.mu.Lock()
	s.cache[ticker] = quote
	s.mu.Unlock()

	return quote, nil
}

// GetQuotes fetches quotes for multiple tickers
func (s *Service) GetQuotes(tickers []string) (map[string]*Quote, error) {
	quotes := make(map[string]*Quote)
	var wg sync.WaitGroup
	var mu sync.Mutex
	errors := make([]error, 0)

	for _, ticker := range tickers {
		wg.Add(1)
		go func(t string) {
			defer wg.Done()

			quote, err := s.GetQuote(t)
			mu.Lock()
			if err != nil {
				errors = append(errors, fmt.Errorf("%s: %w", t, err))
			} else {
				quotes[t] = quote
			}
			mu.Unlock()
		}(ticker)
	}

	wg.Wait()

	if len(errors) > 0 && len(quotes) == 0 {
		return nil, errors[0]
	}

	return quotes, nil
}

// UpdatePortfolioValues updates market values for portfolio holdings
func (s *Service) UpdatePortfolioValues(portfolio *models.Portfolio) error {
	if portfolio == nil || len(portfolio.Holdings) == 0 {
		return nil
	}

	// Collect tickers
	tickers := make([]string, 0, len(portfolio.Holdings))
	for _, h := range portfolio.Holdings {
		if h.Ticker != "" {
			tickers = append(tickers, h.Ticker)
		}
	}

	// Fetch quotes
	quotes, err := s.GetQuotes(tickers)
	if err != nil {
		return err
	}

	// Update holdings
	totalValue := decimal.Zero
	for i := range portfolio.Holdings {
		h := &portfolio.Holdings[i]
		if quote, ok := quotes[h.Ticker]; ok {
			h.CurrentPrice = quote.Price
			h.MarketValue = h.Quantity.Mul(quote.Price)
		}
		totalValue = totalValue.Add(h.MarketValue)
	}

	portfolio.TotalValue = totalValue
	portfolio.LastUpdated = time.Now()

	return nil
}

// IsMarketOpen checks if the US stock market is currently open
func (s *Service) IsMarketOpen() bool {
	now := time.Now().In(time.FixedZone("EST", -5*3600))

	// Check if weekday
	if now.Weekday() == time.Saturday || now.Weekday() == time.Sunday {
		return false
	}

	// Market hours: 9:30 AM - 4:00 PM EST
	hour := now.Hour()
	minute := now.Minute()

	if hour < 9 || (hour == 9 && minute < 30) {
		return false
	}
	if hour >= 16 {
		return false
	}

	return true
}

// Mock data for development/testing
func (s *Service) getMockQuote(ticker string) *Quote {
	// Generate deterministic mock data based on ticker
	basePrice := s.mockBasePrice(ticker)
	changePercent := s.mockChange(ticker)
	change := basePrice.Mul(changePercent).Div(decimal.NewFromInt(100))

	return &Quote{
		Ticker:        ticker,
		Price:         basePrice,
		Change:        change.Round(2),
		ChangePercent: changePercent.Round(2),
		Open:          basePrice.Sub(change.Div(decimal.NewFromInt(2))).Round(2),
		High:          basePrice.Add(basePrice.Mul(decimal.NewFromFloat(0.01))).Round(2),
		Low:           basePrice.Sub(basePrice.Mul(decimal.NewFromFloat(0.01))).Round(2),
		Volume:        1000000 + int64(len(ticker)*100000),
		LastUpdated:   time.Now(),
		IsMarketOpen:  s.IsMarketOpen(),
	}
}

func (s *Service) mockBasePrice(ticker string) decimal.Decimal {
	// Known approximate prices (for realistic mock data)
	prices := map[string]float64{
		"AAPL":  175.00,
		"MSFT":  375.00,
		"GOOGL": 140.00,
		"AMZN":  180.00,
		"NVDA":  475.00,
		"META":  500.00,
		"TSLA":  250.00,
		"JPM":   195.00,
		"V":     280.00,
		"JNJ":   160.00,
		"VOO":   430.00,
		"VTI":   235.00,
		"SPY":   470.00,
		"QQQ":   400.00,
		"BND":   73.00,
		"AGG":   98.00,
		"VNQ":   85.00,
		"GLD":   185.00,
	}

	if price, ok := prices[ticker]; ok {
		return decimal.NewFromFloat(price)
	}

	// Generate from ticker hash
	hash := 0
	for _, c := range ticker {
		hash += int(c)
	}
	return decimal.NewFromFloat(50.0 + float64(hash%200))
}

func (s *Service) mockChange(ticker string) decimal.Decimal {
	// Generate small random-ish change based on ticker and time
	hash := 0
	for _, c := range ticker {
		hash += int(c)
	}
	hash += time.Now().Day()

	change := float64(hash%300-150) / 100.0 // -1.5% to +1.5%
	return decimal.NewFromFloat(change)
}

// Yahoo Finance integration (simplified)
func (s *Service) fetchYahooQuote(ticker string) (*Quote, error) {
	url := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s", ticker)

	resp, err := s.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch quote: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Fall back to mock data
		return s.getMockQuote(ticker), nil
	}

	var result struct {
		Chart struct {
			Result []struct {
				Meta struct {
					RegularMarketPrice decimal.Decimal `json:"regularMarketPrice"`
					PreviousClose      decimal.Decimal `json:"previousClose"`
				} `json:"meta"`
			} `json:"result"`
		} `json:"chart"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Chart.Result) == 0 {
		return s.getMockQuote(ticker), nil
	}

	meta := result.Chart.Result[0].Meta
	change := meta.RegularMarketPrice.Sub(meta.PreviousClose)
	changePercent := decimal.Zero
	if !meta.PreviousClose.IsZero() {
		changePercent = change.Div(meta.PreviousClose).Mul(decimal.NewFromInt(100))
	}

	return &Quote{
		Ticker:        ticker,
		Price:         meta.RegularMarketPrice,
		Change:        change.Round(2),
		ChangePercent: changePercent.Round(2),
		LastUpdated:   time.Now(),
		IsMarketOpen:  s.IsMarketOpen(),
	}, nil
}

// Alpha Vantage integration (simplified)
func (s *Service) fetchAlphaVantageQuote(ticker string) (*Quote, error) {
	if s.apiKey == "" {
		return s.getMockQuote(ticker), nil
	}

	url := fmt.Sprintf("https://www.alphavantage.co/query?function=GLOBAL_QUOTE&symbol=%s&apikey=%s",
		ticker, s.apiKey)

	resp, err := s.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch quote: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		GlobalQuote struct {
			Price         string `json:"05. price"`
			Change        string `json:"09. change"`
			ChangePercent string `json:"10. change percent"`
			Open          string `json:"02. open"`
			High          string `json:"03. high"`
			Low           string `json:"04. low"`
			Volume        string `json:"06. volume"`
		} `json:"Global Quote"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	price, _ := decimal.NewFromString(result.GlobalQuote.Price)
	change, _ := decimal.NewFromString(result.GlobalQuote.Change)
	open, _ := decimal.NewFromString(result.GlobalQuote.Open)
	high, _ := decimal.NewFromString(result.GlobalQuote.High)
	low, _ := decimal.NewFromString(result.GlobalQuote.Low)

	// Parse change percent (remove % sign)
	changePercentStr := result.GlobalQuote.ChangePercent
	if len(changePercentStr) > 0 && changePercentStr[len(changePercentStr)-1] == '%' {
		changePercentStr = changePercentStr[:len(changePercentStr)-1]
	}
	changePercent, _ := decimal.NewFromString(changePercentStr)

	return &Quote{
		Ticker:        ticker,
		Price:         price,
		Change:        change,
		ChangePercent: changePercent,
		Open:          open,
		High:          high,
		Low:           low,
		LastUpdated:   time.Now(),
		IsMarketOpen:  s.IsMarketOpen(),
	}, nil
}

// GetHistoricalPrices fetches historical price data
func (s *Service) GetHistoricalPrices(ticker string, period string) ([]models.PriceHistory, error) {
	// For MVP, return simulated historical data
	startDate := models.GetPeriodStartDate(period)
	endDate := time.Now().UTC()

	// Get current price
	quote, err := s.GetQuote(ticker)
	if err != nil {
		return nil, err
	}

	// Generate historical prices working backward from current
	prices := make([]models.PriceHistory, 0)
	currentPrice := quote.Price
	current := endDate

	// Daily volatility (simplified)
	dailyVol := decimal.NewFromFloat(0.015) // 1.5% daily volatility

	for current.After(startDate) {
		// Random walk backward
		change := dailyVol.Mul(decimal.NewFromFloat(float64(current.Day()%10-5) / 5))
		prevPrice := currentPrice.Div(decimal.NewFromInt(1).Add(change))

		prices = append([]models.PriceHistory{{
			Ticker:   ticker,
			Date:     current,
			Open:     prevPrice.Mul(decimal.NewFromFloat(0.998)).Round(2),
			High:     currentPrice.Mul(decimal.NewFromFloat(1.005)).Round(2),
			Low:      prevPrice.Mul(decimal.NewFromFloat(0.995)).Round(2),
			Close:    currentPrice.Round(2),
			AdjClose: currentPrice.Round(2),
			Volume:   1000000 + int64(current.Day()*10000),
		}}, prices...)

		currentPrice = prevPrice
		current = current.AddDate(0, 0, -1)

		// Skip weekends
		if current.Weekday() == time.Sunday {
			current = current.AddDate(0, 0, -2)
		} else if current.Weekday() == time.Saturday {
			current = current.AddDate(0, 0, -1)
		}
	}

	return prices, nil
}

// MarketStatus represents overall market status
type MarketStatus struct {
	IsOpen       bool      `json:"is_open"`
	NextOpen     time.Time `json:"next_open,omitempty"`
	NextClose    time.Time `json:"next_close,omitempty"`
	Message      string    `json:"message"`
	LastUpdated  time.Time `json:"last_updated"`
}

// GetMarketStatus returns current market status
func (s *Service) GetMarketStatus() *MarketStatus {
	now := time.Now().In(time.FixedZone("EST", -5*3600))
	isOpen := s.IsMarketOpen()

	status := &MarketStatus{
		IsOpen:      isOpen,
		LastUpdated: now,
	}

	if isOpen {
		// Calculate close time today
		status.NextClose = time.Date(now.Year(), now.Month(), now.Day(), 16, 0, 0, 0, now.Location())
		status.Message = "Market is open"
	} else {
		// Calculate next open
		nextOpen := now
		hour := now.Hour()

		if hour >= 16 || (hour < 9 || (hour == 9 && now.Minute() < 30)) {
			// After close or before open today
			if hour >= 16 {
				nextOpen = nextOpen.AddDate(0, 0, 1)
			}
		}

		// Skip to next weekday
		for nextOpen.Weekday() == time.Saturday || nextOpen.Weekday() == time.Sunday {
			nextOpen = nextOpen.AddDate(0, 0, 1)
		}

		status.NextOpen = time.Date(nextOpen.Year(), nextOpen.Month(), nextOpen.Day(), 9, 30, 0, 0, now.Location())
		status.Message = "Market is closed"
	}

	return status
}
