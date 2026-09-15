package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/loloDawit/go-admin/services/catalog/internal/errs"
)

const serviceName = "catalog"

// serviceName and DefaultPort are the two values a service-specific rename touches.
const DefaultPort = "8082"

type Config struct {
	ServiceName         string
	Port                string
	DatabaseURL         string
	S3Endpoint          string
	S3PublicEndpoint    string
	S3Bucket            string
	S3AccessKey         string
	S3SecretKey         string
	S3UseSSL            bool
	ImageMaxBytes       int64
	ImageURLTTL         time.Duration
	ProductPageSizeMax  int
	ResolveBatchMax     int
	MaxRequestBodyBytes int64
	DefaultCurrency     string
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

	s3Endpoint, err := requireEnv("S3_ENDPOINT")
	if err != nil {
		return nil, err
	}
	cfg.S3Endpoint = s3Endpoint

	s3PublicEndpoint, err := requireEnv("S3_PUBLIC_ENDPOINT")
	if err != nil {
		return nil, err
	}
	cfg.S3PublicEndpoint = s3PublicEndpoint

	s3Bucket, err := requireEnv("S3_BUCKET")
	if err != nil {
		return nil, err
	}
	cfg.S3Bucket = s3Bucket

	s3AccessKey, err := requireEnv("S3_ACCESS_KEY")
	if err != nil {
		return nil, err
	}
	cfg.S3AccessKey = s3AccessKey

	s3SecretKey, err := requireEnv("S3_SECRET_KEY")
	if err != nil {
		return nil, err
	}
	cfg.S3SecretKey = s3SecretKey

	s3UseSSLRaw, err := requireEnv("S3_USE_SSL")
	if err != nil {
		return nil, err
	}
	s3UseSSL, err := strconv.ParseBool(s3UseSSLRaw)
	if err != nil {
		return nil, fmt.Errorf("S3_USE_SSL: %w", err)
	}
	cfg.S3UseSSL = s3UseSSL

	imageMaxBytesRaw, err := requireEnv("IMAGE_MAX_BYTES")
	if err != nil {
		return nil, err
	}
	imageMaxBytes, err := strconv.ParseInt(imageMaxBytesRaw, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("IMAGE_MAX_BYTES: %w", err)
	}
	cfg.ImageMaxBytes = imageMaxBytes

	imageURLTTLRaw, err := requireEnv("IMAGE_URL_TTL")
	if err != nil {
		return nil, err
	}
	imageURLTTL, err := time.ParseDuration(imageURLTTLRaw)
	if err != nil {
		return nil, fmt.Errorf("IMAGE_URL_TTL: %w", err)
	}
	cfg.ImageURLTTL = imageURLTTL

	pageSizeMaxRaw, err := requireEnv("PRODUCT_PAGE_SIZE_MAX")
	if err != nil {
		return nil, err
	}
	pageSizeMax, err := strconv.Atoi(pageSizeMaxRaw)
	if err != nil {
		return nil, fmt.Errorf("PRODUCT_PAGE_SIZE_MAX: %w", err)
	}
	cfg.ProductPageSizeMax = pageSizeMax

	resolveBatchMaxRaw, err := requireEnv("PRODUCT_RESOLVE_BATCH_MAX")
	if err != nil {
		return nil, err
	}
	resolveBatchMax, err := strconv.Atoi(resolveBatchMaxRaw)
	if err != nil {
		return nil, fmt.Errorf("PRODUCT_RESOLVE_BATCH_MAX: %w", err)
	}
	cfg.ResolveBatchMax = resolveBatchMax

	maxRequestBodyBytesRaw, err := requireEnv("MAX_REQUEST_BODY_BYTES")
	if err != nil {
		return nil, err
	}
	maxRequestBodyBytes, err := strconv.ParseInt(maxRequestBodyBytesRaw, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("MAX_REQUEST_BODY_BYTES: %w", err)
	}
	cfg.MaxRequestBodyBytes = maxRequestBodyBytes

	defaultCurrency, err := requireEnv("DEFAULT_CURRENCY")
	if err != nil {
		return nil, err
	}
	cfg.DefaultCurrency = defaultCurrency

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
	if c.ImageMaxBytes <= 0 {
		return errs.ErrInvalidImageMaxBytes
	}
	if c.ProductPageSizeMax <= 0 {
		return errs.ErrInvalidPageSizeMax
	}
	if c.ResolveBatchMax <= 0 {
		return errs.ErrInvalidResolveBatchMax
	}
	if c.MaxRequestBodyBytes <= 0 {
		return errs.ErrInvalidMaxRequestBodyBytes
	}
	if len(c.DefaultCurrency) != 3 {
		return errs.ErrInvalidCurrency
	}
	if len(c.PrincipalKey) < 32 {
		return errs.ErrShortPrincipalKey
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
