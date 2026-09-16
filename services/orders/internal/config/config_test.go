package config_test

import (
	"testing"

	"github.com/loloDawit/go-admin/services/orders/internal/config"
)

const validSigningKey = "dev_only_principal_key_at_least_32_bytes"

func setValidOrdersEnv(t *testing.T) {
	t.Helper()
	t.Setenv("CATALOG_URL", "http://catalog:8082")
	t.Setenv("CATALOG_TIMEOUT", "2s")
	t.Setenv("CATALOG_MAX_IDLE_CONNS", "20")
	t.Setenv("ORDER_PAGE_SIZE_MAX", "100")
	t.Setenv("MAX_REQUEST_BODY_BYTES", "1048576")
	t.Setenv("PRINCIPAL_SIGNING_KEY", validSigningKey)
	t.Setenv("NATS_URL", "nats://nats:4222")
	t.Setenv("NATS_DUPLICATE_WINDOW", "2m")
	t.Setenv("OUTBOX_BATCH_SIZE", "100")
	t.Setenv("OUTBOX_POLL_INTERVAL", "500ms")
	t.Setenv("REPORT_WINDOW_DAYS", "30")
}

func TestLoadRejectsAMissingDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("PORT", config.DefaultPort)
	setValidOrdersEnv(t)

	if _, err := config.Load(); err == nil {
		t.Fatal("a missing DATABASE_URL must be fatal at startup")
	}
}

func TestLoadAppliesTheDefaultPort(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5433/orders_db")
	t.Setenv("PORT", "")
	setValidOrdersEnv(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("want success, got %v", err)
	}
	if cfg.Port != config.DefaultPort {
		t.Errorf("port: want %q, got %q", config.DefaultPort, cfg.Port)
	}
	if cfg.ServiceName != "orders" {
		t.Errorf("service name: want orders, got %q", cfg.ServiceName)
	}
}

func TestLoadRejectsAMissingCatalogURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5433/orders_db")
	setValidOrdersEnv(t)
	t.Setenv("CATALOG_URL", "")

	if _, err := config.Load(); err == nil {
		t.Fatal("a missing CATALOG_URL must be fatal at startup")
	}
}

func TestLoadRejectsAnUnparsableCatalogTimeout(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5433/orders_db")
	setValidOrdersEnv(t)
	t.Setenv("CATALOG_TIMEOUT", "not-a-duration")

	if _, err := config.Load(); err == nil {
		t.Fatal("an unparsable CATALOG_TIMEOUT must be fatal at startup")
	}
}

func TestLoadRejectsAZeroCatalogTimeout(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5433/orders_db")
	setValidOrdersEnv(t)
	t.Setenv("CATALOG_TIMEOUT", "0s")

	if _, err := config.Load(); err == nil {
		t.Fatal("a zero CATALOG_TIMEOUT must be rejected")
	}
}

func TestLoadRejectsAZeroCatalogMaxIdleConns(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5433/orders_db")
	setValidOrdersEnv(t)
	t.Setenv("CATALOG_MAX_IDLE_CONNS", "0")

	if _, err := config.Load(); err == nil {
		t.Fatal("a zero CATALOG_MAX_IDLE_CONNS must be rejected: an unbounded pool turns a slow Catalog into an Orders outage")
	}
}

func TestLoadRejectsAZeroOrderPageSizeMax(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5433/orders_db")
	setValidOrdersEnv(t)
	t.Setenv("ORDER_PAGE_SIZE_MAX", "0")

	if _, err := config.Load(); err == nil {
		t.Fatal("a zero ORDER_PAGE_SIZE_MAX must be rejected")
	}
}

func TestLoadRejectsAZeroMaxRequestBodyBytes(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5433/orders_db")
	setValidOrdersEnv(t)
	t.Setenv("MAX_REQUEST_BODY_BYTES", "0")

	if _, err := config.Load(); err == nil {
		t.Fatal("a zero MAX_REQUEST_BODY_BYTES must be rejected")
	}
}

func TestLoadRejectsAShortPrincipalKey(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5433/orders_db")
	setValidOrdersEnv(t)
	t.Setenv("PRINCIPAL_SIGNING_KEY", "too-short")

	if _, err := config.Load(); err == nil {
		t.Fatal("a PRINCIPAL_SIGNING_KEY under 32 bytes must be rejected")
	}
}

func TestLoadCarriesCatalogSettingsThrough(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5433/orders_db")
	setValidOrdersEnv(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("want success, got %v", err)
	}
	if cfg.CatalogURL != "http://catalog:8082" {
		t.Errorf("catalog url: want http://catalog:8082, got %q", cfg.CatalogURL)
	}
	if cfg.CatalogTimeout.String() != "2s" {
		t.Errorf("catalog timeout: want 2s, got %v", cfg.CatalogTimeout)
	}
	if cfg.CatalogMaxIdleConns != 20 {
		t.Errorf("catalog max idle conns: want 20, got %d", cfg.CatalogMaxIdleConns)
	}
	if cfg.OrderPageSizeMax != 100 {
		t.Errorf("order page size max: want 100, got %d", cfg.OrderPageSizeMax)
	}
	if cfg.MaxRequestBodyBytes != 1048576 {
		t.Errorf("max request body bytes: want 1048576, got %d", cfg.MaxRequestBodyBytes)
	}
	if string(cfg.PrincipalKey) != validSigningKey {
		t.Errorf("principal key: want %q, got %q", validSigningKey, cfg.PrincipalKey)
	}
}
