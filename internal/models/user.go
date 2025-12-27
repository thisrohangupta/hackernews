// Package models defines core domain types
package models

import (
	"time"

	"github.com/google/uuid"
)

// User represents an authenticated investor
type User struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // Never serialize to JSON
	Name         string    `json:"name"`
	MFAEnabled   bool      `json:"mfa_enabled"`
	MFASecret    string    `json:"-"` // Never serialize
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// NewUser creates a new user with generated ID and timestamps
func NewUser(email, name, passwordHash string) *User {
	now := time.Now().UTC()
	return &User{
		ID:           uuid.New(),
		Email:        email,
		Name:         name,
		PasswordHash: passwordHash,
		MFAEnabled:   false,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// Session represents an active user session
type Session struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Token     string    `json:"-"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// IsExpired checks if the session has expired
func (s *Session) IsExpired() bool {
	return time.Now().UTC().After(s.ExpiresAt)
}
