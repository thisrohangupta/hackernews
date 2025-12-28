package ai

import (
	"encoding/json"
	"log"
	"os"
	"sync"
	"time"
)

// AuditLogger provides compliance audit logging for AI interactions
type AuditLogger struct {
	enabled bool
	entries []AuditEntry
	mu      sync.Mutex
	writer  *log.Logger
}

// AuditEntry represents a logged AI interaction
type AuditEntry struct {
	ID            string      `json:"id"`
	Timestamp     time.Time   `json:"timestamp"`
	UserID        string      `json:"user_id"`
	QueryID       string      `json:"query_id"`
	QueryText     string      `json:"query_text"`
	QueryIntent   QueryIntent `json:"query_intent"`
	ResponseID    string      `json:"response_id"`
	ResponseModel Model       `json:"response_model"`
	TokensUsed    TokenUsage  `json:"tokens_used"`
	Cached        bool        `json:"cached"`
	ProcessingMs  int64       `json:"processing_ms"`
	Sources       []string    `json:"sources"`
	Disclaimers   []string    `json:"disclaimers"`
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(enabled bool) *AuditLogger {
	al := &AuditLogger{
		enabled: enabled,
		entries: make([]AuditEntry, 0),
		writer:  log.New(os.Stdout, "[AI-AUDIT] ", log.LstdFlags),
	}

	return al
}

// Log records an AI interaction
func (al *AuditLogger) Log(query *Query, response *Response) {
	if !al.enabled {
		return
	}

	// Extract source references
	sources := make([]string, len(response.Sources))
	for i, src := range response.Sources {
		sources[i] = src.Reference
	}

	entry := AuditEntry{
		ID:            response.ID,
		Timestamp:     time.Now().UTC(),
		UserID:        query.UserID,
		QueryID:       query.ID,
		QueryText:     query.Text,
		QueryIntent:   response.Intent,
		ResponseID:    response.ID,
		ResponseModel: response.Model,
		TokensUsed:    response.TokensUsed,
		Cached:        response.Cached,
		ProcessingMs:  response.ProcessingMs,
		Sources:       sources,
		Disclaimers:   response.Disclaimers,
	}

	al.mu.Lock()
	al.entries = append(al.entries, entry)
	al.mu.Unlock()

	// Log to stdout in JSON format for log aggregation
	if data, err := json.Marshal(entry); err == nil {
		al.writer.Println(string(data))
	}
}

// GetEntries returns audit entries for a user (for compliance review)
func (al *AuditLogger) GetEntries(userID string, since time.Time, limit int) []AuditEntry {
	al.mu.Lock()
	defer al.mu.Unlock()

	results := make([]AuditEntry, 0)
	for i := len(al.entries) - 1; i >= 0 && len(results) < limit; i-- {
		entry := al.entries[i]
		if entry.UserID == userID && entry.Timestamp.After(since) {
			results = append(results, entry)
		}
	}

	return results
}

// GetStats returns audit statistics
func (al *AuditLogger) GetStats(since time.Time) map[string]interface{} {
	al.mu.Lock()
	defer al.mu.Unlock()

	totalQueries := 0
	cachedQueries := 0
	totalTokens := 0
	intentCounts := make(map[QueryIntent]int)
	modelCounts := make(map[Model]int)

	for _, entry := range al.entries {
		if entry.Timestamp.Before(since) {
			continue
		}

		totalQueries++
		if entry.Cached {
			cachedQueries++
		}
		totalTokens += entry.TokensUsed.Total
		intentCounts[entry.QueryIntent]++
		modelCounts[entry.ResponseModel]++
	}

	cacheHitRate := 0.0
	if totalQueries > 0 {
		cacheHitRate = float64(cachedQueries) / float64(totalQueries) * 100
	}

	return map[string]interface{}{
		"total_queries":  totalQueries,
		"cached_queries": cachedQueries,
		"cache_hit_rate": cacheHitRate,
		"total_tokens":   totalTokens,
		"by_intent":      intentCounts,
		"by_model":       modelCounts,
	}
}

// ExportForCompliance exports entries in a compliance-friendly format
func (al *AuditLogger) ExportForCompliance(userID string, startDate, endDate time.Time) ([]byte, error) {
	al.mu.Lock()
	defer al.mu.Unlock()

	entries := make([]AuditEntry, 0)
	for _, entry := range al.entries {
		if entry.UserID != userID {
			continue
		}
		if entry.Timestamp.Before(startDate) || entry.Timestamp.After(endDate) {
			continue
		}
		entries = append(entries, entry)
	}

	export := struct {
		UserID      string       `json:"user_id"`
		StartDate   time.Time    `json:"start_date"`
		EndDate     time.Time    `json:"end_date"`
		EntryCount  int          `json:"entry_count"`
		ExportedAt  time.Time    `json:"exported_at"`
		Entries     []AuditEntry `json:"entries"`
		Disclaimer  string       `json:"disclaimer"`
	}{
		UserID:     userID,
		StartDate:  startDate,
		EndDate:    endDate,
		EntryCount: len(entries),
		ExportedAt: time.Now().UTC(),
		Entries:    entries,
		Disclaimer: "This audit log contains AI-generated content for informational purposes only. " +
			"All responses include appropriate disclaimers and do not constitute investment advice.",
	}

	return json.MarshalIndent(export, "", "  ")
}

// Clear removes old entries (for memory management)
func (al *AuditLogger) Clear(before time.Time) int {
	al.mu.Lock()
	defer al.mu.Unlock()

	newEntries := make([]AuditEntry, 0)
	removed := 0

	for _, entry := range al.entries {
		if entry.Timestamp.Before(before) {
			removed++
		} else {
			newEntries = append(newEntries, entry)
		}
	}

	al.entries = newEntries
	return removed
}
