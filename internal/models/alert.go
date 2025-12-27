package models

import (
	"fmt"

	"github.com/shopspring/decimal"
)

// AlertType categorizes portfolio alerts
type AlertType string

const (
	AlertConcentration AlertType = "concentration" // >10% in single ticker
	AlertOverlap       AlertType = "overlap"       // Same ticker in 3+ accounts
	AlertHighExpense   AlertType = "high_expense"  // Expense ratio >1%
	AlertUnclassified  AlertType = "unclassified"  // Holdings in "Other"
	AlertCashDrag      AlertType = "cash_drag"     // >10% in cash
	AlertSectorTilt    AlertType = "sector_tilt"   // >30% in single sector
)

// Severity levels for alerts
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityCritical Severity = "critical"
)

// Alert represents a portfolio risk or issue notification
type Alert struct {
	Type       AlertType `json:"type"`
	Severity   Severity  `json:"severity"`
	Title      string    `json:"title"`
	Message    string    `json:"message"`
	Holdings   []string  `json:"holdings,omitempty"` // Affected tickers
	Suggestion string    `json:"suggestion"`
}

// AlertThresholds defines the thresholds for triggering alerts
type AlertThresholds struct {
	ConcentrationPercent decimal.Decimal // Single position max %
	SectorTiltPercent    decimal.Decimal // Single sector max %
	CashDragPercent      decimal.Decimal // Cash max %
	OverlapAccountCount  int             // Same ticker in N+ accounts
}

// DefaultThresholds returns the default alert thresholds
func DefaultThresholds() *AlertThresholds {
	return &AlertThresholds{
		ConcentrationPercent: decimal.NewFromInt(10),
		SectorTiltPercent:    decimal.NewFromInt(30),
		CashDragPercent:      decimal.NewFromInt(10),
		OverlapAccountCount:  3,
	}
}

// AlertDetector analyzes a portfolio and generates alerts
type AlertDetector struct {
	Thresholds *AlertThresholds
}

// NewAlertDetector creates a detector with default thresholds
func NewAlertDetector() *AlertDetector {
	return &AlertDetector{
		Thresholds: DefaultThresholds(),
	}
}

// DetectAlerts analyzes a portfolio and returns all applicable alerts
func (d *AlertDetector) DetectAlerts(p *Portfolio, allocation *AllocationSummary) []Alert {
	var alerts []Alert

	alerts = append(alerts, d.detectConcentration(p, allocation)...)
	alerts = append(alerts, d.detectOverlap(p)...)
	alerts = append(alerts, d.detectCashDrag(allocation)...)
	alerts = append(alerts, d.detectSectorTilt(allocation)...)
	alerts = append(alerts, d.detectUnclassified(p)...)

	return alerts
}

// detectConcentration finds holdings with >threshold% of portfolio
func (d *AlertDetector) detectConcentration(p *Portfolio, allocation *AllocationSummary) []Alert {
	var alerts []Alert

	if p.TotalValue.IsZero() {
		return alerts
	}

	hundred := decimal.NewFromInt(100)

	for ticker, value := range allocation.TickerTotals {
		pct := value.Div(p.TotalValue).Mul(hundred)
		if pct.GreaterThan(d.Thresholds.ConcentrationPercent) {
			alerts = append(alerts, Alert{
				Type:     AlertConcentration,
				Severity: SeverityWarning,
				Title:    "Position Concentration",
				Message: fmt.Sprintf("%s represents %.1f%% of your portfolio, exceeding the %s%% threshold",
					ticker, pct.InexactFloat64(), d.Thresholds.ConcentrationPercent.String()),
				Holdings:   []string{ticker},
				Suggestion: "Consider reducing this position to lower single-stock risk",
			})
		}
	}

	return alerts
}

// detectOverlap finds tickers held in multiple accounts
func (d *AlertDetector) detectOverlap(p *Portfolio) []Alert {
	var alerts []Alert

	// Count accounts per ticker
	tickerAccounts := make(map[string]map[string]bool)
	for _, h := range p.Holdings {
		if tickerAccounts[h.Ticker] == nil {
			tickerAccounts[h.Ticker] = make(map[string]bool)
		}
		tickerAccounts[h.Ticker][h.AccountName] = true
	}

	for ticker, accounts := range tickerAccounts {
		if len(accounts) >= d.Thresholds.OverlapAccountCount {
			accountList := make([]string, 0, len(accounts))
			for acct := range accounts {
				accountList = append(accountList, acct)
			}
			alerts = append(alerts, Alert{
				Type:     AlertOverlap,
				Severity: SeverityInfo,
				Title:    "Ticker Overlap",
				Message: fmt.Sprintf("%s is held in %d accounts",
					ticker, len(accounts)),
				Holdings:   []string{ticker},
				Suggestion: "Consider consolidating for easier management and potential tax efficiency",
			})
		}
	}

	return alerts
}

// detectCashDrag identifies excessive cash holdings
func (d *AlertDetector) detectCashDrag(allocation *AllocationSummary) []Alert {
	var alerts []Alert

	cashSlice, exists := allocation.ByAssetClass[AssetClassCash]
	if !exists {
		return alerts
	}

	if cashSlice.Percentage.GreaterThan(d.Thresholds.CashDragPercent) {
		alerts = append(alerts, Alert{
			Type:     AlertCashDrag,
			Severity: SeverityInfo,
			Title:    "Cash Drag",
			Message: fmt.Sprintf("Cash holdings at %.1f%% may be reducing long-term returns",
				cashSlice.Percentage.InexactFloat64()),
			Suggestion: "Consider deploying excess cash into investments aligned with your goals",
		})
	}

	return alerts
}

// detectSectorTilt finds over-concentration in sectors
func (d *AlertDetector) detectSectorTilt(allocation *AllocationSummary) []Alert {
	var alerts []Alert

	for sector, slice := range allocation.BySector {
		if slice.Percentage.GreaterThan(d.Thresholds.SectorTiltPercent) {
			alerts = append(alerts, Alert{
				Type:     AlertSectorTilt,
				Severity: SeverityWarning,
				Title:    "Sector Concentration",
				Message: fmt.Sprintf("%s sector at %.1f%% exceeds %s%% threshold",
					sector, slice.Percentage.InexactFloat64(), d.Thresholds.SectorTiltPercent.String()),
				Suggestion: "Consider diversifying across sectors to reduce concentration risk",
			})
		}
	}

	return alerts
}

// detectUnclassified finds holdings that still need classification
func (d *AlertDetector) detectUnclassified(p *Portfolio) []Alert {
	var alerts []Alert
	var unclassified []string

	for _, h := range p.Holdings {
		if h.NeedsClassification() {
			unclassified = append(unclassified, h.Ticker)
		}
	}

	if len(unclassified) > 0 {
		alerts = append(alerts, Alert{
			Type:     AlertUnclassified,
			Severity: SeverityCritical,
			Title:    "Unclassified Holdings",
			Message:  fmt.Sprintf("%d holdings need classification for accurate analysis", len(unclassified)),
			Holdings: unclassified,
			Suggestion: "Review and classify these holdings to get accurate allocation insights",
		})
	}

	return alerts
}
