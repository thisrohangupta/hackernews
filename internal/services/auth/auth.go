// Package auth provides authentication services
package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/findosh/truenorth/internal/config"
	"github.com/findosh/truenorth/internal/models"
	"github.com/findosh/truenorth/internal/storage"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrEmailExists        = errors.New("email already registered")
	ErrSessionExpired     = errors.New("session expired")
	ErrInvalidToken       = errors.New("invalid token")
)

// Service handles authentication operations
type Service struct {
	cfg         *config.Config
	userRepo    *storage.UserRepository
	sessionRepo *storage.SessionRepository
}

// NewService creates a new auth service
func NewService(cfg *config.Config, userRepo *storage.UserRepository, sessionRepo *storage.SessionRepository) *Service {
	return &Service{
		cfg:         cfg,
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
	}
}

// RegisterInput contains registration data
type RegisterInput struct {
	Email    string
	Password string
	Name     string
}

// Register creates a new user account
func (s *Service) Register(input RegisterInput) (*models.User, error) {
	// Check if email exists
	exists, err := s.userRepo.EmailExists(input.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to check email: %w", err)
	}
	if exists {
		return nil, ErrEmailExists
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user := models.NewUser(input.Email, input.Name, string(hash))
	if err := s.userRepo.Create(user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// LoginInput contains login credentials
type LoginInput struct {
	Email    string
	Password string
}

// LoginResult contains the result of a successful login
type LoginResult struct {
	User    *models.User
	Token   string
	Expires time.Time
}

// Login authenticates a user and creates a session
func (s *Service) Login(input LoginInput) (*LoginResult, error) {
	// Find user
	user, err := s.userRepo.GetByEmail(input.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	// Create session token
	token, err := s.createToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to create token: %w", err)
	}

	expires := time.Now().UTC().Add(s.cfg.SessionDuration)

	// Store session
	session := &models.Session{
		ID:        uuid.New(),
		UserID:    user.ID,
		Token:     token,
		ExpiresAt: expires,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.sessionRepo.Create(session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return &LoginResult{
		User:    user,
		Token:   token,
		Expires: expires,
	}, nil
}

// ValidateToken verifies a JWT token and returns the user
func (s *Service) ValidateToken(tokenString string) (*models.User, error) {
	// Parse token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.cfg.SecretKey), nil
	})
	if err != nil {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	// Check expiration
	exp, ok := claims["exp"].(float64)
	if !ok || time.Unix(int64(exp), 0).Before(time.Now()) {
		return nil, ErrSessionExpired
	}

	// Get user ID
	userID, ok := claims["sub"].(string)
	if !ok {
		return nil, ErrInvalidToken
	}

	id, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidToken
	}

	// Load user
	user, err := s.userRepo.GetByID(id)
	if err != nil || user == nil {
		return nil, ErrInvalidToken
	}

	return user, nil
}

// Logout invalidates all sessions for a user
func (s *Service) Logout(userID uuid.UUID) error {
	return s.sessionRepo.DeleteByUserID(userID)
}

// ChangePassword updates a user's password
func (s *Service) ChangePassword(userID uuid.UUID, oldPassword, newPassword string) error {
	user, err := s.userRepo.GetByID(userID)
	if err != nil || user == nil {
		return errors.New("user not found")
	}

	// Verify old password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPassword)); err != nil {
		return ErrInvalidCredentials
	}

	// Hash new password
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	user.PasswordHash = string(hash)
	if err := s.userRepo.Update(user); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Invalidate all sessions
	return s.sessionRepo.DeleteByUserID(userID)
}

func (s *Service) createToken(user *models.User) (string, error) {
	claims := jwt.MapClaims{
		"sub":   user.ID.String(),
		"email": user.Email,
		"name":  user.Name,
		"exp":   time.Now().Add(s.cfg.SessionDuration).Unix(),
		"iat":   time.Now().Unix(),
		"jti":   generateJTI(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.SecretKey))
}

func generateJTI() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// CleanupExpiredSessions removes expired sessions from the database
func (s *Service) CleanupExpiredSessions() error {
	return s.sessionRepo.DeleteExpired()
}
