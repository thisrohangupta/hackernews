package importer

import (
	"strings"

	"github.com/findosh/truenorth/internal/models"
)

// Tagger classifies securities by asset class, sector, and geography
type Tagger struct {
	tickerDB map[string]*TickerInfo
}

// TickerInfo holds classification data for a ticker
type TickerInfo struct {
	Ticker     string
	Name       string
	AssetClass models.AssetClass
	Sector     string
	Geography  string
}

// NewTagger creates a new tagger with built-in classifications
func NewTagger() *Tagger {
	t := &Tagger{
		tickerDB: make(map[string]*TickerInfo),
	}
	t.loadBuiltinData()
	return t
}

// TagHoldings classifies a slice of holdings
func (t *Tagger) TagHoldings(holdings []models.Holding) {
	for i := range holdings {
		t.TagHolding(&holdings[i])
	}
}

// TagHolding classifies a single holding
func (t *Tagger) TagHolding(h *models.Holding) {
	ticker := strings.ToUpper(h.Ticker)

	// Check built-in database
	if info, ok := t.tickerDB[ticker]; ok {
		h.AssetClass = info.AssetClass
		h.Sector = info.Sector
		h.Geography = info.Geography
		return
	}

	// Apply heuristics
	t.applyHeuristics(h)
}

func (t *Tagger) applyHeuristics(h *models.Holding) {
	ticker := strings.ToUpper(h.Ticker)
	name := strings.ToLower(h.Name)

	// Cash and money market detection
	if t.isCashLike(ticker, name) {
		h.AssetClass = models.AssetClassCash
		h.Sector = "Cash"
		h.Geography = "US"
		return
	}

	// ETF/Mutual fund detection by name patterns
	if t.isBondFund(ticker, name) {
		h.AssetClass = models.AssetClassFixedIncome
		h.Sector = "Bonds"
		h.Geography = t.detectGeography(name)
		return
	}

	// Crypto detection
	if t.isCrypto(ticker, name) {
		h.AssetClass = models.AssetClassCrypto
		h.Sector = "Cryptocurrency"
		h.Geography = "Global"
		return
	}

	// Alternative investments
	if t.isAlternative(ticker, name) {
		h.AssetClass = models.AssetClassAlternative
		h.Sector = "Alternatives"
		h.Geography = "US"
		return
	}

	// Default to equity for stocks
	if len(ticker) <= 5 && !strings.Contains(ticker, " ") {
		h.AssetClass = models.AssetClassEquity
		h.Sector = t.detectSector(ticker, name)
		h.Geography = t.detectGeography(name)
		return
	}

	// Keep as Other if can't classify
	h.AssetClass = models.AssetClassOther
}

func (t *Tagger) isCashLike(ticker, name string) bool {
	cashIndicators := []string{
		"money market", "cash", "sweep", "settlement",
		"fdic", "treasury bill", "t-bill", "spaxx", "fdrxx",
	}
	for _, ind := range cashIndicators {
		if strings.Contains(name, ind) || strings.Contains(strings.ToLower(ticker), ind) {
			return true
		}
	}
	return false
}

func (t *Tagger) isBondFund(ticker, name string) bool {
	bondIndicators := []string{
		"bond", "fixed income", "treasury", "municipal",
		"corporate bond", "aggregate bond", "income fund",
		"bnd", "agg", "tlt", "ief", "shy", "tip",
	}
	for _, ind := range bondIndicators {
		if strings.Contains(name, ind) || strings.Contains(strings.ToLower(ticker), ind) {
			return true
		}
	}
	return false
}

func (t *Tagger) isCrypto(ticker, name string) bool {
	cryptoIndicators := []string{
		"bitcoin", "ethereum", "crypto", "btc", "eth",
		"gbtc", "ethe", "bito", "coin",
	}
	for _, ind := range cryptoIndicators {
		if strings.Contains(name, ind) || ticker == strings.ToUpper(ind) {
			return true
		}
	}
	return false
}

