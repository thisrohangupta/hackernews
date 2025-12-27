package models

import (
	"testing"
	"time"
)

func TestGetPeriodDuration(t *testing.T) {
	tests := []struct {
		period   string
		expected time.Duration
	}{
		{Period1Day, 24 * time.Hour},
		{Period1Week, 7 * 24 * time.Hour},
		{Period1Month, 30 * 24 * time.Hour},
		{Period3Month, 90 * 24 * time.Hour},
		{Period6Month, 180 * 24 * time.Hour},
		{Period1Year, 365 * 24 * time.Hour},
		{Period3Year, 3 * 365 * 24 * time.Hour},
		{Period5Year, 5 * 365 * 24 * time.Hour},
		{"unknown", 365 * 24 * time.Hour}, // Default to 1 year
	}

	for _, tt := range tests {
		t.Run(tt.period, func(t *testing.T) {
			duration := GetPeriodDuration(tt.period)
			if duration != tt.expected {
				t.Errorf("GetPeriodDuration(%s) = %v, want %v", tt.period, duration, tt.expected)
			}
		})
	}
}

func TestGetPeriodStartDate(t *testing.T) {
	now := time.Now().UTC()

	tests := []struct {
		period       string
		checkFunc    func(start time.Time) bool
		description  string
	}{
		{
			period: Period1Day,
			checkFunc: func(start time.Time) bool {
				diff := now.Sub(start)
				return diff >= 23*time.Hour && diff <= 25*time.Hour
			},
			description: "should be approximately 1 day ago",
		},
		{
			period: Period1Week,
			checkFunc: func(start time.Time) bool {
				diff := now.Sub(start)
				return diff >= 6*24*time.Hour && diff <= 8*24*time.Hour
			},
			description: "should be approximately 1 week ago",
		},
		{
			period: Period1Month,
			checkFunc: func(start time.Time) bool {
				diff := now.Sub(start)
				return diff >= 27*24*time.Hour && diff <= 32*24*time.Hour
			},
			description: "should be approximately 1 month ago",
		},
		{
			period: Period1Year,
			checkFunc: func(start time.Time) bool {
				diff := now.Sub(start)
				return diff >= 360*24*time.Hour && diff <= 370*24*time.Hour
			},
			description: "should be approximately 1 year ago",
		},
		{
			period: PeriodYTD,
			checkFunc: func(start time.Time) bool {
				return start.Year() == now.Year() && start.Month() == 1 && start.Day() == 1
			},
			description: "should be January 1st of current year",
		},
	}

	for _, tt := range tests {
		t.Run(tt.period, func(t *testing.T) {
			start := GetPeriodStartDate(tt.period)
			if !tt.checkFunc(start) {
				t.Errorf("GetPeriodStartDate(%s) = %v: %s", tt.period, start, tt.description)
			}
		})
	}
}

func TestPeriodConstants(t *testing.T) {
	// Verify period constants are defined correctly
	periods := []string{
		Period1Day,
		Period1Week,
		Period1Month,
		Period3Month,
		Period6Month,
		Period1Year,
		Period3Year,
		Period5Year,
		PeriodYTD,
		PeriodAll,
	}

	for _, p := range periods {
		if p == "" {
			t.Error("Period constant should not be empty")
		}
	}
}
