package marketdata

import (
	"testing"
	"time"

	"github.com/findosh/truenorth/internal/models"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestNewService(t *testing.T) {
	svc := NewService(Config{Provider: ProviderMock})
	if svc == nil {
		t.Fatal("Expected service to be created")
	}
}

func TestService_GetQuote(t *testing.T) {
	svc := NewService(Config{Provider: ProviderMock})

	quote, err := svc.GetQuote("AAPL")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if quote == nil {
		t.Fatal("Expected quote to be returned")
	}

	if quote.Ticker != "AAPL" {
		t.Errorf("Expected ticker AAPL, got %s", quote.Ticker)
	}

	if quote.Price.IsZero() {
		t.Error("Expected non-zero price")
	}
}

func TestService_GetQuote_Cached(t *testing.T) {
	svc := NewService(Config{
		Provider: ProviderMock,
		CacheTTL: 1 * time.Hour,
	})

	// First call
	quote1, err := svc.GetQuote("AAPL")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Second call should return cached
	quote2, err := svc.GetQuote("AAPL")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !quote1.Price.Equal(quote2.Price) {
		t.Error("Expected cached quote to return same price")
	}
}

func TestService_GetQuotes(t *testing.T) {
	svc := NewService(Config{Provider: ProviderMock})

	tickers := []string{"AAPL", "MSFT", "GOOGL"}
	quotes, err := svc.GetQuotes(tickers)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(quotes) != len(tickers) {
		t.Errorf("Expected %d quotes, got %d", len(tickers), len(quotes))
	}

	for _, ticker := range tickers {
		if _, ok := quotes[ticker]; !ok {
			t.Errorf("Missing quote for %s", ticker)
		}
	}
}

func TestService_IsMarketOpen(t *testing.T) {
	svc := NewService(Config{Provider: ProviderMock})

	// Just verify it doesn't panic
	_ = svc.IsMarketOpen()
}

func TestService_GetMarketStatus(t *testing.T) {
	svc := NewService(Config{Provider: ProviderMock})

	status := svc.GetMarketStatus()
	if status == nil {
		t.Fatal("Expected market status")
	}

	if status.Message == "" {
		t.Error("Expected status message")
	}
}

func TestService_UpdatePortfolioValues(t *testing.T) {
	svc := NewService(Config{Provider: ProviderMock})

	portfolio := &models.Portfolio{
		ID:     uuid.New(),
		UserID: uuid.New(),
		Holdings: []models.Holding{
			{
				ID:       uuid.New(),
				Ticker:   "AAPL",
				Quantity: decimal.NewFromInt(10),
			},
			{
				ID:       uuid.New(),
				Ticker:   "MSFT",
				Quantity: decimal.NewFromInt(5),
			},
		},
	}

	err := svc.UpdatePortfolioValues(portfolio)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Check holdings have updated prices
	for _, h := range portfolio.Holdings {
		if h.CurrentPrice.IsZero() {
			t.Errorf("Holding %s should have updated price", h.Ticker)
		}
		if h.MarketValue.IsZero() {
			t.Errorf("Holding %s should have market value", h.Ticker)
		}
	}

	// Check portfolio total value
	if portfolio.TotalValue.IsZero() {
		t.Error("Portfolio total value should be updated")
	}
}

func TestService_UpdatePortfolioValues_Nil(t *testing.T) {
	svc := NewService(Config{Provider: ProviderMock})

	// Should not error on nil
	err := svc.UpdatePortfolioValues(nil)
	if err != nil {
		t.Errorf("Expected no error for nil portfolio, got: %v", err)
	}

	// Should not error on empty holdings
	err = svc.UpdatePortfolioValues(&models.Portfolio{})
	if err != nil {
		t.Errorf("Expected no error for empty portfolio, got: %v", err)
	}
}

func TestService_GetHistoricalPrices(t *testing.T) {
	svc := NewService(Config{Provider: ProviderMock})

	prices, err := svc.GetHistoricalPrices("AAPL", models.Period1Month)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(prices) == 0 {
		t.Error("Expected historical prices")
	}

	for _, p := range prices {
		if p.Ticker != "AAPL" {
			t.Errorf("Expected ticker AAPL, got %s", p.Ticker)
		}
		if p.Close.IsZero() {
			t.Error("Expected non-zero close price")
		}
	}
}

func TestMockBasePrice(t *testing.T) {
	svc := NewService(Config{Provider: ProviderMock})

	// Known tickers should have realistic prices
	knownTickers := map[string]float64{
		"AAPL": 175.00,
		"MSFT": 375.00,
		"VOO":  430.00,
	}

	for ticker, expectedPrice := range knownTickers {
		price := svc.mockBasePrice(ticker)
		if !price.Equal(decimal.NewFromFloat(expectedPrice)) {
			t.Errorf("Expected %s price to be %.2f, got %s", ticker, expectedPrice, price.String())
		}
	}

	// Unknown tickers should still return a valid price
	unknownPrice := svc.mockBasePrice("UNKNOWN")
	if unknownPrice.IsZero() || unknownPrice.IsNegative() {
		t.Error("Unknown ticker should have positive price")
	}
}
