package config_test

import (
	"testing"

	"github.com/loloDawit/go-admin/services/gateway/internal/config"
)

const validSigningKey = "test_only_gateway_signing_key_32b" // 34 bytes

func setValidEnv(t *testing.T) {
	t.Helper()
	t.Setenv("IDENTITY_URL", "http://identity:8081")
	t.Setenv("CATALOG_URL", "http://catalog:8082")
	t.Setenv("ORDERS_URL", "http://orders:8083")
	t.Setenv("PRINCIPAL_SIGNING_KEY", validSigningKey)
	t.Setenv("PRINCIPAL_TTL", "30s")
	t.Setenv("SESSION_CACHE_TTL", "10s")
	t.Setenv("RATE_LIMIT_PER_SECOND", "50")
	t.Setenv("RATE_LIMIT_BURST", "100")
	t.Setenv("LOGIN_RATE_LIMIT_PER_SECOND", "0.2")
	t.Setenv("LOGIN_RATE_LIMIT_BURST", "5")
}

func TestLoadRequiresEveryUpstream(t *testing.T) {
	setValidEnv(t)
	t.Setenv("ORDERS_URL", "")

	if _, err := config.Load(); err == nil {
		t.Fatal("a missing upstream URL must be fatal at startup")
	}
}

func TestLoadAcceptsACompleteConfiguration(t *testing.T) {
	setValidEnv(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("want success, got %v", err)
	}
	if len(cfg.Upstreams) != 3 {
		t.Errorf("want three upstreams, got %d", len(cfg.Upstreams))
	}
	if cfg.PrincipalTTL.String() != "30s" {
		t.Errorf("PrincipalTTL: want 30s, got %v", cfg.PrincipalTTL)
	}
	if cfg.SessionCacheTTL.String() != "10s" {
		t.Errorf("SessionCacheTTL: want 10s, got %v", cfg.SessionCacheTTL)
	}
}

func TestLoadRejectsAShortSigningKey(t *testing.T) {
	setValidEnv(t)
	t.Setenv("PRINCIPAL_SIGNING_KEY", "too-short")

	if _, err := config.Load(); err == nil {
		t.Fatal("a signing key under 32 bytes must be fatal at startup")
	}
}

func TestLoadRejectsAMissingPrincipalTTL(t *testing.T) {
	setValidEnv(t)
	t.Setenv("PRINCIPAL_TTL", "")

	if _, err := config.Load(); err == nil {
		t.Fatal("a missing PRINCIPAL_TTL must be fatal at startup")
	}
}

func TestLoadRejectsAnUnparseablePrincipalTTL(t *testing.T) {
	setValidEnv(t)
	t.Setenv("PRINCIPAL_TTL", "not-a-duration")

	if _, err := config.Load(); err == nil {
		t.Fatal("an unparseable PRINCIPAL_TTL must be fatal at startup")
	}
}

func TestLoadRejectsAMissingSessionCacheTTL(t *testing.T) {
	setValidEnv(t)
	t.Setenv("SESSION_CACHE_TTL", "")

	if _, err := config.Load(); err == nil {
		t.Fatal("a missing SESSION_CACHE_TTL must be fatal at startup")
	}
}

func TestLoadAcceptsASessionCacheTTLLongerThanPrincipalTTL(t *testing.T) {
	setValidEnv(t)
	t.Setenv("PRINCIPAL_TTL", "30s")
	t.Setenv("SESSION_CACHE_TTL", "60s")

	if _, err := config.Load(); err != nil {
		t.Fatalf("want success (the two TTLs are independent), got %v", err)
	}
}

// The default profile is production, so an operator who sets neither
// APP_PROFILE nor a fault knob gets a gateway that would refuse injection.
func TestFaultInjectionIsRefusedOutsideDev(t *testing.T) {
	setValidEnv(t)
	t.Setenv("FAULT_ERROR_RATE", "0.5")

	if _, err := config.Load(); err == nil {
		t.Fatal("fault injection was accepted without APP_PROFILE=dev")
	}
}

func TestFaultInjectionIsAcceptedInDev(t *testing.T) {
	setValidEnv(t)
	t.Setenv("APP_PROFILE", "dev")
	t.Setenv("FAULT_LATENCY_MS", "200")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Faults.LatencyMS != 200 {
		t.Fatalf("LatencyMS = %d", cfg.Faults.LatencyMS)
	}
}
