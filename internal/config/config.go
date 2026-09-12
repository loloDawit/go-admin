// Package config loads and validates runtime configuration. It is the only
// place the application reads os.Getenv.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/loloDawit/go-admin/internal/auth"
	"golang.org/x/crypto/bcrypt"
)

const minSecretLen = 32 // SHA-256 output width, which HS256 keys should match

// UploadsPath is the mount point for uploaded files. Shared by the static
// route and the URL the upload handler returns, so the two can't drift apart.
const UploadsPath = "/api/v1/uploads"

type Config struct {
	DBDSN          string
	SessionSecret  string
	Port           string
	AllowedOrigin  string
	PublicBaseURL  string
	UploadDir      string
	MaxUploadBytes int64
	AppEnv         string
	BcryptCost     int
}

// Gates the cookie Secure flag, which would break login over plain HTTP.
func (c *Config) IsProduction() bool { return c.AppEnv == "production" }

func (c *Config) Hasher() auth.Hasher { return auth.NewHasher(c.BcryptCost) }

// BodyLimit is the Fiber server limit, not the upload cap itself: headroom
// above MaxUploadBytes for multipart framing overhead, so the handler's own
// check (not a raw connection reset) is what rejects an oversized upload.
func (c *Config) BodyLimit() int { return int(c.MaxUploadBytes) + (1 << 20) }

// Any error from Load is fatal: the application must not start.
func Load() (*Config, error) {
	cfg := &Config{
		DBDSN:         os.Getenv("DB_DSN"),
		SessionSecret: os.Getenv("SESSION_SECRET"),
		Port:          withDefault("PORT", "8080"),
		AllowedOrigin: os.Getenv("ALLOWED_ORIGIN"),
		PublicBaseURL: withDefault("PUBLIC_BASE_URL", "http://localhost:8080"),
		UploadDir:     withDefault("UPLOAD_DIR", "./uploads"),
		AppEnv:        withDefault("APP_ENV", "development"),
	}

	maxUpload, err := strconv.ParseInt(withDefault("MAX_UPLOAD_BYTES", "5242880"), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("MAX_UPLOAD_BYTES must be an integer: %w", err)
	}
	cfg.MaxUploadBytes = maxUpload

	bcryptCost, err := strconv.Atoi(withDefault("BCRYPT_COST", strconv.Itoa(auth.DefaultCost)))
	if err != nil {
		return nil, fmt.Errorf("BCRYPT_COST must be an integer: %w", err)
	}
	cfg.BcryptCost = bcryptCost

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	var errs []error

	if c.DBDSN == "" {
		errs = append(errs, errors.New("DB_DSN is required (see .env.example)"))
	} else if dsn, err := mysqldriver.ParseDSN(c.DBDSN); err != nil {
		errs = append(errs, fmt.Errorf("DB_DSN is not a valid MySQL DSN: %w", err))
	} else if !dsn.ParseTime {
		// Models read/write time.Time columns (e.g. Order.CreatedAt); without
		// this the driver hands GORM []uint8 instead, and every read fails
		// at request time rather than at boot. Checked via the driver's own
		// parser (strconv.ParseBool under the hood), not a substring match:
		// parseTime=True/1/t are all valid to the driver and must not be
		// rejected here.
		errs = append(errs, errors.New(
			"DB_DSN must set parseTime=true (see .env.example)"))
	}
	if len(c.SessionSecret) < minSecretLen {
		errs = append(errs, fmt.Errorf(
			"SESSION_SECRET must be at least %d bytes; generate one with `openssl rand -hex 32`",
			minSecretLen))
	}
	if c.AllowedOrigin == "" {
		errs = append(errs, errors.New(
			"ALLOWED_ORIGIN is required; wildcard CORS with credentials is not permitted"))
	} else {
		for _, origin := range strings.Split(c.AllowedOrigin, ",") {
			if strings.TrimSpace(origin) == "*" {
				errs = append(errs, errors.New(
					"ALLOWED_ORIGIN must not be or contain \"*\"; wildcard CORS with credentials is not permitted"))
				break
			}
		}
	}
	if c.MaxUploadBytes <= 0 {
		errs = append(errs, errors.New("MAX_UPLOAD_BYTES must be positive"))
	}
	if c.BcryptCost < auth.MinAllowedCost || c.BcryptCost > bcrypt.MaxCost {
		errs = append(errs, fmt.Errorf(
			"BCRYPT_COST must be between %d and %d (default %d)",
			auth.MinAllowedCost, bcrypt.MaxCost, auth.DefaultCost))
	}

	return errors.Join(errs...)
}

func withDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
