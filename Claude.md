# Claude.md - FinDosh TrueNorth Architecture Guide

## Product Overview

**TrueNorth** is a unified portfolio intelligence platform for sophisticated DIY investors managing $5-10MM+ across multiple accounts. It provides real-time performance tracking, integrated risk-reward analytics, and what-if scenario modeling—without robo-advisory.

### Core Value Propositions
- **Time Savings:** Eliminate hours of monthly account reconciliation
- **Unified Visibility:** True asset allocation across all accounts with zero "Unknown" categories
- **Actionable Intelligence:** Multi-class risk-reward matrix for informed decisions
- **DIY Control:** What-if scenarios without algorithmic trading

### Target User
```
Persona: "Sophisticated DIY Investor"
Portfolio: $5-10MM across 5+ accounts
Pain: Fragmented views, manual reconciliation, no cross-class analytics
Goal: Superior risk-adjusted returns with full process control
```

---

## Architecture Principles

### 1. Security First (Non-Negotiable)
```
┌─────────────────────────────────────────────────────────────┐
│  SECURITY REQUIREMENTS                                       │
├─────────────────────────────────────────────────────────────┤
│  • 256-bit AES encryption at rest                           │
│  • TLS 1.3 for all data in transit                          │
│  • Read-only data model - NO trade execution                │
│  • NO credential storage - CSV upload only (MVP)            │
│  • MFA required for all accounts                            │
│  • SOC 2 Type II compliance roadmap                         │
└─────────────────────────────────────────────────────────────┘
```

### 2. Simplicity & Clarity
- Every function does one thing well
- Any developer should understand any module within 30 minutes
- Explicit code over clever abstractions
- Document the WHY, not the WHAT

### 3. Financial Data Accuracy
- All monetary values use `decimal` types, never floating point
- Currency always explicit (USD default)
- Timestamps in UTC with timezone awareness
- Audit trail for all data modifications

### 4. Graceful Degradation
- Network failures don't crash the application
- Missing data shows clear indicators, not errors
- Partial uploads should be resumable

---

## Tech Stack

### Backend (Go)
```
Language:    Go 1.21+
Framework:   Standard library + minimal dependencies
Database:    SQLite (MVP) → PostgreSQL (Production)
Auth:        JWT with secure httpOnly cookies
Encryption:  AES-256-GCM for sensitive data
```

### Frontend
```
Approach:    Server-rendered HTML + minimal JavaScript
Templates:   Go html/template with components
Styling:     CSS with custom properties (theming support)
Charts:      Chart.js or D3.js for visualizations
```

### Why This Stack?
- **Go:** Fast, secure, excellent for financial applications
- **SQLite → PostgreSQL:** Simple start, scales when needed
- **Server-rendered:** Faster initial load, better security, simpler architecture
- **Minimal JS:** Reduces attack surface, improves reliability

---

## Project Structure

