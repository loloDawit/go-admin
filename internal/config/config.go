// Package config loads and validates runtime configuration. It is the only
// place the application reads os.Getenv.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/loloDawit/go-admin/internal/auth"
	"golang.org/x/crypto/bcrypt"
)

const minSecretLen = 32 // SHA-256 output width, which HS256 keys should match

type Config struct {
	DBDSN          string
	SessionSecret  string
	Port           string
	AllowedOrigin  string
	PublicBaseURL  string
	UploadDir      string
	MaxUploadBytes int64
	AppEnv         string
	BcryptCost     int
}

// Gates the cookie Secure flag, which would break login over plain HTTP.
func (c *Config) IsProduction() bool { return c.AppEnv == "production" }

func (c *Config) Hasher() auth.Hasher { return auth.NewHasher(c.BcryptCost) }

// Any error from Load is fatal: the application must not start.
func Load() (*Config, error) {
	cfg := &Config{
		DBDSN:         os.Getenv("DB_DSN"),
		SessionSecret: os.Getenv("SESSION_SECRET"),
		Port:          withDefault("PORT", "8080"),
		AllowedOrigin: os.Getenv("ALLOWED_ORIGIN"),
		PublicBaseURL: withDefault("PUBLIC_BASE_URL", "http://localhost:8080"),
		UploadDir:     withDefault("UPLOAD_DIR", "./uploads"),
		AppEnv:        withDefault("APP_ENV", "development"),
	}

	maxUpload, err := strconv.ParseInt(withDefault("MAX_UPLOAD_BYTES", "5242880"), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("MAX_UPLOAD_BYTES must be an integer: %w", err)
	}
	cfg.MaxUploadBytes = maxUpload

	bcryptCost, err := strconv.Atoi(withDefault("BCRYPT_COST", strconv.Itoa(auth.DefaultCost)))
	if err != nil {
		return nil, fmt.Errorf("BCRYPT_COST must be an integer: %w", err)
	}
	cfg.BcryptCost = bcryptCost

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	var errs []error

	if c.DBDSN == "" {
		errs = append(errs, errors.New("DB_DSN is required (see .env.example)"))
	}
	if len(c.SessionSecret) < minSecretLen {
		errs = append(errs, fmt.Errorf(
			"SESSION_SECRET must be at least %d bytes; generate one with `openssl rand -hex 32`",
			minSecretLen))
	}
	if c.AllowedOrigin == "" {
		errs = append(errs, errors.New(
			"ALLOWED_ORIGIN is required; wildcard CORS with credentials is not permitted"))
	}
	if c.MaxUploadBytes <= 0 {
		errs = append(errs, errors.New("MAX_UPLOAD_BYTES must be positive"))
	}
	if c.BcryptCost < auth.MinAllowedCost || c.BcryptCost > bcrypt.MaxCost {
		errs = append(errs, fmt.Errorf(
			"BCRYPT_COST must be between %d and %d (default %d)",
			auth.MinAllowedCost, bcrypt.MaxCost, auth.DefaultCost))
	}

	return errors.Join(errs...)
}

func withDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
