# Claude.md - Engineering Architecture Guide

## Project Overview

This is a **Go-based static site generator** that creates a HackerNews clone by fetching top stories from the HackerNews API and rendering them as a static HTML website.

**Primary Goal:** Provide a clean, fast, and maintainable news aggregation platform that can serve as a foundation for financial news and investment content curation.

---

## Architecture Principles

### 1. Simplicity First
- **Every function should do one thing well**
- Avoid premature abstraction - only refactor when patterns emerge
- Prefer explicit code over clever code
- Any developer should understand the codebase within 30 minutes

### 2. Clean Separation of Concerns
```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   Data Layer    │ ──▶ │  Business Logic │ ──▶ │  Presentation   │
│  (API/Storage)  │     │  (Processing)   │     │  (Templates)    │
└─────────────────┘     └─────────────────┘     └─────────────────┘
```

### 3. Fail Gracefully
- Network errors should not crash the application
- Missing data should be handled with sensible defaults
- Always log errors with context for debugging

### 4. Security by Default
- Escape all user-generated content (titles, authors)
- Use HTTPS for all external API calls
- No client-side JavaScript injection risks

---

## Current Architecture

### Directory Structure
```
hackernews/
├── main.go              # Core application logic
├── go.mod               # Go module definition
├── Claude.md            # This architecture guide
├── README.md            # User documentation
├── templates/
│   └── index.html       # HTML template
├── static/
│   └── style.css        # Stylesheet
└── public/              # Generated output (gitignored)
    ├── index.html
    └── static/
        └── style.css
```

### Data Flow
```
HackerNews API → Fetch Stories → Process Data → Render HTML → Write Files
```

### Key Data Structures

```go
// Story represents a single news item
// In a fintech context, this could represent market news, SEC filings, or earnings reports
type Story struct {
    ID    int    // Unique identifier
    Title string // Headline
    By    string // Author/Source
    Score int    // Relevance score (upvotes)
    URL   string // Source link
    Time  int64  // Unix timestamp
}

// PageData contains all data needed to render a page
type PageData struct {
    Stories template.HTML // Pre-rendered HTML content
}
```

---

## Coding Standards

### Naming Conventions
| Type | Convention | Example |
|------|------------|---------|
| Functions | camelCase, verb-first | `fetchStories`, `renderHTML` |
| Types/Structs | PascalCase, noun | `Story`, `PageData` |
| Constants | PascalCase | `MaxStories`, `APIBaseURL` |
| Variables | camelCase, descriptive | `storyList`, `httpClient` |
| Files | lowercase, descriptive | `main.go`, `api_client.go` |

### Function Design
```go
// Good: Single responsibility, clear name, handles errors
func fetchStory(id int) (*Story, error) {
    // Implementation
}

// Bad: Does too much, unclear name
func getData() {
    // Fetches, processes, and saves - too many responsibilities
}
```

### Error Handling
```go
// Always wrap errors with context
if err != nil {
    return fmt.Errorf("failed to fetch story %d: %w", id, err)
}

// Log errors at the appropriate level
log.Printf("Warning: skipping story %d due to error: %v", id, err)
```

### Comments
- **Do** comment on WHY, not WHAT
- **Do** document public functions and types
- **Don't** add obvious comments that repeat the code

```go
// Good: Explains business logic
// Score threshold filters out low-engagement stories to maintain quality
if story.Score < MinScoreThreshold {
    continue
}

// Bad: States the obvious
// Increment i by 1
i++
```

---

## Refactoring Guidelines

### Phase 1: Code Organization
Split `main.go` into logical modules:

```
hackernews/
├── cmd/
│   └── generator/
│       └── main.go          # Entry point only
├── internal/
│   ├── api/
│   │   └── hackernews.go    # API client
│   ├── models/
│   │   └── story.go         # Data structures
│   ├── renderer/
│   │   └── html.go          # Template rendering
│   └── storage/
│       └── filesystem.go    # File operations
├── templates/
├── static/
└── public/
```

### Phase 2: Configuration
Extract hardcoded values into configuration:

```go
type Config struct {
    APIBaseURL    string        // HackerNews API endpoint
    MaxStories    int           // Number of stories to fetch
    OutputDir     string        // Where to write generated files
    TemplateDir   string        // Template location
    RequestTimeout time.Duration // HTTP timeout
}
```

