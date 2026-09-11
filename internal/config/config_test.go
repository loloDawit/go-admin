package config

import "testing"

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
