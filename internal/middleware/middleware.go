// Package middleware provides HTTP middleware functions
package middleware

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/findosh/truenorth/internal/models"
	"github.com/findosh/truenorth/internal/services/auth"
)

type contextKey string

const (
	UserContextKey contextKey = "user"
)

// Logger logs all HTTP requests
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

// SecurityHeaders adds security headers to all responses
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net; img-src 'self' data:;")
		next.ServeHTTP(w, r)
	})
}

// Recover handles panics gracefully
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("panic recovered: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// Auth middleware for protected routes
type Auth struct {
	authService *auth.Service
}

// NewAuth creates a new auth middleware
func NewAuth(authService *auth.Service) *Auth {
	return &Auth{authService: authService}
}

// RequireAuth ensures the user is authenticated
func (m *Auth) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := m.getUserFromRequest(r)
		if user == nil {
			// Redirect to login for HTML requests
			if strings.Contains(r.Header.Get("Accept"), "text/html") {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}
			// Return 401 for API requests
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Add user to context
		ctx := context.WithValue(r.Context(), UserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// OptionalAuth adds user to context if authenticated, but doesn't require it
func (m *Auth) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := m.getUserFromRequest(r)
		if user != nil {
			ctx := context.WithValue(r.Context(), UserContextKey, user)
			r = r.WithContext(ctx)
		}
		next.ServeHTTP(w, r)
	})
}

func (m *Auth) getUserFromRequest(r *http.Request) *models.User {
	// Try cookie first
	cookie, err := r.Cookie("session")
	if err == nil && cookie.Value != "" {
		user, err := m.authService.ValidateToken(cookie.Value)
		if err == nil {
			return user
		}
	}

	// Try Authorization header
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		token := strings.TrimPrefix(authHeader, "Bearer ")
		user, err := m.authService.ValidateToken(token)
		if err == nil {
			return user
		}
	}

	return nil
}

// GetUser retrieves the user from the request context
func GetUser(r *http.Request) *models.User {
	user, ok := r.Context().Value(UserContextKey).(*models.User)
	if !ok {
		return nil
	}
	return user
}

// Chain applies middleware in order
func Chain(h http.Handler, middleware ...func(http.Handler) http.Handler) http.Handler {
	for i := len(middleware) - 1; i >= 0; i-- {
		h = middleware[i](h)
	}
	return h
}
