package config

import (
	"errors"
	"fmt"
	"os"
	"time"
)

type Config struct {
	Port            string
	Upstreams       map[string]string
	UpstreamTimeout time.Duration

	// PrincipalKey must be byte-identical to identity's PRINCIPAL_SIGNING_KEY.
	PrincipalKey []byte
	PrincipalTTL time.Duration
	// SessionCacheTTL is the revocation-latency budget: a revoked session can
	// still be honored for up to this long.
	SessionCacheTTL time.Duration
}

// Any error here is fatal: a misconfigured gateway must fail at startup, not at
// the first request.
func Load() (*Config, error) {
	cfg := &Config{
		Port:            withDefault("PORT", "8080"),
		UpstreamTimeout: 5 * time.Second,
		Upstreams: map[string]string{
			"identity": os.Getenv("IDENTITY_URL"),
			"catalog":  os.Getenv("CATALOG_URL"),
			"orders":   os.Getenv("ORDERS_URL"),
		},
		PrincipalKey: []byte(os.Getenv("PRINCIPAL_SIGNING_KEY")),
	}
	for name, url := range cfg.Upstreams {
		if url == "" {
			return nil, errors.New("upstream URL is required for " + name)
		}
	}

	ttlRaw, err := requireEnv("PRINCIPAL_TTL")
	if err != nil {
		return nil, err
	}
	principalTTL, err := time.ParseDuration(ttlRaw)
	if err != nil {
		return nil, fmt.Errorf("PRINCIPAL_TTL: %w", err)
	}
	cfg.PrincipalTTL = principalTTL

	cacheTTLRaw, err := requireEnv("SESSION_CACHE_TTL")
	if err != nil {
		return nil, err
	}
	cacheTTL, err := time.ParseDuration(cacheTTLRaw)
	if err != nil {
		return nil, fmt.Errorf("SESSION_CACHE_TTL: %w", err)
	}
	cfg.SessionCacheTTL = cacheTTL

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	if len(c.PrincipalKey) < 32 {
		return errors.New("PRINCIPAL_SIGNING_KEY must be at least 32 bytes")
	}
	if c.PrincipalTTL <= 0 {
		return errors.New("PRINCIPAL_TTL must be positive")
	}
	if c.SessionCacheTTL <= 0 {
		return errors.New("SESSION_CACHE_TTL must be positive")
	}
	return nil
}

// requireEnv names the missing variable, rather than letting an empty string
// reach time.ParseDuration and read as a parse failure.
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
