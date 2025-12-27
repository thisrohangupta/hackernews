package models

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestDefaultThresholds(t *testing.T) {
	thresholds := DefaultThresholds()

	if !thresholds.ConcentrationPercent.Equal(decimal.NewFromInt(10)) {
		t.Errorf("Expected concentration threshold 10, got %s", thresholds.ConcentrationPercent)
	}
	if !thresholds.SectorTiltPercent.Equal(decimal.NewFromInt(30)) {
		t.Errorf("Expected sector tilt threshold 30, got %s", thresholds.SectorTiltPercent)
	}
	if !thresholds.CashDragPercent.Equal(decimal.NewFromInt(10)) {
		t.Errorf("Expected cash drag threshold 10, got %s", thresholds.CashDragPercent)
	}
	if thresholds.OverlapAccountCount != 3 {
		t.Errorf("Expected overlap count 3, got %d", thresholds.OverlapAccountCount)
	}
}

func TestAlertDetector_DetectConcentration(t *testing.T) {
	detector := NewAlertDetector()

	p := &Portfolio{
		ID:         uuid.New(),
		TotalValue: decimal.NewFromFloat(100000.00),
		Holdings: []Holding{
			{Ticker: "AAPL", MarketValue: decimal.NewFromFloat(15000.00)}, // 15% - should alert
			{Ticker: "MSFT", MarketValue: decimal.NewFromFloat(8000.00)},  // 8% - ok
			{Ticker: "GOOGL", MarketValue: decimal.NewFromFloat(77000.00)}, // 77% - should alert
		},
	}

	alloc := &AllocationSummary{
		TickerTotals: map[string]decimal.Decimal{
			"AAPL":  decimal.NewFromFloat(15000.00),
			"MSFT":  decimal.NewFromFloat(8000.00),
			"GOOGL": decimal.NewFromFloat(77000.00),
		},
	}

	alerts := detector.DetectAlerts(p, alloc)

	// Should have 2 concentration alerts (AAPL at 15%, GOOGL at 77%)
	concentrationCount := 0
	for _, alert := range alerts {
		if alert.Type == AlertConcentration {
			concentrationCount++
		}
	}

	if concentrationCount != 2 {
		t.Errorf("Expected 2 concentration alerts, got %d", concentrationCount)
	}
}

func TestAlertDetector_DetectOverlap(t *testing.T) {
	detector := NewAlertDetector()

	p := &Portfolio{
		ID:         uuid.New(),
		TotalValue: decimal.NewFromFloat(100000.00),
		Holdings: []Holding{
			{Ticker: "AAPL", AccountName: "IRA", MarketValue: decimal.NewFromFloat(10000.00)},
			{Ticker: "AAPL", AccountName: "401k", MarketValue: decimal.NewFromFloat(10000.00)},
			{Ticker: "AAPL", AccountName: "Brokerage", MarketValue: decimal.NewFromFloat(10000.00)},
			{Ticker: "MSFT", AccountName: "IRA", MarketValue: decimal.NewFromFloat(10000.00)},
			{Ticker: "MSFT", AccountName: "401k", MarketValue: decimal.NewFromFloat(10000.00)},
		},
	}

	alloc := &AllocationSummary{
		TickerTotals: map[string]decimal.Decimal{
			"AAPL": decimal.NewFromFloat(30000.00),
			"MSFT": decimal.NewFromFloat(20000.00),
		},
	}

	alerts := detector.DetectAlerts(p, alloc)

	// Should have 1 overlap alert (AAPL in 3 accounts)
	overlapCount := 0
	for _, alert := range alerts {
		if alert.Type == AlertOverlap {
			overlapCount++
			if len(alert.Holdings) != 1 || alert.Holdings[0] != "AAPL" {
				t.Errorf("Expected AAPL in overlap alert, got %v", alert.Holdings)
			}
		}
	}

	if overlapCount != 1 {
		t.Errorf("Expected 1 overlap alert, got %d", overlapCount)
	}
}

func TestAlertDetector_DetectCashDrag(t *testing.T) {
	detector := NewAlertDetector()

	p := &Portfolio{
		ID:         uuid.New(),
		TotalValue: decimal.NewFromFloat(100000.00),
		Holdings:   []Holding{},
	}

	alloc := &AllocationSummary{
		ByAssetClass: map[AssetClass]AllocationSlice{
			AssetClassCash: {
				Value:      decimal.NewFromFloat(15000.00),
				Percentage: decimal.NewFromFloat(15.00),
			},
			AssetClassEquity: {
				Value:      decimal.NewFromFloat(85000.00),
				Percentage: decimal.NewFromFloat(85.00),
			},
		},
		TickerTotals: map[string]decimal.Decimal{},
	}

	alerts := detector.DetectAlerts(p, alloc)

	// Should have cash drag alert (15% > 10% threshold)
	cashDragCount := 0
	for _, alert := range alerts {
		if alert.Type == AlertCashDrag {
			cashDragCount++
		}
	}

	if cashDragCount != 1 {
		t.Errorf("Expected 1 cash drag alert, got %d", cashDragCount)
	}
}