### Phase 3: Testability
Design for testing:

```go
// Use interfaces for external dependencies
type StoryFetcher interface {
    FetchTopStories(limit int) ([]Story, error)
    FetchStory(id int) (*Story, error)
}

// Inject dependencies
type Generator struct {
    fetcher  StoryFetcher
    renderer Renderer
    writer   FileWriter
}
```

---

## API Integration

### HackerNews API
- **Base URL:** `https://hacker-news.firebaseio.com/v0/`
- **Endpoints:**
  - `GET /topstories.json` - List of top story IDs
  - `GET /item/{id}.json` - Individual story details
- **Rate Limiting:** Be respectful, add delays between requests if fetching many items
- **Error Handling:** API may return null for deleted/dead stories

### Fintech Extension Points
For financial news integration, consider these data sources:
- SEC EDGAR API for filings
- Alpha Vantage for market data
- NewsAPI for financial news
- RSS feeds from financial publications

---

## Security Considerations

### Input Sanitization
All external data (API responses) must be sanitized before rendering:

```go
// Use html/template for automatic escaping
// Never use text/template for user-facing content
import "html/template"

// Explicitly escape when building HTML strings
title := template.HTMLEscapeString(story.Title)
```

### Sensitive Data
- Never commit API keys or secrets
- Use environment variables for configuration
- Keep `public/` directory out of version control if it contains any dynamic content

---

## Performance Guidelines

### HTTP Client Best Practices
```go
// Reuse HTTP client - don't create new ones per request
var httpClient = &http.Client{
    Timeout: 10 * time.Second,
}

// Consider connection pooling for high-volume fetching
transport := &http.Transport{
    MaxIdleConns:        100,
    MaxIdleConnsPerHost: 100,
}
```

### Concurrent Fetching
For fetching multiple stories, use goroutines with proper synchronization:

```go
// Use worker pool pattern for controlled concurrency
// Limit concurrent requests to avoid overwhelming the API
const maxConcurrent = 5
```

---

## Testing Strategy

### Unit Tests
- Test each function in isolation
- Mock external dependencies (HTTP client, filesystem)
- Cover edge cases: empty responses, malformed data, network errors

### Integration Tests
- Test API integration with real endpoints (sparingly)
- Verify generated HTML is valid
- Check file system operations

### Test File Naming
```
api/
├── hackernews.go
└── hackernews_test.go
```

---

## Development Workflow

### Running the Generator
```bash
go run main.go
# or after refactoring:
go run cmd/generator/main.go
```

### Building
```bash
go build -o hackernews main.go
./hackernews
```

### Testing
```bash
go test ./...
go test -v -cover ./...
```

### Linting
```bash
go vet ./...
gofmt -s -w .
```

---

## Future Considerations

### Potential Enhancements
1. **Caching Layer** - Store fetched stories to reduce API calls
2. **Incremental Generation** - Only update changed content
3. **Multiple Output Formats** - JSON feed, RSS, Markdown
4. **Scheduling** - Automated regeneration via cron or systemd timer
5. **Theming** - Dark mode, custom color schemes
6. **Search** - Client-side or static search index

### Fintech-Specific Features
1. **Ticker Detection** - Highlight stock symbols in titles
2. **Sentiment Analysis** - Tag stories as bullish/bearish
3. **Watchlist Integration** - Filter stories by relevant tickers
4. **Market Hours Indicator** - Show if markets are open
5. **Price Widgets** - Display current prices for mentioned securities

---

## Quick Reference

### Common Tasks

| Task | Command |
|------|---------|
| Generate site | `go run main.go` |
| Run tests | `go test ./...` |
| Format code | `gofmt -s -w .` |
| Check errors | `go vet ./...` |
| Build binary | `go build -o hackernews` |

### Important Files
- `main.go` - All application logic
- `templates/index.html` - Page template
- `static/style.css` - Styles
- `public/` - Generated output

### External Resources
- [HackerNews API Docs](https://github.com/HackerNews/API)
- [Go html/template](https://pkg.go.dev/html/template)
- [Effective Go](https://go.dev/doc/effective_go)

---

*This document should be updated as the architecture evolves. When in doubt, favor simplicity and clarity over complexity.*