```
truenorth/
├── cmd/
│   └── server/
│       └── main.go                 # Application entry point
│
├── internal/
│   ├── config/
│   │   └── config.go               # Configuration management
│   │
│   ├── models/                     # Core domain models
│   │   ├── user.go                 # User account
│   │   ├── portfolio.go            # OmniFolio (o'Folio)
│   │   ├── holding.go              # Individual position
│   │   ├── asset.go                # Asset metadata (ticker info)
│   │   └── scenario.go             # What-if scenarios
│   │
│   ├── services/                   # Business logic
│   │   ├── auth/
│   │   │   └── auth.go             # Authentication service
│   │   ├── portfolio/
│   │   │   ├── aggregator.go       # Multi-account aggregation
│   │   │   ├── allocation.go       # Allocation calculations
│   │   │   └── performance.go      # Performance metrics
│   │   ├── import/
│   │   │   ├── csv_parser.go       # CSV parsing engine
│   │   │   ├── schwab.go           # Schwab format handler
│   │   │   ├── fidelity.go         # Fidelity format handler
│   │   │   ├── vanguard.go         # Vanguard format handler
│   │   │   └── tagger.go           # AI ticker classification
│   │   ├── analytics/
│   │   │   ├── risk_reward.go      # R2R matrix calculations
│   │   │   ├── alerts.go           # Concentration/overlap alerts
│   │   │   └── scenarios.go        # What-if simulations
│   │   └── market/
│   │       ├── prices.go           # Real-time price service
│   │       └── news.go             # News aggregation
│   │
│   ├── handlers/                   # HTTP handlers
│   │   ├── auth.go                 # Login/logout/register
│   │   ├── dashboard.go            # Main dashboard
│   │   ├── import.go               # CSV upload endpoints
│   │   ├── portfolio.go            # Portfolio views
│   │   ├── scenarios.go            # What-if interface
│   │   └── api.go                  # JSON API endpoints
│   │
│   ├── middleware/
│   │   ├── auth.go                 # Authentication middleware
│   │   ├── security.go             # Security headers, CSRF
│   │   └── logging.go              # Request logging
│   │
│   └── storage/
│       ├── database.go             # Database connection
│       ├── users.go                # User repository
│       ├── portfolios.go           # Portfolio repository
│       └── migrations/             # Schema migrations
│
├── web/
│   ├── templates/
│   │   ├── layouts/
│   │   │   └── base.html           # Base layout with nav
│   │   ├── pages/
│   │   │   ├── login.html
│   │   │   ├── dashboard.html
│   │   │   ├── import.html
│   │   │   ├── portfolio.html
│   │   │   └── scenarios.html
│   │   └── components/
│   │       ├── allocation_chart.html
│   │       ├── holdings_table.html
│   │       ├── alert_badge.html
│   │       └── scenario_slider.html
│   │
│   └── static/
│       ├── css/
│       │   ├── main.css
│       │   └── charts.css
│       ├── js/
│       │   ├── charts.js
│       │   └── scenarios.js
│       └── images/
│
├── testdata/                       # Sample CSV files for testing
│   ├── schwab_sample.csv
│   ├── fidelity_sample.csv
│   └── vanguard_sample.csv
│
├── go.mod
├── go.sum
├── Claude.md                       # This file
└── README.md
```

---

## Core Data Models

### User
```go
// User represents an authenticated investor
type User struct {
    ID           uuid.UUID  `json:"id"`
    Email        string     `json:"email"`
    PasswordHash string     `json:"-"`  // Never serialize
    Name         string     `json:"name"`
    MFAEnabled   bool       `json:"mfa_enabled"`
    CreatedAt    time.Time  `json:"created_at"`
    UpdatedAt    time.Time  `json:"updated_at"`
}
```

### Portfolio (OmniFolio)
```go
// Portfolio represents a unified view across all accounts
type Portfolio struct {
    ID          uuid.UUID   `json:"id"`
    UserID      uuid.UUID   `json:"user_id"`
    Name        string      `json:"name"`        // e.g., "Family Portfolio"
    Holdings    []Holding   `json:"holdings"`
    TotalValue  Decimal     `json:"total_value"` // Calculated
    FreeCash    Decimal     `json:"free_cash"`
    LastUpdated time.Time   `json:"last_updated"`
}
```

### Holding
```go
// Holding represents a single position in an account
type Holding struct {
    ID            uuid.UUID   `json:"id"`
    PortfolioID   uuid.UUID   `json:"portfolio_id"`
    AccountName   string      `json:"account_name"`   // e.g., "Schwab IRA"
    Ticker        string      `json:"ticker"`         // e.g., "AAPL"
    Name          string      `json:"name"`           // e.g., "Apple Inc."
    Quantity      Decimal     `json:"quantity"`
    CostBasis     Decimal     `json:"cost_basis"`
    CurrentPrice  Decimal     `json:"current_price"`
    MarketValue   Decimal     `json:"market_value"`

    // Classification (AI-tagged or manual)
    AssetClass    AssetClass  `json:"asset_class"`    // Equity, FixedIncome, Alternative, Cash
    Sector        string      `json:"sector"`         // Technology, Healthcare, etc.
    Geography     string      `json:"geography"`      // US, International, Emerging

    // Metadata
    IsManualEntry bool        `json:"is_manual_entry"`
    Source        string      `json:"source"`         // "schwab_csv", "manual"
    ImportedAt    time.Time   `json:"imported_at"`
}

// AssetClass enumeration
type AssetClass string

const (
    AssetClassEquity      AssetClass = "equity"
    AssetClassFixedIncome AssetClass = "fixed_income"
    AssetClassAlternative AssetClass = "alternative"  // PE, VC, Real Estate
    AssetClassCrypto      AssetClass = "crypto"
    AssetClassCash        AssetClass = "cash"
    AssetClassOther       AssetClass = "other"        // Should be zero in final view
)
```

