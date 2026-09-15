package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

const serviceName = "orders"

// serviceName and DefaultPort are the two values a service-specific rename touches.
const DefaultPort = "8083"

type Config struct {
	ServiceName         string
	Port                string
	DatabaseURL         string
	CatalogURL          string
	CatalogTimeout      time.Duration
	CatalogMaxIdleConns int
	OrderPageSizeMax    int
	PrincipalKey        []byte
}

// Any error here is fatal: a misconfigured service must fail at startup, not at
// the first request.
func Load() (*Config, error) {
	cfg := &Config{
		ServiceName: serviceName,
		Port:        withDefault("PORT", DefaultPort),
	}

	databaseURL, err := requireEnv("DATABASE_URL")
	if err != nil {
		return nil, err
	}
	cfg.DatabaseURL = databaseURL

	catalogURL, err := requireEnv("CATALOG_URL")
	if err != nil {
		return nil, err
	}
	cfg.CatalogURL = catalogURL

	catalogTimeoutRaw, err := requireEnv("CATALOG_TIMEOUT")
	if err != nil {
		return nil, err
	}
	catalogTimeout, err := time.ParseDuration(catalogTimeoutRaw)
	if err != nil {
		return nil, fmt.Errorf("CATALOG_TIMEOUT: %w", err)
	}
	cfg.CatalogTimeout = catalogTimeout

	catalogMaxIdleConnsRaw, err := requireEnv("CATALOG_MAX_IDLE_CONNS")
	if err != nil {
		return nil, err
	}
	catalogMaxIdleConns, err := strconv.Atoi(catalogMaxIdleConnsRaw)
	if err != nil {
		return nil, fmt.Errorf("CATALOG_MAX_IDLE_CONNS: %w", err)
	}
	cfg.CatalogMaxIdleConns = catalogMaxIdleConns

	orderPageSizeMaxRaw, err := requireEnv("ORDER_PAGE_SIZE_MAX")
	if err != nil {
		return nil, err
	}
	orderPageSizeMax, err := strconv.Atoi(orderPageSizeMaxRaw)
	if err != nil {
		return nil, fmt.Errorf("ORDER_PAGE_SIZE_MAX: %w", err)
	}
	cfg.OrderPageSizeMax = orderPageSizeMax

	principalKey, err := requireEnv("PRINCIPAL_SIGNING_KEY")
	if err != nil {
		return nil, err
	}
	cfg.PrincipalKey = []byte(principalKey)

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	if c.CatalogTimeout <= 0 {
		return fmt.Errorf("CATALOG_TIMEOUT must be positive")
	}
	if c.CatalogMaxIdleConns <= 0 {
		return fmt.Errorf("CATALOG_MAX_IDLE_CONNS must be positive")
	}
	if c.OrderPageSizeMax <= 0 {
		return fmt.Errorf("ORDER_PAGE_SIZE_MAX must be positive")
	}
	if len(c.PrincipalKey) < 32 {
		return fmt.Errorf("PRINCIPAL_SIGNING_KEY must be at least 32 bytes")
	}
	return nil
}

// requireEnv names the missing variable, rather than letting an empty string reach a downstream parser and read as a parse failure.
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
