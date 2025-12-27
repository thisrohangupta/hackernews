# FinDosh TrueNorth

**Unified Portfolio Intelligence for Sophisticated DIY Investors**

TrueNorth is a portfolio management platform designed for investors with $5-10MM+ portfolios across multiple brokerage accounts. It provides:

- **Unified View**: See all accounts in one place
- **Smart Classification**: AI-powered ticker tagging by asset class, sector, and geography
- **Risk Alerts**: Concentration, overlap, and sector tilt detection
- **What-If Scenarios**: Model allocation changes before trading

## Quick Start

```bash
# Run the server
go run cmd/server/main.go

# Open in browser
open http://localhost:8080
```

## Features

### CSV Import
Upload portfolio exports from:
- Charles Schwab
- Fidelity
- Vanguard
- Generic CSV format

### Dashboard
- Total portfolio value
- Asset allocation charts (by class, sector, geography)
- Top 10 holdings
- Concentration alerts

### Scenario Modeling
- Adjust target allocations with sliders
- See projected best/worst/average returns
- Save scenarios for comparison

## Tech Stack

- **Backend**: Go 1.21+
- **Database**: SQLite (MVP)
- **Frontend**: Server-rendered HTML + Chart.js
- **Auth**: JWT with secure cookies

## Project Structure

```
truenorth/
├── cmd/server/         # Application entry point
├── internal/
│   ├── config/         # Configuration
│   ├── models/         # Domain models
│   ├── services/       # Business logic
│   ├── handlers/       # HTTP handlers
│   ├── middleware/     # HTTP middleware
│   └── storage/        # Database access
├── web/
│   ├── templates/      # HTML templates
│   └── static/         # CSS, JS, images
└── testdata/           # Sample CSV files
```

## Environment Variables

```bash
TRUENORTH_PORT=8080
TRUENORTH_ENV=development
TRUENORTH_DATABASE_URL=truenorth.db
TRUENORTH_SECRET_KEY=your-secret-key
```

## Security

- Passwords hashed with bcrypt
- JWT tokens for sessions
- Security headers on all responses
- Read-only data model (no trade execution)

## License

Proprietary - Delta Capital & Research
