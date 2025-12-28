package ai

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"sync"
	"time"
)

// QueryCache provides caching for AI responses with semantic similarity
type QueryCache struct {
	cache      map[string]*CacheEntry
	mu         sync.RWMutex
	ttl        time.Duration
	threshold  float64 // Similarity threshold for cache hits
	maxEntries int
}

// CacheEntry represents a cached response
type CacheEntry struct {
	Query       string
	QueryHash   string
	PortfolioID string
	Response    *Response
	Keywords    []string
	CreatedAt   time.Time
	HitCount    int
}

// NewQueryCache creates a new query cache
func NewQueryCache(ttl time.Duration, threshold float64) *QueryCache {
	if ttl == 0 {
		ttl = 1 * time.Hour
	}
	if threshold == 0 {
		threshold = 0.92
	}

	cache := &QueryCache{
		cache:      make(map[string]*CacheEntry),
		ttl:        ttl,
		threshold:  threshold,
		maxEntries: 10000,
	}

	// Start cleanup goroutine
	go cache.cleanupLoop()

	return cache
}

// Get retrieves a cached response if available
func (c *QueryCache) Get(query, portfolioID string) *Response {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Try exact match first
	hash := c.hashQuery(query, portfolioID)
	if entry, ok := c.cache[hash]; ok {
		if time.Since(entry.CreatedAt) < c.ttl {
			entry.HitCount++
			// Clone response to avoid mutation
			return c.cloneResponse(entry.Response)
		}
	}

	// Try semantic similarity match
	normalizedQuery := c.normalizeQuery(query)
	queryKeywords := c.extractKeywords(query)

	for _, entry := range c.cache {
		if entry.PortfolioID != portfolioID {
			continue
		}
		if time.Since(entry.CreatedAt) >= c.ttl {
			continue
		}

		similarity := c.calculateSimilarity(normalizedQuery, queryKeywords, entry)
		if similarity >= c.threshold {
			entry.HitCount++
			return c.cloneResponse(entry.Response)
		}
	}

	return nil
}

// Set stores a response in the cache
func (c *QueryCache) Set(query, portfolioID string, response *Response) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Evict if at capacity
	if len(c.cache) >= c.maxEntries {
		c.evictOldest()
	}

	hash := c.hashQuery(query, portfolioID)
	c.cache[hash] = &CacheEntry{
		Query:       c.normalizeQuery(query),
		QueryHash:   hash,
		PortfolioID: portfolioID,
		Response:    response,
		Keywords:    c.extractKeywords(query),
		CreatedAt:   time.Now(),
		HitCount:    0,
	}
}

// Invalidate removes entries for a portfolio
func (c *QueryCache) Invalidate(portfolioID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for hash, entry := range c.cache {
		if entry.PortfolioID == portfolioID {
			delete(c.cache, hash)
		}
	}
}

// Stats returns cache statistics
func (c *QueryCache) Stats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	totalHits := 0
	for _, entry := range c.cache {
		totalHits += entry.HitCount
	}

	return map[string]interface{}{
		"entries":    len(c.cache),
		"max_size":   c.maxEntries,
		"total_hits": totalHits,
		"ttl":        c.ttl.String(),
	}
}