### Allocation Summary
```go
// AllocationSummary provides portfolio breakdown
type AllocationSummary struct {
    ByAssetClass map[AssetClass]AllocationSlice `json:"by_asset_class"`
    BySector     map[string]AllocationSlice     `json:"by_sector"`
    ByGeography  map[string]AllocationSlice     `json:"by_geography"`
    TopHoldings  []HoldingSummary               `json:"top_holdings"`  // Top 10
    TickerTotals map[string]Decimal             `json:"ticker_totals"` // Cross-account
}

type AllocationSlice struct {
    Value      Decimal `json:"value"`
    Percentage Decimal `json:"percentage"`
    Count      int     `json:"count"`  // Number of positions
}
```

### Scenario
```go
// Scenario represents a what-if allocation model
type Scenario struct {
    ID           uuid.UUID                    `json:"id"`
    PortfolioID  uuid.UUID                    `json:"portfolio_id"`
    Name         string                       `json:"name"`
    Allocations  map[AssetClass]Decimal       `json:"allocations"` // Target %
    Projections  ScenarioProjections          `json:"projections"`
    CreatedAt    time.Time                    `json:"created_at"`
}

type ScenarioProjections struct {
    BestCase    Decimal `json:"best_case"`    // Best year return
    WorstCase   Decimal `json:"worst_case"`   // Worst year return
    AverageCase Decimal `json:"average_case"` // Average return
    MaxDrawdown Decimal `json:"max_drawdown"` // Maximum drawdown
}
```

---

## CSV Import System

### Supported Brokerages (MVP)
| Brokerage | Format | Key Columns |
|-----------|--------|-------------|
| Schwab | CSV | Symbol, Description, Quantity, Price, Market Value |
| Fidelity | CSV | Symbol, Description, Quantity, Last Price, Current Value |
| Vanguard | CSV | Symbol, Investment Name, Shares, Share Price, Total Value |

### Parser Architecture
```go
// CSVParser interface for brokerage-specific implementations
type CSVParser interface {
    // Parse reads CSV data and returns normalized holdings
    Parse(reader io.Reader) ([]Holding, error)

    // Detect checks if this parser handles the given CSV format
    Detect(header []string) bool
}

// Import flow:
// 1. User uploads CSV
// 2. System detects brokerage format
// 3. Parser extracts holdings
// 4. Tagger classifies each ticker (asset class, sector, geography)
// 5. User reviews/corrects classifications
// 6. Holdings saved to portfolio
```

### Ticker Tagger
```go
// TickerTagger classifies securities by asset class, sector, geography
type TickerTagger interface {
    // Tag returns classification for a ticker
    Tag(ticker string) (*TickerClassification, error)

    // TagBatch efficiently classifies multiple tickers
    TagBatch(tickers []string) (map[string]*TickerClassification, error)
}

type TickerClassification struct {
    Ticker     string     `json:"ticker"`
    AssetClass AssetClass `json:"asset_class"`
    Sector     string     `json:"sector"`
    Geography  string     `json:"geography"`
    Confidence float64    `json:"confidence"` // 0.0 - 1.0
}
```

---

## Risk-Reward Matrix (R2R)

### Calculation Methodology
```go
// RiskRewardScore for a single holding or asset class
type RiskRewardScore struct {
    ExpectedReturn Decimal `json:"expected_return"` // Annualized
    Volatility     Decimal `json:"volatility"`      // Standard deviation
    SharpeRatio    Decimal `json:"sharpe_ratio"`    // Risk-adjusted return
    MaxDrawdown    Decimal `json:"max_drawdown"`    // Historical worst
    Beta           Decimal `json:"beta"`            // Market correlation
}

// Historical data periods
const (
    Period1Year  = "1y"
    Period3Year  = "3y"
    Period5Year  = "5y"  // Default for analysis
    Period10Year = "10y"
)
```

