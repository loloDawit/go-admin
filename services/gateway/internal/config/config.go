package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/loloDawit/go-admin/services/gateway/internal/faults"
	"github.com/loloDawit/go-admin/services/gateway/internal/ratelimit"
)

type Config struct {
	Port            string
	OTLPEndpoint    string
	Upstreams       map[string]string
	UpstreamTimeout time.Duration

	// PrincipalKey must be byte-identical to identity's PRINCIPAL_SIGNING_KEY.
	PrincipalKey []byte
	PrincipalTTL time.Duration
	// SessionCacheTTL is the revocation-latency budget: a revoked session can
	// still be honored for up to this long.
	SessionCacheTTL time.Duration

	// RateLimit bounds one member of staff, not one IP address: a shop behind
	// one NAT address shares an IP.
	RateLimit ratelimit.Config

	// Profile gates fault injection. Anything but "dev" refuses it at startup.
	Profile string
	Faults  faults.Config

	// WebRoot is the built frontend's directory. Empty serves no frontend,
	// which is how the API-only tests and a dev-server frontend both run.
	WebRoot string
}

// Any error here is fatal: a misconfigured gateway must fail at startup, not at
// the first request.
func Load() (*Config, error) {
	cfg := &Config{
		Port: withDefault("PORT", "8080"),
		// Optional by design: an unconfigured collector means a no-op provider,
		// not a failed boot.
		OTLPEndpoint:    withDefault("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
		UpstreamTimeout: 5 * time.Second,
		Upstreams: map[string]string{
			"identity": os.Getenv("IDENTITY_URL"),
			"catalog":  os.Getenv("CATALOG_URL"),
			"orders":   os.Getenv("ORDERS_URL"),
		},
		PrincipalKey: []byte(os.Getenv("PRINCIPAL_SIGNING_KEY")),
		WebRoot:      os.Getenv("WEB_ROOT"),
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

	cfg.Profile = withDefault("APP_PROFILE", "production")
	faultCfg, err := loadFaults()
	if err != nil {
		return nil, err
	}
	cfg.Faults = faultCfg

	limits, err := loadRateLimits()
	if err != nil {
		return nil, err
	}
	cfg.RateLimit = limits

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Both knobs default to off, so an operator who sets neither gets no injector
// and no way to be surprised by one.
func loadFaults() (faults.Config, error) {
	latency, err := optionalInt("FAULT_LATENCY_MS")
	if err != nil {
		return faults.Config{}, err
	}
	errorRate, err := optionalFloat("FAULT_ERROR_RATE")
	if err != nil {
		return faults.Config{}, err
	}
	return faults.Config{LatencyMS: latency, ErrorRate: errorRate}, nil
}

func optionalInt(key string) (int, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return 0, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return v, nil
}

func optionalFloat(key string) (float64, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return 0, nil
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return v, nil
}

func loadRateLimits() (ratelimit.Config, error) {
	perSecond, err := requireFloat("RATE_LIMIT_PER_SECOND")
	if err != nil {
		return ratelimit.Config{}, err
	}
	burst, err := requireInt("RATE_LIMIT_BURST")
	if err != nil {
		return ratelimit.Config{}, err
	}
	loginPerSecond, err := requireFloat("LOGIN_RATE_LIMIT_PER_SECOND")
	if err != nil {
		return ratelimit.Config{}, err
	}
	loginBurst, err := requireInt("LOGIN_RATE_LIMIT_BURST")
	if err != nil {
		return ratelimit.Config{}, err
	}
	return ratelimit.Config{
		Rate: perSecond, Burst: burst,
		LoginRate: loginPerSecond, LoginBurst: loginBurst,
	}, nil
}

func requireFloat(key string) (float64, error) {
	raw, err := requireEnv(key)
	if err != nil {
		return 0, err
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return v, nil
}

func requireInt(key string) (int, error) {
	raw, err := requireEnv(key)
	if err != nil {
		return 0, err
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return v, nil
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
	if c.RateLimit.Rate <= 0 || c.RateLimit.Burst <= 0 {
		return errors.New("RATE_LIMIT_PER_SECOND and RATE_LIMIT_BURST must be positive")
	}
	if c.RateLimit.LoginRate <= 0 || c.RateLimit.LoginBurst <= 0 {
		return errors.New("LOGIN_RATE_LIMIT_PER_SECOND and LOGIN_RATE_LIMIT_BURST must be positive")
	}
	if err := faults.Validate(c.Faults, c.Profile); err != nil {
		return err
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
