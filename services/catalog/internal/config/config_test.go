package config_test

import (
	"strings"
	"testing"

	"github.com/loloDawit/go-admin/services/catalog/internal/config"
)

const validSigningKey = "dev_only_principal_key_at_least_32_bytes"

func setValidCatalogEnv(t *testing.T) {
	t.Helper()
	t.Setenv("S3_ENDPOINT", "minio:9000")
	t.Setenv("S3_PUBLIC_ENDPOINT", "localhost:9000")
	t.Setenv("S3_BUCKET", "catalog-images")
	t.Setenv("S3_ACCESS_KEY", "dev_only_minio")
	t.Setenv("S3_SECRET_KEY", "dev_only_minio_secret")
	t.Setenv("S3_USE_SSL", "false")
	t.Setenv("IMAGE_MAX_BYTES", "5242880")
	t.Setenv("IMAGE_URL_TTL", "15m")
	t.Setenv("PRODUCT_PAGE_SIZE_MAX", "100")
	t.Setenv("PRODUCT_RESOLVE_BATCH_MAX", "100")
	t.Setenv("MAX_REQUEST_BODY_BYTES", "1048576")
	t.Setenv("DEFAULT_CURRENCY", "USD")
	t.Setenv("PRINCIPAL_SIGNING_KEY", validSigningKey)
}

func TestLoadRejectsAMissingDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("PORT", config.DefaultPort)
	setValidCatalogEnv(t)

	if _, err := config.Load(); err == nil {
		t.Fatal("a missing DATABASE_URL must be fatal at startup")
	}
}

func TestLoadAppliesTheDefaultPort(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5433/catalog_db")
	t.Setenv("PORT", "")
	setValidCatalogEnv(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("want success, got %v", err)
	}
	if cfg.Port != config.DefaultPort {
		t.Errorf("port: want %q, got %q", config.DefaultPort, cfg.Port)
	}
	if cfg.ServiceName != "catalog" {
		t.Errorf("service name: want catalog, got %q", cfg.ServiceName)
	}
}

func TestLoadRejectsAZeroImageMaxBytes(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5433/catalog_db")
	setValidCatalogEnv(t)
	t.Setenv("IMAGE_MAX_BYTES", "0")

	if _, err := config.Load(); err == nil {
		t.Fatal("a zero IMAGE_MAX_BYTES must be rejected")
	}
}

func TestLoadRejectsAZeroProductPageSizeMax(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5433/catalog_db")
	setValidCatalogEnv(t)
	t.Setenv("PRODUCT_PAGE_SIZE_MAX", "0")

	if _, err := config.Load(); err == nil {
		t.Fatal("a zero PRODUCT_PAGE_SIZE_MAX must be rejected")
	}
}

func TestLoadRejectsAZeroResolveBatchMax(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5433/catalog_db")
	setValidCatalogEnv(t)
	t.Setenv("PRODUCT_RESOLVE_BATCH_MAX", "0")

	if _, err := config.Load(); err == nil {
		t.Fatal("a zero PRODUCT_RESOLVE_BATCH_MAX must be rejected")
	}
}

func TestLoadRejectsAZeroMaxRequestBodyBytes(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5433/catalog_db")
	setValidCatalogEnv(t)
	t.Setenv("MAX_REQUEST_BODY_BYTES", "0")

	if _, err := config.Load(); err == nil {
		t.Fatal("a zero MAX_REQUEST_BODY_BYTES must be rejected")
	}
}

func TestLoadRejectsATwoLetterCurrency(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5433/catalog_db")
	setValidCatalogEnv(t)
	t.Setenv("DEFAULT_CURRENCY", "US")

	if _, err := config.Load(); err == nil {
		t.Fatal("a two-letter DEFAULT_CURRENCY must be rejected")
	}
}

func TestLoadRejectsAShortSigningKey(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5433/catalog_db")
	setValidCatalogEnv(t)
	t.Setenv("PRINCIPAL_SIGNING_KEY", "0123456789012345678901234567890") // 31 bytes

	if _, err := config.Load(); err == nil {
		t.Fatal("a 31-byte signing key must be rejected")
	}
}

// A missing S3_ENDPOINT must say so by name, not surface a downstream parser's internals.
func TestLoadReportsAMissingS3EndpointByName(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5433/catalog_db")
	setValidCatalogEnv(t)
	t.Setenv("S3_ENDPOINT", "")

	_, err := config.Load()
	if err == nil {
		t.Fatal("a missing S3_ENDPOINT must be fatal at startup")
	}
	if !strings.Contains(err.Error(), "S3_ENDPOINT") {
		t.Fatalf("error must name the missing variable: %v", err)
	}
}

func TestLoadReportsAMissingImageMaxBytesByName(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5433/catalog_db")
	setValidCatalogEnv(t)
	t.Setenv("IMAGE_MAX_BYTES", "")

	_, err := config.Load()
	if err == nil {
		t.Fatal("a missing IMAGE_MAX_BYTES must be fatal at startup")
	}
	if !strings.Contains(err.Error(), "IMAGE_MAX_BYTES") {
		t.Fatalf("error must name the missing variable: %v", err)
	}
	if strings.Contains(err.Error(), "strconv") {
		t.Fatalf("error leaked the parser's internals instead of naming the missing variable: %v", err)
	}
}

func TestLoadAcceptsAValidConfig(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5433/catalog_db")
	setValidCatalogEnv(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("want success, got %v", err)
	}
	if cfg.S3Bucket != "catalog-images" {
		t.Errorf("s3 bucket: want catalog-images, got %q", cfg.S3Bucket)
	}
	if cfg.ImageMaxBytes != 5242880 {
		t.Errorf("image max bytes: want 5242880, got %d", cfg.ImageMaxBytes)
	}
	if cfg.ProductPageSizeMax != 100 {
		t.Errorf("product page size max: want 100, got %d", cfg.ProductPageSizeMax)
	}
	if cfg.ResolveBatchMax != 100 {
		t.Errorf("resolve batch max: want 100, got %d", cfg.ResolveBatchMax)
	}
	if cfg.MaxRequestBodyBytes != 1048576 {
		t.Errorf("max request body bytes: want 1048576, got %d", cfg.MaxRequestBodyBytes)
	}
	if cfg.DefaultCurrency != "USD" {
		t.Errorf("default currency: want USD, got %q", cfg.DefaultCurrency)
	}
}

// A presigned URL names a host the browser reaches, not the one this service
// dials, so the two endpoints are separate settings and neither may default to
// the other: a silent fallback produced URLs no client could fetch.
func TestLoadReportsAMissingPublicEndpointByName(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5433/catalog_db")
	setValidCatalogEnv(t)
	t.Setenv("S3_PUBLIC_ENDPOINT", "")

	_, err := config.Load()
	if err == nil {
		t.Fatal("a missing S3_PUBLIC_ENDPOINT was accepted")
	}
	if !strings.Contains(err.Error(), "S3_PUBLIC_ENDPOINT") {
		t.Errorf("error must name the variable, got %q", err)
	}
}
