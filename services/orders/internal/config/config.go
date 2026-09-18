package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// ServiceName is this service's identity in logs, traces and metrics.
const ServiceName = "orders"

// ServiceName and DefaultPort are the two values a service-specific rename touches.
const DefaultPort = "8083"

type Config struct {
	ServiceName         string
	Port                string
	OTLPEndpoint        string
	DatabaseURL         string
	CatalogURL          string
	CatalogTimeout      time.Duration
	CatalogMaxIdleConns int
	OrderPageSizeMax    int
	MaxRequestBodyBytes int64
	PrincipalKey        []byte
	NATSURL             string
	NATSDuplicateWindow time.Duration
	OutboxBatchSize     int
	OutboxPollInterval  time.Duration
	ReportWindowDays    int
}

// Any error here is fatal: a misconfigured service must fail at startup, not at
// the first request.
func Load() (*Config, error) {
	cfg := &Config{
		ServiceName: ServiceName,
		Port:        withDefault("PORT", DefaultPort),
		// Optional by design: an unconfigured collector means a no-op provider,
		// not a failed boot.
		OTLPEndpoint: withDefault("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
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

	maxRequestBodyBytesRaw, err := requireEnv("MAX_REQUEST_BODY_BYTES")
	if err != nil {
		return nil, err
	}
	maxRequestBodyBytes, err := strconv.ParseInt(maxRequestBodyBytesRaw, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("MAX_REQUEST_BODY_BYTES: %w", err)
	}
	cfg.MaxRequestBodyBytes = maxRequestBodyBytes

	principalKey, err := requireEnv("PRINCIPAL_SIGNING_KEY")
	if err != nil {
		return nil, err
	}
	cfg.PrincipalKey = []byte(principalKey)

	natsURL, err := requireEnv("NATS_URL")
	if err != nil {
		return nil, err
	}
	cfg.NATSURL = natsURL

	duplicateWindowRaw, err := requireEnv("NATS_DUPLICATE_WINDOW")
	if err != nil {
		return nil, err
	}
	duplicateWindow, err := time.ParseDuration(duplicateWindowRaw)
	if err != nil {
		return nil, fmt.Errorf("NATS_DUPLICATE_WINDOW: %w", err)
	}
	cfg.NATSDuplicateWindow = duplicateWindow

	outboxBatchSizeRaw, err := requireEnv("OUTBOX_BATCH_SIZE")
	if err != nil {
		return nil, err
	}
	outboxBatchSize, err := strconv.Atoi(outboxBatchSizeRaw)
	if err != nil {
		return nil, fmt.Errorf("OUTBOX_BATCH_SIZE: %w", err)
	}
	cfg.OutboxBatchSize = outboxBatchSize

	pollIntervalRaw, err := requireEnv("OUTBOX_POLL_INTERVAL")
	if err != nil {
		return nil, err
	}
	pollInterval, err := time.ParseDuration(pollIntervalRaw)
	if err != nil {
		return nil, fmt.Errorf("OUTBOX_POLL_INTERVAL: %w", err)
	}
	cfg.OutboxPollInterval = pollInterval

	reportWindowDaysRaw, err := requireEnv("REPORT_WINDOW_DAYS")
	if err != nil {
		return nil, err
	}
	reportWindowDays, err := strconv.Atoi(reportWindowDaysRaw)
	if err != nil {
		return nil, fmt.Errorf("REPORT_WINDOW_DAYS: %w", err)
	}
	cfg.ReportWindowDays = reportWindowDays

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
	if c.MaxRequestBodyBytes <= 0 {
		return fmt.Errorf("MAX_REQUEST_BODY_BYTES must be positive")
	}
	if len(c.PrincipalKey) < 32 {
		return fmt.Errorf("PRINCIPAL_SIGNING_KEY must be at least 32 bytes")
	}
	if c.NATSDuplicateWindow <= 0 {
		return fmt.Errorf("NATS_DUPLICATE_WINDOW must be positive")
	}
	if c.OutboxBatchSize <= 0 {
		return fmt.Errorf("OUTBOX_BATCH_SIZE must be positive")
	}
	if c.OutboxPollInterval <= 0 {
		return fmt.Errorf("OUTBOX_POLL_INTERVAL must be positive")
	}
	if c.ReportWindowDays <= 0 {
		return fmt.Errorf("REPORT_WINDOW_DAYS must be positive")
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