### Matrix Visualization
```
                    HIGH RETURN
                         │
         ┌───────────────┼───────────────┐
         │   Aggressive  │   Optimal     │
         │   Growth      │   Growth      │
         │               │               │
LOW RISK─┼───────────────┼───────────────┼─HIGH RISK
         │               │               │
         │   Capital     │   High Risk   │
         │   Preservation│   Speculation │
         └───────────────┼───────────────┘
                         │
                    LOW RETURN
```

---

## Alert System

### Alert Types
```go
type AlertType string

const (
    AlertConcentration  AlertType = "concentration"   // >10% in single ticker
    AlertOverlap        AlertType = "overlap"         // Same ticker in 3+ accounts
    AlertHighExpense    AlertType = "high_expense"    // Expense ratio >1%
    AlertUnclassified   AlertType = "unclassified"    // Holdings in "Other"
    AlertCashDrag       AlertType = "cash_drag"       // >10% in cash
    AlertSectorTilt     AlertType = "sector_tilt"     // >30% in single sector
)

type Alert struct {
    Type        AlertType `json:"type"`
    Severity    string    `json:"severity"`  // "info", "warning", "critical"
    Message     string    `json:"message"`
    Holdings    []string  `json:"holdings"`  // Affected tickers
    Suggestion  string    `json:"suggestion"`
}
```

### Thresholds (Configurable)
| Alert | Default Threshold | Severity |
|-------|-------------------|----------|
| Single Position Concentration | >10% | Warning |
| Sector Concentration | >30% | Warning |
| Cash Drag | >10% | Info |
| Expense Ratio | >1% | Warning |
| Unclassified Holdings | >0 | Critical |

---

## Security Implementation

### Authentication Flow
```
┌──────────┐      ┌──────────┐      ┌──────────┐
│  Login   │ ───▶ │  Verify  │ ───▶ │  Issue   │
│  Form    │      │  Creds   │      │  JWT     │
└──────────┘      └──────────┘      └──────────┘
                                          │
                                          ▼
┌──────────┐      ┌──────────┐      ┌──────────┐
│  Access  │ ◀─── │  Verify  │ ◀─── │  Cookie  │
│  Granted │      │  Token   │      │ httpOnly │
└──────────┘      └──────────┘      └──────────┘
```

### Security Headers
```go
// Required security headers for all responses
func SecurityHeaders(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("X-Frame-Options", "DENY")
        w.Header().Set("X-XSS-Protection", "1; mode=block")
        w.Header().Set("Strict-Transport-Security", "max-age=31536000")
        w.Header().Set("Content-Security-Policy", "default-src 'self'")
        next.ServeHTTP(w, r)
    })
}
```

### Data Encryption
```go
// Sensitive fields encrypted at rest
// - User email (for privacy)
// - Portfolio names
// - Account names
// - Any PII

func EncryptField(plaintext string, key []byte) (string, error) {
    // AES-256-GCM encryption
}

func DecryptField(ciphertext string, key []byte) (string, error) {
    // AES-256-GCM decryption
}
```

---

## API Endpoints

### Authentication
| Method | Path | Description |
|--------|------|-------------|
| POST | `/auth/register` | Create new account |
| POST | `/auth/login` | Login, receive JWT |
| POST | `/auth/logout` | Invalidate session |
| POST | `/auth/mfa/enable` | Enable MFA |
| POST | `/auth/mfa/verify` | Verify MFA code |

### Portfolio
| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/portfolios` | List user's portfolios |
| POST | `/api/portfolios` | Create new portfolio |
| GET | `/api/portfolios/:id` | Get portfolio details |
| GET | `/api/portfolios/:id/allocation` | Get allocation breakdown |
| GET | `/api/portfolios/:id/alerts` | Get active alerts |

### Import
| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/import/upload` | Upload CSV file |
| GET | `/api/import/:id/preview` | Preview parsed holdings |
| POST | `/api/import/:id/confirm` | Confirm and save import |
| PUT | `/api/holdings/:id` | Edit holding classification |

