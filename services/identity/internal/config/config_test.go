package config_test

import (
	"strings"
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
	t.Setenv("MAX_REQUEST_BODY_BYTES", "1048576")
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

// A missing BCRYPT_COST must say so by name, not surface strconv's "invalid syntax" for an empty string.
func TestLoadReportsAMissingBcryptCostByName(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5433/identity_db")
	setValidAuthEnv(t)
	t.Setenv("BCRYPT_COST", "")

	_, err := config.Load()
	if err == nil {
		t.Fatal("a missing BCRYPT_COST must be fatal at startup")
	}
	if !strings.Contains(err.Error(), "BCRYPT_COST") {
		t.Fatalf("error must name the missing variable: %v", err)
	}
	if strings.Contains(err.Error(), "strconv") {
		t.Fatalf("error leaked the parser's internals instead of naming the missing variable: %v", err)
	}
}

func TestLoadReportsAMissingSessionTTLByName(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5433/identity_db")
	setValidAuthEnv(t)
	t.Setenv("SESSION_TTL", "")

	_, err := config.Load()
	if err == nil {
		t.Fatal("a missing SESSION_TTL must be fatal at startup")
	}
	if !strings.Contains(err.Error(), "SESSION_TTL") {
		t.Fatalf("error must name the missing variable: %v", err)
	}
	if strings.Contains(err.Error(), "time:") {
		t.Fatalf("error leaked the parser's internals instead of naming the missing variable: %v", err)
	}
}

func TestLoadReportsAMissingCookieSecureByName(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5433/identity_db")
	setValidAuthEnv(t)
	t.Setenv("COOKIE_SECURE", "")

	_, err := config.Load()
	if err == nil {
		t.Fatal("a missing COOKIE_SECURE must be fatal at startup")
	}
	if !strings.Contains(err.Error(), "COOKIE_SECURE") {
		t.Fatalf("error must name the missing variable: %v", err)
	}
	if strings.Contains(err.Error(), "strconv") {
		t.Fatalf("error leaked the parser's internals instead of naming the missing variable: %v", err)
	}
}

func TestLoadReportsAMissingMaxRequestBodyBytesByName(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5433/identity_db")
	setValidAuthEnv(t)
	t.Setenv("MAX_REQUEST_BODY_BYTES", "")

	_, err := config.Load()
	if err == nil {
		t.Fatal("a missing MAX_REQUEST_BODY_BYTES must be fatal at startup")
	}
	if !strings.Contains(err.Error(), "MAX_REQUEST_BODY_BYTES") {
		t.Fatalf("error must name the missing variable: %v", err)
	}
	if strings.Contains(err.Error(), "strconv") {
		t.Fatalf("error leaked the parser's internals instead of naming the missing variable: %v", err)
	}
}

func TestLoadRejectsANonPositiveMaxRequestBodyBytes(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5433/identity_db")
	setValidAuthEnv(t)
	t.Setenv("MAX_REQUEST_BODY_BYTES", "0")

	if _, err := config.Load(); err == nil {
		t.Fatal("a non-positive MAX_REQUEST_BODY_BYTES must be rejected")
	}
}

func TestLoadAcceptsAValidMaxRequestBodyBytes(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5433/identity_db")
	setValidAuthEnv(t)
	t.Setenv("MAX_REQUEST_BODY_BYTES", "2097152")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("want success, got %v", err)
	}
	if cfg.MaxRequestBodyBytes != 2097152 {
		t.Errorf("max request body bytes: want 2097152, got %d", cfg.MaxRequestBodyBytes)
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