func TestAlertDetector_DetectSectorTilt(t *testing.T) {
	detector := NewAlertDetector()

	p := &Portfolio{
		ID:         uuid.New(),
		TotalValue: decimal.NewFromFloat(100000.00),
		Holdings:   []Holding{},
	}

	alloc := &AllocationSummary{
		BySector: map[string]AllocationSlice{
			"Technology": {
				Value:      decimal.NewFromFloat(45000.00),
				Percentage: decimal.NewFromFloat(45.00), // > 30% threshold
			},
			"Healthcare": {
				Value:      decimal.NewFromFloat(25000.00),
				Percentage: decimal.NewFromFloat(25.00),
			},
		},
		ByAssetClass: map[AssetClass]AllocationSlice{},
		TickerTotals: map[string]decimal.Decimal{},
	}

	alerts := detector.DetectAlerts(p, alloc)

	sectorTiltCount := 0
	for _, alert := range alerts {
		if alert.Type == AlertSectorTilt {
			sectorTiltCount++
		}
	}

	if sectorTiltCount != 1 {
		t.Errorf("Expected 1 sector tilt alert, got %d", sectorTiltCount)
	}
}

func TestAlertDetector_DetectUnclassified(t *testing.T) {
	detector := NewAlertDetector()

	p := &Portfolio{
		ID:         uuid.New(),
		TotalValue: decimal.NewFromFloat(100000.00),
		Holdings: []Holding{
			{Ticker: "AAPL", AssetClass: AssetClassEquity},
			{Ticker: "UNKNOWN1", AssetClass: AssetClassOther},
			{Ticker: "UNKNOWN2", AssetClass: AssetClassOther},
		},
	}

	alloc := &AllocationSummary{
		ByAssetClass: map[AssetClass]AllocationSlice{},
		TickerTotals: map[string]decimal.Decimal{},
	}

	alerts := detector.DetectAlerts(p, alloc)

	unclassifiedCount := 0
	var unclassifiedAlert *Alert
	for i, alert := range alerts {
		if alert.Type == AlertUnclassified {
			unclassifiedCount++
			unclassifiedAlert = &alerts[i]
		}
	}

	if unclassifiedCount != 1 {
		t.Errorf("Expected 1 unclassified alert, got %d", unclassifiedCount)
	}

	if unclassifiedAlert != nil {
		if len(unclassifiedAlert.Holdings) != 2 {
			t.Errorf("Expected 2 unclassified holdings, got %d", len(unclassifiedAlert.Holdings))
		}
		if unclassifiedAlert.Severity != SeverityCritical {
			t.Errorf("Expected critical severity, got %s", unclassifiedAlert.Severity)
		}
	}
}

func TestAlertDetector_NoAlerts(t *testing.T) {
	detector := NewAlertDetector()

	p := &Portfolio{
		ID:         uuid.New(),
		TotalValue: decimal.NewFromFloat(100000.00),
		Holdings: []Holding{
			{Ticker: "AAPL", AccountName: "IRA", MarketValue: decimal.NewFromFloat(5000.00), AssetClass: AssetClassEquity},
			{Ticker: "MSFT", AccountName: "IRA", MarketValue: decimal.NewFromFloat(5000.00), AssetClass: AssetClassEquity},
		},
	}

	alloc := &AllocationSummary{
		ByAssetClass: map[AssetClass]AllocationSlice{
			AssetClassEquity: {Value: decimal.NewFromFloat(10000.00), Percentage: decimal.NewFromFloat(10.00)},
		},
		BySector:     map[string]AllocationSlice{},
		ByGeography:  map[string]AllocationSlice{},
		TickerTotals: map[string]decimal.Decimal{
			"AAPL": decimal.NewFromFloat(5000.00),
			"MSFT": decimal.NewFromFloat(5000.00),
		},
	}

	alerts := detector.DetectAlerts(p, alloc)

	if len(alerts) != 0 {
		t.Errorf("Expected no alerts, got %d: %+v", len(alerts), alerts)
	}
}
