package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/findosh/truenorth/internal/models"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// PortfolioRepository provides portfolio data access
type PortfolioRepository struct {
	db *DB
}

// NewPortfolioRepository creates a new portfolio repository
func NewPortfolioRepository(db *DB) *PortfolioRepository {
	return &PortfolioRepository{db: db}
}

// Create inserts a new portfolio
func (r *PortfolioRepository) Create(p *models.Portfolio) error {
	query := `
		INSERT INTO portfolios (id, user_id, name, total_value, free_cash, last_updated, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.Exec(query,
		p.ID.String(),
		p.UserID.String(),
		p.Name,
		p.TotalValue.String(),
		p.FreeCash.String(),
		p.LastUpdated,
		p.CreatedAt,
	)
	return err
}

// GetByID retrieves a portfolio by ID with holdings
func (r *PortfolioRepository) GetByID(id uuid.UUID) (*models.Portfolio, error) {
	query := `
		SELECT id, user_id, name, total_value, free_cash, last_updated, created_at
		FROM portfolios WHERE id = ?
	`
	p, err := r.scanPortfolio(r.db.QueryRow(query, id.String()))
	if err != nil || p == nil {
		return p, err
	}

	// Load holdings
	holdings, err := r.getHoldings(id)
	if err != nil {
		return nil, err
	}
	p.Holdings = holdings

	return p, nil
}

// GetByUserID retrieves all portfolios for a user
func (r *PortfolioRepository) GetByUserID(userID uuid.UUID) ([]*models.Portfolio, error) {
	query := `
		SELECT id, user_id, name, total_value, free_cash, last_updated, created_at
		FROM portfolios WHERE user_id = ? ORDER BY created_at DESC
	`
	rows, err := r.db.Query(query, userID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var portfolios []*models.Portfolio
	for rows.Next() {
		p, err := r.scanPortfolioRow(rows)
		if err != nil {
			return nil, err
		}
		portfolios = append(portfolios, p)
	}

	return portfolios, rows.Err()
}

// Update modifies an existing portfolio
func (r *PortfolioRepository) Update(p *models.Portfolio) error {
	p.LastUpdated = time.Now().UTC()
	query := `
		UPDATE portfolios SET name = ?, total_value = ?, free_cash = ?, last_updated = ?
		WHERE id = ?
	`
	_, err := r.db.Exec(query,
		p.Name,
		p.TotalValue.String(),
		p.FreeCash.String(),
		p.LastUpdated,
		p.ID.String(),
	)
	return err
}

// Delete removes a portfolio and all its holdings
func (r *PortfolioRepository) Delete(id uuid.UUID) error {
	_, err := r.db.Exec("DELETE FROM portfolios WHERE id = ?", id.String())
	return err
}

func (r *PortfolioRepository) scanPortfolio(row *sql.Row) (*models.Portfolio, error) {
	var p models.Portfolio
	var id, userID, totalValue, freeCash string

	err := row.Scan(&id, &userID, &p.Name, &totalValue, &freeCash, &p.LastUpdated, &p.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan portfolio: %w", err)
	}

	p.ID, _ = uuid.Parse(id)
	p.UserID, _ = uuid.Parse(userID)
	p.TotalValue, _ = decimal.NewFromString(totalValue)
	p.FreeCash, _ = decimal.NewFromString(freeCash)

	return &p, nil
}

func (r *PortfolioRepository) scanPortfolioRow(rows *sql.Rows) (*models.Portfolio, error) {
	var p models.Portfolio
	var id, userID, totalValue, freeCash string

	err := rows.Scan(&id, &userID, &p.Name, &totalValue, &freeCash, &p.LastUpdated, &p.CreatedAt)
	if err != nil {
		return nil, err
	}

	p.ID, _ = uuid.Parse(id)
	p.UserID, _ = uuid.Parse(userID)
	p.TotalValue, _ = decimal.NewFromString(totalValue)
	p.FreeCash, _ = decimal.NewFromString(freeCash)

	return &p, nil
}

// HoldingRepository provides holding data access
type HoldingRepository struct {
	db *DB
}

// NewHoldingRepository creates a new holding repository
func NewHoldingRepository(db *DB) *HoldingRepository {
	return &HoldingRepository{db: db}
}

// Create inserts a new holding
func (r *HoldingRepository) Create(h *models.Holding) error {
	query := `
		INSERT INTO holdings (
			id, portfolio_id, account_name, ticker, name, quantity,
			cost_basis, current_price, market_value, asset_class,
			sector, geography, is_manual_entry, source, imported_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.Exec(query,
		h.ID.String(),
		h.PortfolioID.String(),
		h.AccountName,
		h.Ticker,
		h.Name,
		h.Quantity.String(),
		h.CostBasis.String(),
		h.CurrentPrice.String(),
		h.MarketValue.String(),
		string(h.AssetClass),
		h.Sector,
		h.Geography,
		h.IsManualEntry,
		h.Source,
		h.ImportedAt,
	)
	return err
}

// CreateBatch inserts multiple holdings in a transaction
func (r *HoldingRepository) CreateBatch(holdings []models.Holding) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO holdings (
			id, portfolio_id, account_name, ticker, name, quantity,
			cost_basis, current_price, market_value, asset_class,
			sector, geography, is_manual_entry, source, imported_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, h := range holdings {
		_, err := stmt.Exec(
			h.ID.String(),
			h.PortfolioID.String(),
			h.AccountName,
			h.Ticker,
			h.Name,
			h.Quantity.String(),
			h.CostBasis.String(),
			h.CurrentPrice.String(),
			h.MarketValue.String(),
			string(h.AssetClass),
			h.Sector,
			h.Geography,
			h.IsManualEntry,
			h.Source,
			h.ImportedAt,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// Update modifies an existing holding
func (r *HoldingRepository) Update(h *models.Holding) error {
	query := `
		UPDATE holdings SET
			account_name = ?, ticker = ?, name = ?, quantity = ?,
			cost_basis = ?, current_price = ?, market_value = ?,
			asset_class = ?, sector = ?, geography = ?, is_manual_entry = ?
		WHERE id = ?
	`
	_, err := r.db.Exec(query,
		h.AccountName,
		h.Ticker,
		h.Name,
		h.Quantity.String(),
		h.CostBasis.String(),
		h.CurrentPrice.String(),
		h.MarketValue.String(),
		string(h.AssetClass),
		h.Sector,
		h.Geography,
		h.IsManualEntry,
		h.ID.String(),
	)
	return err
}

// Delete removes a holding
func (r *HoldingRepository) Delete(id uuid.UUID) error {
	_, err := r.db.Exec("DELETE FROM holdings WHERE id = ?", id.String())
	return err
}

// DeleteByPortfolioID removes all holdings for a portfolio
func (r *HoldingRepository) DeleteByPortfolioID(portfolioID uuid.UUID) error {
	_, err := r.db.Exec("DELETE FROM holdings WHERE portfolio_id = ?", portfolioID.String())
	return err
}

// getHoldings retrieves all holdings for a portfolio
func (r *PortfolioRepository) getHoldings(portfolioID uuid.UUID) ([]models.Holding, error) {
	query := `
		SELECT id, portfolio_id, account_name, ticker, name, quantity,
			cost_basis, current_price, market_value, asset_class,
			sector, geography, is_manual_entry, source, imported_at
		FROM holdings WHERE portfolio_id = ? ORDER BY market_value DESC
	`
	rows, err := r.db.Query(query, portfolioID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var holdings []models.Holding
	for rows.Next() {
		h, err := scanHoldingRow(rows)
		if err != nil {
			return nil, err
		}
		holdings = append(holdings, *h)
	}

	return holdings, rows.Err()
}

func scanHoldingRow(rows *sql.Rows) (*models.Holding, error) {
	var h models.Holding
	var id, portfolioID string
	var quantity, costBasis, currentPrice, marketValue string
	var assetClass string
	var sector, geography, source sql.NullString

	err := rows.Scan(
		&id, &portfolioID, &h.AccountName, &h.Ticker, &h.Name,
		&quantity, &costBasis, &currentPrice, &marketValue,
		&assetClass, &sector, &geography, &h.IsManualEntry, &source, &h.ImportedAt,
	)
	if err != nil {
		return nil, err
	}

	h.ID, _ = uuid.Parse(id)
	h.PortfolioID, _ = uuid.Parse(portfolioID)
	h.Quantity, _ = decimal.NewFromString(quantity)
	h.CostBasis, _ = decimal.NewFromString(costBasis)
	h.CurrentPrice, _ = decimal.NewFromString(currentPrice)
	h.MarketValue, _ = decimal.NewFromString(marketValue)
	h.AssetClass = models.AssetClass(assetClass)

	if sector.Valid {
		h.Sector = sector.String
	}
	if geography.Valid {
		h.Geography = geography.String
	}
	if source.Valid {
		h.Source = source.String
	}

	return &h, nil
}

// ScenarioRepository provides scenario data access
type ScenarioRepository struct {
	db *DB
}

// NewScenarioRepository creates a new scenario repository
func NewScenarioRepository(db *DB) *ScenarioRepository {
	return &ScenarioRepository{db: db}
}

// Create inserts a new scenario
func (r *ScenarioRepository) Create(s *models.Scenario) error {
	allocJSON, _ := json.Marshal(s.Allocations)
	projJSON, _ := json.Marshal(s.Projections)

	query := `
		INSERT INTO scenarios (id, portfolio_id, name, allocations, projections, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.Exec(query,
		s.ID.String(),
		s.PortfolioID.String(),
		s.Name,
		string(allocJSON),
		string(projJSON),
		s.CreatedAt,
	)
	return err
}

// GetByPortfolioID retrieves all scenarios for a portfolio
func (r *ScenarioRepository) GetByPortfolioID(portfolioID uuid.UUID) ([]*models.Scenario, error) {
	query := `
		SELECT id, portfolio_id, name, allocations, projections, created_at
		FROM scenarios WHERE portfolio_id = ? ORDER BY created_at DESC
	`
	rows, err := r.db.Query(query, portfolioID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var scenarios []*models.Scenario
	for rows.Next() {
		s, err := r.scanScenarioRow(rows)
		if err != nil {
			return nil, err
		}
		scenarios = append(scenarios, s)
	}

	return scenarios, rows.Err()
}

// Delete removes a scenario
func (r *ScenarioRepository) Delete(id uuid.UUID) error {
	_, err := r.db.Exec("DELETE FROM scenarios WHERE id = ?", id.String())
	return err
}

func (r *ScenarioRepository) scanScenarioRow(rows *sql.Rows) (*models.Scenario, error) {
	var s models.Scenario
	var id, portfolioID, allocJSON, projJSON string

	err := rows.Scan(&id, &portfolioID, &s.Name, &allocJSON, &projJSON, &s.CreatedAt)
	if err != nil {
		return nil, err
	}

	s.ID, _ = uuid.Parse(id)
	s.PortfolioID, _ = uuid.Parse(portfolioID)

	if allocJSON != "" {
		json.Unmarshal([]byte(allocJSON), &s.Allocations)
	}
	if projJSON != "" {
		json.Unmarshal([]byte(projJSON), &s.Projections)
	}

	return &s, nil
}
