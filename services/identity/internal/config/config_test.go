package config_test

import (
	"testing"

	"github.com/loloDawit/go-admin/services/identity/internal/config"
)

func TestLoadRejectsAMissingDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("PORT", "8081")

	if _, err := config.Load(); err == nil {
		t.Fatal("a missing DATABASE_URL must be fatal at startup")
	}
}

func TestLoadAppliesTheDefaultPort(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5433/identity_db")
	t.Setenv("PORT", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("want success, got %v", err)
	}
	if cfg.Port != "8081" {
		t.Errorf("port: want 8081, got %q", cfg.Port)
	}
	if cfg.ServiceName != "identity" {
		t.Errorf("service name: want identity, got %q", cfg.ServiceName)
	}
}
