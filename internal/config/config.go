// Package config loads and validates all runtime configuration from the
// environment. Load is the only place the application reads os.Getenv;
// everything else takes a *Config.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/loloDawit/go-admin/internal/auth"
	"golang.org/x/crypto/bcrypt"
)

// minSecretLen is the shortest session signing key we accept. 32 bytes is the
// output width of SHA-256, which is what HS256 keys should match.
const minSecretLen = 32

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

// IsProduction reports whether the app is running in production. It gates the
// Secure flag on the session cookie: Secure cookies are not sent over plain
// HTTP, so forcing it on in local development would break login on :3000.
func (c *Config) IsProduction() bool { return c.AppEnv == "production" }

// Hasher returns a password hasher at the configured cost.
func (c *Config) Hasher() auth.Hasher { return auth.NewHasher(c.BcryptCost) }

// Load reads configuration from the environment and validates it. It returns
// an error rather than panicking so that main can decide how to report the
// failure. Any error from Load is fatal: the application must not start.
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
	// Too low is a security problem; too high is a login-latency DoS.
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
