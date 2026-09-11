package config

import (
	"testing"

	"github.com/loloDawit/go-admin/internal/auth"
)

func TestLoadRejectsMissingDSN(t *testing.T) {
	t.Setenv("DB_DSN", "")
	t.Setenv("SESSION_SECRET", "0123456789abcdef0123456789abcdef")
	t.Setenv("ALLOWED_ORIGIN", "http://localhost:3000")

	if _, err := Load(); err == nil {
		t.Fatal("expected an error when DB_DSN is unset, got nil")
	}
}

func TestLoadRejectsShortSecret(t *testing.T) {
	t.Setenv("DB_DSN", "user:pass@tcp(127.0.0.1:3306)/go_admin?parseTime=true")
	t.Setenv("SESSION_SECRET", "tooshort")
	t.Setenv("ALLOWED_ORIGIN", "http://localhost:3000")

	if _, err := Load(); err == nil {
		t.Fatal("expected an error for a secret under 32 bytes, got nil")
	}
}

func TestLoadRejectsMissingOrigin(t *testing.T) {
	t.Setenv("DB_DSN", "user:pass@tcp(127.0.0.1:3306)/go_admin?parseTime=true")
	t.Setenv("SESSION_SECRET", "0123456789abcdef0123456789abcdef")
	t.Setenv("ALLOWED_ORIGIN", "")

	if _, err := Load(); err == nil {
		t.Fatal("expected an error when ALLOWED_ORIGIN is unset, got nil")
	}
}

func TestLoadAppliesDefaults(t *testing.T) {
	t.Setenv("DB_DSN", "user:pass@tcp(127.0.0.1:3306)/go_admin?parseTime=true")
	t.Setenv("SESSION_SECRET", "0123456789abcdef0123456789abcdef")
	t.Setenv("ALLOWED_ORIGIN", "http://localhost:3000")
	t.Setenv("PORT", "")
	t.Setenv("UPLOAD_DIR", "")
	t.Setenv("APP_ENV", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected a valid config, got error: %v", err)
	}
	if cfg.Port != "8080" {
		t.Errorf("Port: want 8080, got %q", cfg.Port)
	}
	if cfg.UploadDir != "./uploads" {
		t.Errorf("UploadDir: want ./uploads, got %q", cfg.UploadDir)
	}
	if cfg.MaxUploadBytes != 5<<20 {
		t.Errorf("MaxUploadBytes: want 5242880, got %d", cfg.MaxUploadBytes)
	}
	if cfg.IsProduction() {
		t.Error("APP_ENV should default to development, not production")
	}
}

func TestIsProduction(t *testing.T) {
	t.Setenv("DB_DSN", "user:pass@tcp(127.0.0.1:3306)/go_admin?parseTime=true")
	t.Setenv("SESSION_SECRET", "0123456789abcdef0123456789abcdef")
	t.Setenv("ALLOWED_ORIGIN", "https://admin.example.com")
	t.Setenv("APP_ENV", "production")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.IsProduction() {
		t.Error("APP_ENV=production must report IsProduction() == true")
	}
}

func validEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DB_DSN", "user:pass@tcp(127.0.0.1:3306)/go_admin?parseTime=true")
	t.Setenv("SESSION_SECRET", "0123456789abcdef0123456789abcdef")
	t.Setenv("ALLOWED_ORIGIN", "http://localhost:3000")
}

func TestBcryptCostDefaultsAndValidates(t *testing.T) {
	cases := []struct {
		name    string
		value   string
		wantErr bool
		want    int
	}{
		{name: "defaults when unset", value: "", want: auth.DefaultCost},
		{name: "accepts the minimum", value: "10", want: 10},
		{name: "accepts a high cost", value: "15", want: 15},
		{name: "rejects below the floor", value: "4", wantErr: true},
		{name: "rejects zero", value: "0", wantErr: true},
		{name: "rejects above bcrypt max", value: "99", wantErr: true},
		{name: "rejects non-numeric", value: "strong", wantErr: true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			validEnv(t)
			t.Setenv("BCRYPT_COST", c.value)

			cfg, err := Load()
			if c.wantErr {
				if err == nil {
					t.Fatalf("BCRYPT_COST=%q must be rejected", c.value)
				}
				return
			}
			if err != nil {
				t.Fatalf("BCRYPT_COST=%q should be valid: %v", c.value, err)
			}
			if cfg.BcryptCost != c.want {
				t.Errorf("BcryptCost: want %d, got %d", c.want, cfg.BcryptCost)
			}
			if got := cfg.Hasher().Cost(); got != c.want {
				t.Errorf("Hasher().Cost(): want %d, got %d", c.want, got)
			}
		})
	}
}
