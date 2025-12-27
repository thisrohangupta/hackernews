package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// AssetClass categorizes holdings by type
type AssetClass string

const (
	AssetClassEquity      AssetClass = "equity"
	AssetClassFixedIncome AssetClass = "fixed_income"
	AssetClassAlternative AssetClass = "alternative" // PE, VC, Real Estate
	AssetClassCrypto      AssetClass = "crypto"
	AssetClassCash        AssetClass = "cash"
	AssetClassOther       AssetClass = "other" // Should be zero in final view
)

// AllAssetClasses returns all valid asset classes for iteration
func AllAssetClasses() []AssetClass {
	return []AssetClass{
		AssetClassEquity,
		AssetClassFixedIncome,
		AssetClassAlternative,
		AssetClassCrypto,
		AssetClassCash,
		AssetClassOther,
	}
}

// DisplayName returns human-readable name for the asset class
func (a AssetClass) DisplayName() string {
	switch a {
	case AssetClassEquity:
		return "Equities"
	case AssetClassFixedIncome:
		return "Fixed Income"
	case AssetClassAlternative:
		return "Alternatives"
	case AssetClassCrypto:
		return "Cryptocurrency"
	case AssetClassCash:
		return "Cash"
	case AssetClassOther:
		return "Other"
	default:
		return string(a)
	}
}

// Holding represents a single position in an account
type Holding struct {
	ID           uuid.UUID       `json:"id"`
	PortfolioID  uuid.UUID       `json:"portfolio_id"`
	AccountName  string          `json:"account_name"` // e.g., "Schwab IRA"
	Ticker       string          `json:"ticker"`       // e.g., "AAPL"
	Name         string          `json:"name"`         // e.g., "Apple Inc."
	Quantity     decimal.Decimal `json:"quantity"`
	CostBasis    decimal.Decimal `json:"cost_basis"`
	CurrentPrice decimal.Decimal `json:"current_price"`
	MarketValue  decimal.Decimal `json:"market_value"`

	// Classification (AI-tagged or manual)
	AssetClass AssetClass `json:"asset_class"`
	Sector     string     `json:"sector"`    // Technology, Healthcare, etc.
	Geography  string     `json:"geography"` // US, International, Emerging

	// Metadata
	IsManualEntry bool      `json:"is_manual_entry"`
	Source        string    `json:"source"` // "schwab_csv", "fidelity_csv", "manual"
	ImportedAt    time.Time `json:"imported_at"`
}

// NewHolding creates a new holding with generated ID
func NewHolding(portfolioID uuid.UUID, ticker, name, accountName string) *Holding {
	return &Holding{
		ID:           uuid.New(),
		PortfolioID:  portfolioID,
		AccountName:  accountName,
		Ticker:       ticker,
		Name:         name,
		Quantity:     decimal.Zero,
		CostBasis:    decimal.Zero,
		CurrentPrice: decimal.Zero,
		MarketValue:  decimal.Zero,
		AssetClass:   AssetClassOther,
		ImportedAt:   time.Now().UTC(),
	}
}

// CalculateMarketValue updates market value based on quantity and current price
func (h *Holding) CalculateMarketValue() {
	h.MarketValue = h.Quantity.Mul(h.CurrentPrice)
}

// GainLoss returns the unrealized gain/loss
func (h *Holding) GainLoss() decimal.Decimal {
	return h.MarketValue.Sub(h.CostBasis)
}

// GainLossPercent returns the unrealized gain/loss as a percentage
func (h *Holding) GainLossPercent() decimal.Decimal {
	if h.CostBasis.IsZero() {
		return decimal.Zero
	}
	return h.GainLoss().Div(h.CostBasis).Mul(decimal.NewFromInt(100)).Round(2)
}

// IsCash returns true if this holding represents cash or money market
func (h *Holding) IsCash() bool {
	return h.AssetClass == AssetClassCash
}

// NeedsClassification returns true if the holding is still unclassified
func (h *Holding) NeedsClassification() bool {
	return h.AssetClass == AssetClassOther
}

// Standard sectors for classification
var StandardSectors = []string{
	"Technology",
	"Healthcare",
	"Financial Services",
	"Consumer Cyclical",
	"Consumer Defensive",
	"Industrials",
	"Energy",
	"Utilities",
	"Real Estate",
	"Basic Materials",
	"Communication Services",
	"Diversified",
}

// Standard geographies for classification
var StandardGeographies = []string{
	"US",
	"International Developed",
	"Emerging Markets",
	"Global",
}
