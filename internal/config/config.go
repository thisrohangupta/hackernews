// Package config manages application configuration
package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration
type Config struct {
	// Server settings
	Port        string
	Environment string // "development" or "production"

	// Database
	DatabaseURL string

	// Security
	SecretKey     string // For JWT signing
	EncryptionKey string // For AES encryption

	// Session settings
	SessionDuration time.Duration

	// Feature flags
	EnableMFA bool
}

// Load reads configuration from environment variables with sensible defaults
func Load() *Config {
	return &Config{
		Port:            getEnv("TRUENORTH_PORT", "8080"),
		Environment:     getEnv("TRUENORTH_ENV", "development"),
		DatabaseURL:     getEnv("TRUENORTH_DATABASE_URL", "truenorth.db"),
		SecretKey:       getEnv("TRUENORTH_SECRET_KEY", "dev-secret-key-change-in-production"),
		EncryptionKey:   getEnv("TRUENORTH_ENCRYPTION_KEY", "dev-encryption-key-32bytes!"),
		SessionDuration: getDurationEnv("TRUENORTH_SESSION_DURATION", 24*time.Hour),
		EnableMFA:       getBoolEnv("TRUENORTH_ENABLE_MFA", false),
	}
}

// IsDevelopment returns true if running in development mode
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

// IsProduction returns true if running in production mode
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}