### Scenarios
| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/scenarios` | List saved scenarios |
| POST | `/api/scenarios` | Create new scenario |
| POST | `/api/scenarios/simulate` | Run simulation (no save) |

### Market Data
| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/prices/:ticker` | Get current price |
| GET | `/api/prices/batch` | Get multiple prices |
| GET | `/api/news` | Get portfolio-relevant news |

---

## Testing Strategy

### Test Categories
```
tests/
├── unit/           # Function-level tests
├── integration/    # API endpoint tests
├── e2e/            # Full user flow tests
└── fixtures/       # Test data
```

### Critical Test Cases
1. **CSV Parsing:** Each brokerage format with edge cases
2. **Allocation Calculation:** Decimal precision, rounding
3. **Alert Detection:** All threshold scenarios
4. **Authentication:** Login, logout, session expiry, MFA
5. **Data Encryption:** Encrypt/decrypt cycle integrity

### Test Data Requirements
- Sample CSVs for each supported brokerage
- Holdings with various asset classes
- Edge cases: empty files, malformed data, special characters

---

## Development Workflow

### Commands
```bash
# Run development server (hot reload)
go run cmd/server/main.go

# Run tests
go test ./...

# Run with coverage
go test -cover ./...

# Format code
gofmt -s -w .

# Lint
go vet ./...

# Build production binary
go build -o truenorth cmd/server/main.go
```

### Environment Variables
```bash
# Required
TRUENORTH_DATABASE_URL="sqlite:./truenorth.db"
TRUENORTH_SECRET_KEY="32-byte-secret-key-here"
TRUENORTH_ENCRYPTION_KEY="32-byte-encryption-key"

# Optional
TRUENORTH_PORT="8080"
TRUENORTH_ENV="development"  # or "production"
TRUENORTH_LOG_LEVEL="info"
```

---

## MVP Feature Checklist

### P0 - Must Have (MVP Launch)
- [ ] User registration and login with MFA
- [ ] CSV upload (Schwab, Fidelity, Vanguard)
- [ ] Auto-tagging of tickers (asset class, sector, geography)
- [ ] Manual editing of classifications
- [ ] Dashboard with allocation charts
- [ ] Top 10 holdings display
- [ ] Free cash identification
- [ ] Concentration alerts
- [ ] Basic what-if scenarios

### P1 - Should Have (Post-MVP)
- [ ] 5-year historical performance
- [ ] Full R2R matrix visualization
- [ ] Investment expense tracking
- [ ] Real-time price updates
- [ ] News feed integration

### P2 - Nice to Have (Future)
- [ ] API sync (Plaid/Yodlee)
- [ ] Efficient Frontier analysis
- [ ] PDF statement OCR
- [ ] Family office tier features

---

## Coding Standards

### Naming Conventions
| Type | Convention | Example |
|------|------------|---------|
| Functions | camelCase, verb-first | `calculateAllocation`, `parseCSV` |
| Types | PascalCase, noun | `Holding`, `Portfolio` |
| Interfaces | PascalCase, -er suffix | `CSVParser`, `TickerTagger` |
| Constants | PascalCase | `MaxConcentration`, `DefaultTimeout` |
| Files | snake_case | `csv_parser.go`, `risk_reward.go` |

### Error Handling
```go
// Always wrap errors with context
if err != nil {
    return fmt.Errorf("failed to parse CSV row %d: %w", rowNum, err)
}

// Use custom error types for business logic
type ValidationError struct {
    Field   string
    Message string
}
```

### Financial Calculations
```go
// ALWAYS use decimal for money - NEVER float64
import "github.com/shopspring/decimal"

// Good
value := decimal.NewFromFloat(1234.56)
total := value.Mul(quantity)

// Bad - DO NOT USE
value := 1234.56  // Float precision issues!
```

---

## Success Metrics

### North Star Metric
**Monthly Active Users (MAU) with refreshed portfolio data**

### KPIs
| Metric | Target (6mo) |
|--------|--------------|
| Time-to-First-Value | < 5 minutes |
| Onboarding Completion | > 70% |
| 30-Day Retention | > 50% |
| NPS Score | > 40 |

---

*Document Version: 1.0 | Last Updated: December 2025*
*When in doubt, favor security and simplicity over features.*
