package config_test

import (
	"testing"

	"github.com/loloDawit/go-admin/services/orders/internal/config"
)

func TestLoadRejectsAMissingDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("PORT", config.DefaultPort)

	if _, err := config.Load(); err == nil {
		t.Fatal("a missing DATABASE_URL must be fatal at startup")
	}
}

func TestLoadAppliesTheDefaultPort(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5433/orders_db")
	t.Setenv("PORT", "")

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
