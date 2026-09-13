package config_test

import (
	"testing"

	"github.com/loloDawit/go-admin/services/gateway/internal/config"
)

func TestLoadRequiresEveryUpstream(t *testing.T) {
	t.Setenv("IDENTITY_URL", "http://identity:8081")
	t.Setenv("CATALOG_URL", "http://catalog:8082")
	t.Setenv("ORDERS_URL", "")

	if _, err := config.Load(); err == nil {
		t.Fatal("a missing upstream URL must be fatal at startup")
	}
}

func TestLoadAcceptsACompleteConfiguration(t *testing.T) {
	t.Setenv("IDENTITY_URL", "http://identity:8081")
	t.Setenv("CATALOG_URL", "http://catalog:8082")
	t.Setenv("ORDERS_URL", "http://orders:8083")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("want success, got %v", err)
	}
	if len(cfg.Upstreams) != 3 {
		t.Errorf("want three upstreams, got %d", len(cfg.Upstreams))
	}
}
