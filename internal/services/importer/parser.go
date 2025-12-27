// Package importer handles CSV import and parsing
package importer

import (
	"encoding/csv"
	"errors"
	"io"
	"strings"

	"github.com/findosh/truenorth/internal/models"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var (
	ErrUnknownFormat = errors.New("unknown CSV format")
	ErrEmptyFile     = errors.New("CSV file is empty")
	ErrNoData        = errors.New("no valid holdings found")
)

// CSVParser interface for brokerage-specific implementations
type CSVParser interface {
	// Parse reads CSV data and returns normalized holdings
	Parse(reader io.Reader, portfolioID uuid.UUID, accountName string) ([]models.Holding, error)

	// Detect checks if this parser handles the given CSV format
	Detect(header []string) bool

	// Name returns the parser name
	Name() string
}

// ParseResult contains the result of parsing a CSV file
type ParseResult struct {
	Holdings    []models.Holding
	Source      string
	AccountName string
	Errors      []string
}

// Service handles CSV import operations
type Service struct {
	parsers []CSVParser
	tagger  *Tagger
}

// NewService creates a new import service
func NewService() *Service {
	return &Service{
		parsers: []CSVParser{
			NewSchwabParser(),
			NewFidelityParser(),
			NewVanguardParser(),
		},
		tagger: NewTagger(),
	}
}

// ParseCSV auto-detects the format and parses the CSV
func (s *Service) ParseCSV(reader io.Reader, portfolioID uuid.UUID, accountName string) (*ParseResult, error) {
	// Read all data first
	csvReader := csv.NewReader(reader)
	csvReader.FieldsPerRecord = -1 // Allow variable fields
	csvReader.TrimLeadingSpace = true

	records, err := csvReader.ReadAll()
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return nil, ErrEmptyFile
	}

	// Find header row (might not be first row)
	headerIdx, header := findHeader(records)
	if headerIdx < 0 {
		return nil, ErrUnknownFormat
	}

	// Detect parser
	var parser CSVParser
	for _, p := range s.parsers {
		if p.Detect(header) {
			parser = p
			break
		}
	}

	if parser == nil {
		return nil, ErrUnknownFormat
	}

	// Create a reader from remaining records
	dataRecords := records[headerIdx:]
	holdings, err := s.parseRecords(parser, dataRecords, portfolioID, accountName)
	if err != nil {
		return nil, err
	}

	if len(holdings) == 0 {
		return nil, ErrNoData
	}

	// Auto-tag holdings
	s.tagger.TagHoldings(holdings)

	return &ParseResult{
		Holdings:    holdings,
		Source:      parser.Name(),
		AccountName: accountName,
	}, nil
}

func findHeader(records [][]string) (int, []string) {
	// Common header keywords
	keywords := []string{"symbol", "ticker", "description", "quantity", "shares", "price", "value"}

	for i, row := range records {
		if len(row) < 3 {
			continue
		}
		// Check if row contains header keywords
		rowStr := strings.ToLower(strings.Join(row, " "))
		matches := 0
		for _, kw := range keywords {
			if strings.Contains(rowStr, kw) {
				matches++
			}
		}
		if matches >= 2 {
			return i, row
		}
	}
	return -1, nil
}

func (s *Service) parseRecords(parser CSVParser, records [][]string, portfolioID uuid.UUID, accountName string) ([]models.Holding, error) {
	var holdings []models.Holding

	// Skip header row
	for i := 1; i < len(records); i++ {
		row := records[i]
		if len(row) < 3 {
			continue
		}

		// Check for empty or summary rows
		if isSkipRow(row) {
			continue
		}

		holding := parseRow(parser, row, portfolioID, accountName)
		if holding != nil {
			holdings = append(holdings, *holding)
		}
	}

	return holdings, nil
}

func isSkipRow(row []string) bool {
	if len(row) == 0 {
		return true
	}

	// Skip total/summary rows
	firstCell := strings.ToLower(strings.TrimSpace(row[0]))
	skipPrefixes := []string{"total", "account total", "cash", "--", "***", ""}

	for _, prefix := range skipPrefixes {
		if strings.HasPrefix(firstCell, prefix) && len(firstCell) < 20 {
			return true
		}
	}

	return false
}

func parseRow(parser CSVParser, row []string, portfolioID uuid.UUID, accountName string) *models.Holding {
	// This is a simplified common parser
	// Each specific parser handles the actual parsing
	return nil
}

// Helper functions for parsing values

func parseDecimal(s string) decimal.Decimal {
	// Clean up the string
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ",", "")
	s = strings.ReplaceAll(s, "$", "")
	s = strings.ReplaceAll(s, "%", "")

	// Handle parentheses for negative numbers
	if strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")") {
		s = "-" + s[1:len(s)-1]
	}

	// Handle empty or invalid
	if s == "" || s == "--" || s == "n/a" || s == "N/A" {
		return decimal.Zero
	}

	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero
	}
	return d
}

func cleanTicker(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ToUpper(s)

	// Remove common suffixes (loop to handle multiple)
	for {
		trimmed := s
		trimmed = strings.TrimSuffix(trimmed, " ")
		trimmed = strings.TrimSuffix(trimmed, "*")
		if trimmed == s {
			break
		}
		s = trimmed
	}

	return s
}

func cleanName(s string) string {
	s = strings.TrimSpace(s)
	// Truncate very long names
	if len(s) > 100 {
		s = s[:100] + "..."
	}
	return s
}