func (t *Tagger) isAlternative(ticker, name string) bool {
	altIndicators := []string{
		"real estate", "reit", "private equity", "venture",
		"commodity", "gold", "silver", "oil", "infrastructure",
	}
	for _, ind := range altIndicators {
		if strings.Contains(name, ind) {
			return true
		}
	}
	return false
}

func (t *Tagger) detectSector(ticker, name string) string {
	sectorKeywords := map[string][]string{
		"Technology":            {"tech", "software", "semiconductor", "computer", "apple", "microsoft", "google", "nvidia"},
		"Healthcare":            {"health", "pharma", "biotech", "medical", "drug"},
		"Financial Services":    {"bank", "financial", "insurance", "capital", "invest"},
		"Consumer Cyclical":     {"retail", "amazon", "tesla", "consumer disc"},
		"Consumer Defensive":    {"consumer staple", "food", "beverage", "procter", "coca-cola"},
		"Industrials":           {"industrial", "aerospace", "defense", "manufacturing"},
		"Energy":                {"energy", "oil", "gas", "petroleum", "exxon"},
		"Utilities":             {"utility", "utilities", "electric", "water"},
		"Real Estate":           {"real estate", "reit", "property"},
		"Basic Materials":       {"materials", "mining", "chemical"},
		"Communication Services": {"communication", "media", "telecom", "netflix", "disney"},
	}

	nameLower := strings.ToLower(name)
	for sector, keywords := range sectorKeywords {
		for _, kw := range keywords {
			if strings.Contains(nameLower, kw) {
				return sector
			}
		}
	}

	return "Diversified"
}

func (t *Tagger) detectGeography(name string) string {
	nameLower := strings.ToLower(name)

	if strings.Contains(nameLower, "international") || strings.Contains(nameLower, "intl") {
		return "International Developed"
	}
	if strings.Contains(nameLower, "emerging") || strings.Contains(nameLower, "em ") {
		return "Emerging Markets"
	}
	if strings.Contains(nameLower, "global") || strings.Contains(nameLower, "world") {
		return "Global"
	}

	// Default to US
	return "US"
}

