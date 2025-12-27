package storage

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/findosh/truenorth/internal/models"
	"github.com/google/uuid"
)

// UserRepository provides user data access
type UserRepository struct {
	db *DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create inserts a new user
func (r *UserRepository) Create(user *models.User) error {
	query := `
		INSERT INTO users (id, email, password_hash, name, mfa_enabled, mfa_secret, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.Exec(query,
		user.ID.String(),
		user.Email,
		user.PasswordHash,
		user.Name,
		user.MFAEnabled,
		user.MFASecret,
		user.CreatedAt,
		user.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(id uuid.UUID) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, name, mfa_enabled, mfa_secret, created_at, updated_at
		FROM users WHERE id = ?
	`
	return r.scanUser(r.db.QueryRow(query, id.String()))
}

// GetByEmail retrieves a user by email
func (r *UserRepository) GetByEmail(email string) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, name, mfa_enabled, mfa_secret, created_at, updated_at
		FROM users WHERE email = ?
	`
	return r.scanUser(r.db.QueryRow(query, email))
}

// Update modifies an existing user
func (r *UserRepository) Update(user *models.User) error {
	user.UpdatedAt = time.Now().UTC()
	query := `
		UPDATE users SET email = ?, name = ?, mfa_enabled = ?, mfa_secret = ?, updated_at = ?
		WHERE id = ?
	`
	_, err := r.db.Exec(query,
		user.Email,
		user.Name,
		user.MFAEnabled,
		user.MFASecret,
		user.UpdatedAt,
		user.ID.String(),
	)
	return err
}

// Delete removes a user
func (r *UserRepository) Delete(id uuid.UUID) error {
	_, err := r.db.Exec("DELETE FROM users WHERE id = ?", id.String())
	return err
}

// EmailExists checks if an email is already registered
func (r *UserRepository) EmailExists(email string) (bool, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM users WHERE email = ?", email).Scan(&count)
	return count > 0, err
}

func (r *UserRepository) scanUser(row *sql.Row) (*models.User, error) {
	var user models.User
	var id string
	var mfaSecret sql.NullString

	err := row.Scan(
		&id,
		&user.Email,
		&user.PasswordHash,
		&user.Name,
		&user.MFAEnabled,
		&mfaSecret,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan user: %w", err)
	}

	user.ID, _ = uuid.Parse(id)
	if mfaSecret.Valid {
		user.MFASecret = mfaSecret.String
	}

	return &user, nil
}

// SessionRepository provides session data access
type SessionRepository struct {
	db *DB
}

// NewSessionRepository creates a new session repository
func NewSessionRepository(db *DB) *SessionRepository {
	return &SessionRepository{db: db}
}

// Create inserts a new session
func (r *SessionRepository) Create(session *models.Session) error {
	query := `
		INSERT INTO sessions (id, user_id, token, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?)
	`
	_, err := r.db.Exec(query,
		session.ID.String(),
		session.UserID.String(),
		session.Token,
		session.ExpiresAt,
		session.CreatedAt,
	)
	return err
}

// GetByToken retrieves a session by token
func (r *SessionRepository) GetByToken(token string) (*models.Session, error) {
	query := `
		SELECT id, user_id, token, expires_at, created_at
		FROM sessions WHERE token = ?
	`
	var session models.Session
	var id, userID string

	err := r.db.QueryRow(query, token).Scan(
		&id,
		&userID,
		&session.Token,
		&session.ExpiresAt,
		&session.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	session.ID, _ = uuid.Parse(id)
	session.UserID, _ = uuid.Parse(userID)

	return &session, nil
}

// DeleteByUserID removes all sessions for a user
func (r *SessionRepository) DeleteByUserID(userID uuid.UUID) error {
	_, err := r.db.Exec("DELETE FROM sessions WHERE user_id = ?", userID.String())
	return err
}

// DeleteExpired removes all expired sessions
func (r *SessionRepository) DeleteExpired() error {
	_, err := r.db.Exec("DELETE FROM sessions WHERE expires_at < ?", time.Now().UTC())
	return err
}
