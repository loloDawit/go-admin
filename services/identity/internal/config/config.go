package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const serviceName = "identity"

// DefaultPort is copied by every service derived from this template.
// serviceName above and DefaultPort are the two values in this file a
// service-specific rename touches.
const DefaultPort = "8081"

type Config struct {
	ServiceName         string
	Port                string
	DatabaseURL         string
	BcryptCost          int
	SessionTTL          time.Duration
	CookieSecure        bool
	PrincipalKey        []byte
	MaxRequestBodyBytes int64
}

// Any error here is fatal: a misconfigured service must fail at startup, not at
// the first request.
func Load() (*Config, error) {
	cfg := &Config{
		ServiceName: serviceName,
		Port:        withDefault("PORT", DefaultPort),
		DatabaseURL: os.Getenv("DATABASE_URL"),
	}
	if cfg.DatabaseURL == "" {
		return nil, errors.New("DATABASE_URL is required")
	}

	costRaw, err := requireEnv("BCRYPT_COST")
	if err != nil {
		return nil, err
	}
	cost, err := strconv.Atoi(costRaw)
	if err != nil {
		return nil, fmt.Errorf("BCRYPT_COST: %w", err)
	}
	cfg.BcryptCost = cost

	ttlRaw, err := requireEnv("SESSION_TTL")
	if err != nil {
		return nil, err
	}
	ttl, err := time.ParseDuration(ttlRaw)
	if err != nil {
		return nil, fmt.Errorf("SESSION_TTL: %w", err)
	}
	cfg.SessionTTL = ttl

	secureRaw, err := requireEnv("COOKIE_SECURE")
	if err != nil {
		return nil, err
	}
	secure, err := strconv.ParseBool(secureRaw)
	if err != nil {
		return nil, fmt.Errorf("COOKIE_SECURE: %w", err)
	}
	cfg.CookieSecure = secure

	cfg.PrincipalKey = []byte(os.Getenv("PRINCIPAL_SIGNING_KEY"))

	maxBodyRaw, err := requireEnv("MAX_REQUEST_BODY_BYTES")
	if err != nil {
		return nil, err
	}
	maxBody, err := strconv.ParseInt(maxBodyRaw, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("MAX_REQUEST_BODY_BYTES: %w", err)
	}
	cfg.MaxRequestBodyBytes = maxBody

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	if c.BcryptCost < bcrypt.MinCost || c.BcryptCost > bcrypt.MaxCost {
		return fmt.Errorf("BCRYPT_COST must be %d-%d, got %d", bcrypt.MinCost, bcrypt.MaxCost, c.BcryptCost)
	}
	if len(c.PrincipalKey) < 32 {
		return errors.New("PRINCIPAL_SIGNING_KEY must be at least 32 bytes")
	}
	if c.SessionTTL <= 0 {
		return errors.New("SESSION_TTL must be positive")
	}
	if c.MaxRequestBodyBytes <= 0 {
		return errors.New("MAX_REQUEST_BODY_BYTES must be positive")
	}
	return nil
}

// requireEnv names the missing variable in its error rather than letting a
// downstream parser (strconv, time.ParseDuration) report on an empty string,
// which reads as a parse failure and hides that the variable is simply unset.
func requireEnv(key string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return "", fmt.Errorf("%s is required", key)
	}
	return v, nil
}

func withDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
