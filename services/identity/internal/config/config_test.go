package config_test

import (
	"testing"
	"time"

	"github.com/loloDawit/go-admin/services/identity/internal/config"
)

const validSigningKey = "dev_only_principal_key_at_least_32_bytes"

func setValidAuthEnv(t *testing.T) {
	t.Helper()
	t.Setenv("BCRYPT_COST", "4")
	t.Setenv("SESSION_TTL", "24h")
	t.Setenv("COOKIE_SECURE", "false")
	t.Setenv("PRINCIPAL_SIGNING_KEY", validSigningKey)
}

func TestLoadRejectsAMissingDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("PORT", config.DefaultPort)
	setValidAuthEnv(t)

	if _, err := config.Load(); err == nil {
		t.Fatal("a missing DATABASE_URL must be fatal at startup")
	}
}

func TestLoadAppliesTheDefaultPort(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5433/identity_db")
	t.Setenv("PORT", "")
	setValidAuthEnv(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("want success, got %v", err)
	}
	if cfg.Port != config.DefaultPort {
		t.Errorf("port: want %q, got %q", config.DefaultPort, cfg.Port)
	}
	if cfg.ServiceName != "identity" {
		t.Errorf("service name: want identity, got %q", cfg.ServiceName)
	}
}

func TestLoadRejectsAShortSigningKey(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5433/identity_db")
	setValidAuthEnv(t)
	t.Setenv("PRINCIPAL_SIGNING_KEY", "0123456789012345678901234567890") // 31 bytes

	if _, err := config.Load(); err == nil {
		t.Fatal("a 31-byte signing key must be rejected")
	}
}

func TestLoadRejectsAnOutOfRangeCost(t *testing.T) {
	cases := []string{"3", "32"}
	for _, cost := range cases {
		t.Run(cost, func(t *testing.T) {
			t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5433/identity_db")
			setValidAuthEnv(t)
			t.Setenv("BCRYPT_COST", cost)

			if _, err := config.Load(); err == nil {
				t.Fatalf("bcrypt cost %s must be rejected", cost)
			}
		})
	}
}

func TestLoadRejectsAnInvalidSessionTTL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5433/identity_db")
	setValidAuthEnv(t)
	t.Setenv("SESSION_TTL", "not-a-duration")

	if _, err := config.Load(); err == nil {
		t.Fatal("an invalid SESSION_TTL must be rejected")
	}
}

func TestLoadAcceptsValidAuthConfiguration(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5433/identity_db")
	setValidAuthEnv(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("want success, got %v", err)
	}
	if cfg.BcryptCost != 4 {
		t.Errorf("bcrypt cost: want 4, got %d", cfg.BcryptCost)
	}
	if cfg.SessionTTL != 24*time.Hour {
		t.Errorf("session ttl: want 24h, got %v", cfg.SessionTTL)
	}
	if cfg.CookieSecure {
		t.Error("cookie secure: want false")
	}
	if string(cfg.PrincipalKey) != validSigningKey {
		t.Errorf("principal key: want %q, got %q", validSigningKey, cfg.PrincipalKey)
	}
}
