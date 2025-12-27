// Package storage provides database access and repositories
package storage

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps the database connection
type DB struct {
	*sql.DB
}

// New creates a new database connection
func New(databaseURL string) (*DB, error) {
	db, err := sql.Open("sqlite3", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Enable WAL mode for better concurrency
	if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	return &DB{db}, nil
}

// Migrate runs database migrations
func (db *DB) Migrate() error {
	migrations := []string{
		createUsersTable,
		createPortfoliosTable,
		createHoldingsTable,
		createScenariosTable,
		createSessionsTable,
	}

	for _, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	return nil
}

const createUsersTable = `
CREATE TABLE IF NOT EXISTS users (
	id TEXT PRIMARY KEY,
	email TEXT UNIQUE NOT NULL,
	password_hash TEXT NOT NULL,
	name TEXT NOT NULL,
	mfa_enabled INTEGER DEFAULT 0,
	mfa_secret TEXT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
`

const createPortfoliosTable = `
CREATE TABLE IF NOT EXISTS portfolios (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL,
	name TEXT NOT NULL,
	total_value TEXT DEFAULT '0',
	free_cash TEXT DEFAULT '0',
	last_updated DATETIME DEFAULT CURRENT_TIMESTAMP,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_portfolios_user_id ON portfolios(user_id);
`

const createHoldingsTable = `
CREATE TABLE IF NOT EXISTS holdings (
	id TEXT PRIMARY KEY,
	portfolio_id TEXT NOT NULL,
	account_name TEXT NOT NULL,
	ticker TEXT NOT NULL,
	name TEXT NOT NULL,
	quantity TEXT NOT NULL,
	cost_basis TEXT DEFAULT '0',
	current_price TEXT DEFAULT '0',
	market_value TEXT DEFAULT '0',
	asset_class TEXT DEFAULT 'other',
	sector TEXT,
	geography TEXT,
	is_manual_entry INTEGER DEFAULT 0,
	source TEXT,
	imported_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (portfolio_id) REFERENCES portfolios(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_holdings_portfolio_id ON holdings(portfolio_id);
CREATE INDEX IF NOT EXISTS idx_holdings_ticker ON holdings(ticker);
`

const createScenariosTable = `
CREATE TABLE IF NOT EXISTS scenarios (
	id TEXT PRIMARY KEY,
	portfolio_id TEXT NOT NULL,
	name TEXT NOT NULL,
	allocations TEXT NOT NULL,
	projections TEXT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (portfolio_id) REFERENCES portfolios(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_scenarios_portfolio_id ON scenarios(portfolio_id);
`

const createSessionsTable = `
CREATE TABLE IF NOT EXISTS sessions (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL,
	token TEXT NOT NULL,
	expires_at DATETIME NOT NULL,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(token);
`
