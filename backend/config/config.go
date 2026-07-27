package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Port           string
	DatabaseURL    string
	JWTSecret      string
	CardCodePepper string
	AllowedOrigins []string
	Environment    string
}

func Load() *Config {
	cfg := &Config{
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/aiswitch?sslmode=disable"),
		JWTSecret:   getEnv("JWT_SECRET", "aiswitch-dev-secret-change-in-production"),
		Environment: strings.ToLower(getEnv("APP_ENV", "development")),
	}
	cfg.CardCodePepper = getEnv("CARD_CODE_PEPPER", cfg.JWTSecret)
	cfg.AllowedOrigins = splitCSV(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:1420,tauri://localhost,https://tauri.localhost"))
	return cfg
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if item := strings.TrimSpace(part); item != "" {
			result = append(result, item)
		}
	}
	return result
}

// Validate rejects insecure defaults before a production server accepts users.
func (c *Config) Validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if c.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}
	if c.Environment == "production" {
		if c.JWTSecret == "aiswitch-dev-secret-change-in-production" || len(c.JWTSecret) < 32 {
			return fmt.Errorf("production JWT_SECRET must be a unique value of at least 32 characters")
		}
		if c.CardCodePepper == c.JWTSecret || len(c.CardCodePepper) < 32 {
			return fmt.Errorf("production CARD_CODE_PEPPER must be a distinct value of at least 32 characters")
		}
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