// loadBuiltinData populates the ticker database with known classifications
func (t *Tagger) loadBuiltinData() {
	// Major US stocks
	stocks := []TickerInfo{
		{"AAPL", "Apple Inc.", models.AssetClassEquity, "Technology", "US"},
		{"MSFT", "Microsoft Corporation", models.AssetClassEquity, "Technology", "US"},
		{"GOOGL", "Alphabet Inc.", models.AssetClassEquity, "Technology", "US"},
		{"GOOG", "Alphabet Inc.", models.AssetClassEquity, "Technology", "US"},
		{"AMZN", "Amazon.com Inc.", models.AssetClassEquity, "Consumer Cyclical", "US"},
		{"NVDA", "NVIDIA Corporation", models.AssetClassEquity, "Technology", "US"},
		{"META", "Meta Platforms Inc.", models.AssetClassEquity, "Technology", "US"},
		{"TSLA", "Tesla Inc.", models.AssetClassEquity, "Consumer Cyclical", "US"},
		{"BRK.B", "Berkshire Hathaway Inc.", models.AssetClassEquity, "Financial Services", "US"},
		{"JPM", "JPMorgan Chase & Co.", models.AssetClassEquity, "Financial Services", "US"},
		{"V", "Visa Inc.", models.AssetClassEquity, "Financial Services", "US"},
		{"JNJ", "Johnson & Johnson", models.AssetClassEquity, "Healthcare", "US"},
		{"UNH", "UnitedHealth Group", models.AssetClassEquity, "Healthcare", "US"},
		{"XOM", "Exxon Mobil Corporation", models.AssetClassEquity, "Energy", "US"},
		{"PG", "Procter & Gamble Co.", models.AssetClassEquity, "Consumer Defensive", "US"},
		{"MA", "Mastercard Inc.", models.AssetClassEquity, "Financial Services", "US"},
		{"HD", "The Home Depot Inc.", models.AssetClassEquity, "Consumer Cyclical", "US"},
		{"CVX", "Chevron Corporation", models.AssetClassEquity, "Energy", "US"},
		{"MRK", "Merck & Co.", models.AssetClassEquity, "Healthcare", "US"},
		{"ABBV", "AbbVie Inc.", models.AssetClassEquity, "Healthcare", "US"},
	}

	// Major ETFs
	etfs := []TickerInfo{
		{"SPY", "SPDR S&P 500 ETF", models.AssetClassEquity, "Diversified", "US"},
		{"VOO", "Vanguard S&P 500 ETF", models.AssetClassEquity, "Diversified", "US"},
		{"VTI", "Vanguard Total Stock Market ETF", models.AssetClassEquity, "Diversified", "US"},
		{"QQQ", "Invesco QQQ Trust", models.AssetClassEquity, "Technology", "US"},
		{"IVV", "iShares Core S&P 500 ETF", models.AssetClassEquity, "Diversified", "US"},
		{"VEA", "Vanguard FTSE Developed Markets ETF", models.AssetClassEquity, "Diversified", "International Developed"},
		{"VWO", "Vanguard FTSE Emerging Markets ETF", models.AssetClassEquity, "Diversified", "Emerging Markets"},
		{"VXUS", "Vanguard Total International Stock ETF", models.AssetClassEquity, "Diversified", "International Developed"},
		{"BND", "Vanguard Total Bond Market ETF", models.AssetClassFixedIncome, "Bonds", "US"},
		{"AGG", "iShares Core U.S. Aggregate Bond ETF", models.AssetClassFixedIncome, "Bonds", "US"},
		{"TLT", "iShares 20+ Year Treasury Bond ETF", models.AssetClassFixedIncome, "Bonds", "US"},
		{"IEF", "iShares 7-10 Year Treasury Bond ETF", models.AssetClassFixedIncome, "Bonds", "US"},
		{"SHY", "iShares 1-3 Year Treasury Bond ETF", models.AssetClassFixedIncome, "Bonds", "US"},
		{"TIP", "iShares TIPS Bond ETF", models.AssetClassFixedIncome, "Bonds", "US"},
		{"VNQ", "Vanguard Real Estate ETF", models.AssetClassAlternative, "Real Estate", "US"},
		{"GLD", "SPDR Gold Trust", models.AssetClassAlternative, "Commodities", "Global"},
		{"GBTC", "Grayscale Bitcoin Trust", models.AssetClassCrypto, "Cryptocurrency", "Global"},
	}

	// Cash instruments
	cash := []TickerInfo{
		{"SPAXX", "Fidelity Government Money Market", models.AssetClassCash, "Cash", "US"},
		{"FDRXX", "Fidelity Government Cash Reserves", models.AssetClassCash, "Cash", "US"},
		{"VMFXX", "Vanguard Federal Money Market", models.AssetClassCash, "Cash", "US"},
		{"SWVXX", "Schwab Value Advantage Money Fund", models.AssetClassCash, "Cash", "US"},
	}

	// Populate database
	for _, info := range stocks {
		t.tickerDB[info.Ticker] = &TickerInfo{
			Ticker:     info.Ticker,
			Name:       info.Name,
			AssetClass: info.AssetClass,
			Sector:     info.Sector,
			Geography:  info.Geography,
		}
	}
	for _, info := range etfs {
		t.tickerDB[info.Ticker] = &TickerInfo{
			Ticker:     info.Ticker,
			Name:       info.Name,
			AssetClass: info.AssetClass,
			Sector:     info.Sector,
			Geography:  info.Geography,
		}
	}
	for _, info := range cash {
		t.tickerDB[info.Ticker] = &TickerInfo{
			Ticker:     info.Ticker,
			Name:       info.Name,
			AssetClass: info.AssetClass,
			Sector:     info.Sector,
			Geography:  info.Geography,
		}
	}
}