// hashQuery creates a deterministic hash for a query
func (c *QueryCache) hashQuery(query, portfolioID string) string {
	normalized := c.normalizeQuery(query)
	data := normalized + "|" + portfolioID
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// normalizeQuery standardizes query text for comparison
func (c *QueryCache) normalizeQuery(query string) string {
	// Lowercase
	query = strings.ToLower(query)

	// Remove extra whitespace
	query = strings.Join(strings.Fields(query), " ")

	// Remove punctuation except important financial symbols
	var result strings.Builder
	for _, r := range query {
		if (r >= 'a' && r <= 'z') ||
			(r >= '0' && r <= '9') ||
			r == ' ' || r == '$' || r == '%' {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// extractKeywords extracts important terms from a query
func (c *QueryCache) extractKeywords(query string) []string {
	query = strings.ToLower(query)
	words := strings.Fields(query)

	// Stop words to filter out
	stopWords := map[string]bool{
		"a": true, "an": true, "the": true, "is": true, "are": true,
		"was": true, "were": true, "be": true, "been": true, "being": true,
		"have": true, "has": true, "had": true, "do": true, "does": true,
		"did": true, "will": true, "would": true, "could": true, "should": true,
		"may": true, "might": true, "must": true, "shall": true,
		"i": true, "me": true, "my": true, "we": true, "our": true,
		"you": true, "your": true, "it": true, "its": true,
		"this": true, "that": true, "these": true, "those": true,
		"what": true, "which": true, "who": true, "whom": true,
		"how": true, "when": true, "where": true, "why": true,
		"and": true, "or": true, "but": true, "if": true, "then": true,
		"so": true, "as": true, "of": true, "at": true, "by": true,
		"for": true, "with": true, "about": true, "to": true, "from": true,
		"in": true, "on": true, "can": true, "tell": true, "show": true,
	}

	keywords := make([]string, 0)
	for _, word := range words {
		// Clean word
		word = strings.Trim(word, ".,!?;:'\"")
		if len(word) < 2 {
			continue
		}
		if stopWords[word] {
			continue
		}
		keywords = append(keywords, word)
	}

	return keywords
}

// calculateSimilarity computes similarity between query and cache entry
func (c *QueryCache) calculateSimilarity(normalizedQuery string, queryKeywords []string, entry *CacheEntry) float64 {
	// Jaccard similarity on keywords
	if len(queryKeywords) == 0 || len(entry.Keywords) == 0 {
		return 0
	}

	// Create sets
	querySet := make(map[string]bool)
	for _, kw := range queryKeywords {
		querySet[kw] = true
	}

	entrySet := make(map[string]bool)
	for _, kw := range entry.Keywords {
		entrySet[kw] = true
	}

	// Calculate intersection and union
	intersection := 0
	for kw := range querySet {
		if entrySet[kw] {
			intersection++
		}
	}

	union := len(querySet)
	for kw := range entrySet {
		if !querySet[kw] {
			union++
		}
	}

	if union == 0 {
		return 0
	}

	jaccardSim := float64(intersection) / float64(union)

	// Also consider length similarity
	lenRatio := float64(len(normalizedQuery)) / float64(len(entry.Query))
	if lenRatio > 1 {
		lenRatio = 1 / lenRatio
	}

	// Weighted combination
	return jaccardSim*0.7 + lenRatio*0.3
}

// cloneResponse creates a copy of a response
func (c *QueryCache) cloneResponse(r *Response) *Response {
	if r == nil {
		return nil
	}

	clone := *r
	clone.Sources = make([]Source, len(r.Sources))
	copy(clone.Sources, r.Sources)
	clone.Disclaimers = make([]string, len(r.Disclaimers))
	copy(clone.Disclaimers, r.Disclaimers)

	return &clone
}

// evictOldest removes the oldest cache entries
func (c *QueryCache) evictOldest() {
	// Find entries to evict (oldest 10%)
	evictCount := c.maxEntries / 10
	if evictCount < 1 {
		evictCount = 1
	}

	type entryAge struct {
		hash string
		age  time.Time
	}

	entries := make([]entryAge, 0, len(c.cache))
	for hash, entry := range c.cache {
		entries = append(entries, entryAge{hash: hash, age: entry.CreatedAt})
	}

	// Sort by age (oldest first) - simple bubble sort for small sets
	for i := 0; i < len(entries)-1; i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[j].age.Before(entries[i].age) {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}

	// Evict oldest
	for i := 0; i < evictCount && i < len(entries); i++ {
		delete(c.cache, entries[i].hash)
	}
}

// cleanupLoop periodically removes expired entries
func (c *QueryCache) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.cleanup()
	}
}

// cleanup removes expired entries
func (c *QueryCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for hash, entry := range c.cache {
		if now.Sub(entry.CreatedAt) >= c.ttl {
			delete(c.cache, hash)
		}
	}
}
