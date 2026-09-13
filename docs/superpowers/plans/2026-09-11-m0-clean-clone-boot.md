# M0 — "Boots Safely From a Clean Clone" Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `git clone && make dev` produce a running, seeded, non-compromised application, and close every critical security defect in the code that exists today.

**Architecture:** Deliberately *not* a restructure. M0 keeps Fiber, GORM, and MySQL, and keeps the existing `controllers/` / `models/` / `middlewares/` package layout. It adds one new package (`internal/config`), one test harness (`internal/testutil`), and one command (`cmd/seed`). The architectural rework — `cmd/` + `internal/{domain,http,store}`, migrations, service/repository layers — is M1 and must not be started here. M0's job is to make the existing application safe and verifiable so that M1 has something to restructure *against*.

**Tech Stack:** Go 1.27, Fiber v2, GORM v1.23 + MySQL 8, testcontainers-go, React 18 + CRA (frontend touched only where the contract is wrong), Docker Compose.

**Spec:** `docs/ASSESSMENT.md` §10 (milestone M0) and §12. Defect IDs below (§4a, §4l, …) refer to `docs/ASSESSMENT.md` §4.

**Decision record:** `docs/decisions/0001-remove-public-registration.md` — read it before Task 6.

## Global Constraints

- **Go 1.27**, toolchain `auto`. Bump `go.mod` from `go 1.17` to `go 1.27`.
- **Module path is `github.com/loloDawit/go-admin`** (merged in PR #1). Never reintroduce the former internal host path.
- **No secrets in source.** Every credential comes from the environment. `.env` stays gitignored; `.env.example` is committed with placeholder values only.
- **The application must refuse to start on invalid config.** Missing `DB_DSN`, missing/short `SESSION_SECRET`, or missing `ALLOWED_ORIGIN` is a fatal startup error, never a runtime surprise.
- **Tests must not require manual setup.** `make test` spins its own MySQL via testcontainers. No developer ever hand-creates a test database.
- **Tests come first.** Every defect fix in Tasks 3–11 begins with a test that *fails against current code*. A fix whose test passes before the fix is applied is a bad test — rewrite it.
- **`database.DB` stays a package-level global in M0.** Tests assign to it directly. This is a known wart, documented, and removed in M1 via dependency injection. Do not fix it here.
- **Do not change frontend framework, styling, or add screens.** Frontend changes in M0 are limited to: deleting the registration page, correcting the broken type contract, removing `//@ts-ignore`, and adding error handling to login.
- **Never construct an error inline.** Every failure returns a value from
  `internal/errs` via `return httpx.Fail(ctx, errs.Something)`. Adding a new
  failure mode means adding it to `internal/errs/registry.go` first —
  `registry_test.go` enforces unique codes, well-formed shapes, and that no
  5xx message leaks internals. Never `errors.New` in a handler, and never put
  a driver error in a client-facing message; wrap it as the cause instead
  (`errs.Database.Wrap(err)`), which logs it without serializing it.
- **Never hardcode a work factor, limit, or timeout.** Anything an operator
  might tune belongs in `internal/config` with validation. `SetPassword` takes
  an `auth.Hasher`; tests use `auth.NewTestHasher()`.
- **Comments explain constraints, not changes.** A comment earns its place
  only if it states something you would violate by "improving" the code — a
  map-based `Updates` that must not become a struct, two branches that must
  return the same error. Do not restate what the code says, do not narrate
  what the old code did (git and `docs/ASSESSMENT.md` hold that), and do not
  write prose. One line where one line does. The commit message is where the
  reasoning goes.
- **Commit after every task.** Conventional commit prefixes (`feat:`, `fix:`, `test:`, `chore:`, `docs:`).

---

## File Structure

**Created**

| Path | Responsibility |
|---|---|
| `internal/config/config.go` | Typed config loaded from env; fail-fast validation. Single source of truth for every tunable. |
| `internal/config/config_test.go` | Unit tests for validation rules. No database. |
| `internal/testutil/db.go` | Spins a MySQL testcontainer, runs `AutoMigrate`, truncates between tests. |
| `internal/testutil/app.go` | Builds a Fiber app wired to the test DB; helpers for authenticated requests. |
| ~~`internal/auth/password.go`~~ | **Built in Task 3.** `auth.Hasher` — a value carrying its bcrypt cost, so the work factor is an explicit dependency. |
| `internal/auth/password_test.go` | Proves a hashed password verifies and a wrong one does not. |
| `internal/httpx/dto.go` | Typed request DTOs replacing `map[string]string`. |
| ~~`internal/httpx/respond.go`~~ | **Built in Task 3.** `httpx.Fail(ctx, err)` maps any error to its response. |
| ~~`internal/errs/`~~ | **Built in Task 3.** The single registry of every application error (code, client-safe message, HTTP status). |
| `cmd/seed/main.go` | Idempotent seeding of permissions, roles, and the owner account. |
| `docker-compose.yml` | MySQL 8 for local development. |
| `.env.example` | Committed template. Placeholders only. |
| `.github/workflows/ci.yml` | vet, test, gofmt, frontend build on every PR. |
| `controllers/*_test.go` | Integration tests per controller. |

**Modified**

| Path | Change |
|---|---|
| `main.go` | Load config, fail fast, pass DSN to `database.Connect`, configure CORS from config. |
| `database/db.go` | Accept a DSN parameter; return an error instead of `panic`. |
| `models/user.go` | `SetPassword` delegates to `internal/auth` and returns an error. Fix `Preload("role")`. |
| `models/order.go` | `CreatedAt`/`UpdatedAt` → `time.Time`; drop `json:"-"` from customer names. |
| `models/pagination.go` | Correct ceiling arithmetic; accept a bounded page size. |
| `controllers/auth_controller.go` | Delete `Register`. Typed DTOs. Remove token from login body. |
| `controllers/user_controller.go` | Remove the manual `IsAuthorized` calls (now route-group middleware). Admin-supplied initial password. |
| `controllers/role_controller.go` | Typed DTO replacing unchecked `fiber.Map` assertions. |
| `controllers/image_controller.go` | Filename sanitization, MIME allowlist, size cap, config-driven URL. |
| `controllers/order_controller.go` | Per-request temp CSV; decimal-safe formatting; `Export` becomes GET. |
| `middlewares/permission_middleware.go` | Becomes real Fiber middleware (`RequirePermission(resource)`). Remove `fmt.Println`. |
| `routes/routes.go` | Route groups with permission middleware. Remove `/register`. |
| `Makefile` | `dev`, `up`, `down`, `test`, `seed`, `fmt`, `lint`. |
| `go.mod` | Go 1.27; `golang-jwt/jwt` → v5; add testcontainers. |
| `clients/src/pages/Login.tsx` | Error state; remove `//@ts-ignore`; controlled inputs. |
| `clients/src/App.tsx` | Remove the `/register` route. |
| `clients/src/interfaces/user.ts` | Match the real response shape. |
| `README.md` | Real setup instructions. |

**Deleted**

- `clients/src/pages/Register.tsx`, `clients/src/pages/Register.css` (per ADR 0001)

---

## Task 1: Typed configuration with fail-fast validation

Closes §4h (hardcoded DSN in source) and §4d (empty signing key silently accepted).

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`
- Create: `.env.example`
- Modify: `go.mod` (Go 1.17 → 1.27)

**Interfaces:**
- Consumes: nothing (first task).
- Produces: `config.Config` struct with fields `DBDSN, SessionSecret, Port, AllowedOrigin, PublicBaseURL, UploadDir, MaxUploadBytes int64` and `config.Load() (*Config, error)`. Every later task reads config through this type.

- [ ] **Step 1: Bump the Go directive**

```bash
cd /Users/dawitnoah/Desktop/project/go-admin
go mod edit -go=1.27
go mod tidy
```

- [ ] **Step 2: Write the failing test**

Create `internal/config/config_test.go`:

```go
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
}
```

- [ ] **Step 3: Run the test to verify it fails**

Run: `go test ./internal/config/ -v`
Expected: FAIL — the package does not compile, `undefined: Load`.

- [ ] **Step 4: Write the implementation**

Create `internal/config/config.go`:

```go
// Package config loads and validates all runtime configuration from the
// environment. Load is the only place the application reads os.Getenv;
// everything else takes a *Config.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
)

// minSecretLen is the shortest session signing key we accept. 32 bytes is
// the output width of SHA-256, which is what HS256 keys should match.
const minSecretLen = 32

type Config struct {
	DBDSN          string
	SessionSecret  string
	Port           string
	AllowedOrigin  string
	PublicBaseURL  string
	UploadDir      string
	MaxUploadBytes int64
	AppEnv         string
}

// IsProduction reports whether the app is running in production. It gates the
// Secure flag on the session cookie: Secure cookies are not sent over plain
// HTTP, so forcing it on in local development would break login on :3000.
func (c *Config) IsProduction() bool { return c.AppEnv == "production" }

// Load reads configuration from the environment and validates it. It returns
// an error rather than panicking so that main can decide how to report the
// failure. Any error from Load is fatal: the application must not start.
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

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	var errs []error

	if c.DBDSN == "" {
		errs = append(errs, errors.New("DB_DSN is required (see .env.example)"))
	}
	if len(c.SessionSecret) < minSecretLen {
		errs = append(errs, fmt.Errorf(
			"SESSION_SECRET must be at least %d bytes; generate one with `openssl rand -hex 32`",
			minSecretLen))
	}
	if c.AllowedOrigin == "" {
		errs = append(errs, errors.New(
			"ALLOWED_ORIGIN is required; wildcard CORS with credentials is not permitted"))
	}
	if c.MaxUploadBytes <= 0 {
		errs = append(errs, errors.New("MAX_UPLOAD_BYTES must be positive"))
	}

	return errors.Join(errs...)
}

func withDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
```

- [ ] **Step 5: Run the test to verify it passes**

Run: `go test ./internal/config/ -v`
Expected: PASS — all four tests.

- [ ] **Step 6: Write `.env.example`**

Create `.env.example`:

```bash
# Copy to .env and fill in. .env is gitignored and must never be committed.

# MySQL connection string. parseTime=true is REQUIRED — GORM cannot scan
# DATETIME columns into time.Time without it.
DB_DSN=go_admin:go_admin@tcp(127.0.0.1:3306)/go_admin?parseTime=true&charset=utf8mb4&loc=UTC

# Session signing key. Minimum 32 bytes. Generate with: openssl rand -hex 32
# The application refuses to start if this is missing or too short.
SESSION_SECRET=replace-me-with-openssl-rand-hex-32

# Exact origin of the frontend. Wildcards are rejected: this API sends
# credentialed cookies, and "*" with credentials is a CSRF hole.
ALLOWED_ORIGIN=http://localhost:3000

# Public base URL of THIS api, used to build upload URLs.
PUBLIC_BASE_URL=http://localhost:8080

# "production" turns on Secure cookies (HTTPS only). Leave as development
# locally, or login over plain HTTP on :3000 will not set a cookie.
APP_ENV=development

PORT=8080
UPLOAD_DIR=./uploads
MAX_UPLOAD_BYTES=5242880

# Consumed only by `make seed`. Delete these after the owner exists.
OWNER_EMAIL=owner@example.com
OWNER_PASSWORD=change-me-immediately
```

- [ ] **Step 7: Commit**

```bash
git add go.mod go.sum internal/config .env.example
git commit -m "feat(config): typed env config with fail-fast validation

Replaces the hardcoded DSN in database/db.go and the silent empty-secret
path in utils/jwt.go. Missing DB_DSN, a short SESSION_SECRET, or a missing
ALLOWED_ORIGIN are now fatal at startup rather than surprises at request
time.

Refs ASSESSMENT 4d, 4h."
```

---

## Task 2: Local database, test harness, and Makefile

Closes §6.2 (cannot boot from a clean clone) and establishes the harness every later task's tests depend on.

**Files:**
- Create: `docker-compose.yml`
- Create: `internal/testutil/db.go`
- Create: `internal/testutil/db_test.go`
- Modify: `Makefile`
- Modify: `database/db.go`
- Modify: `main.go`

**Interfaces:**
- Consumes: `config.Config` (Task 1).
- Produces:
  - `database.Connect(dsn string) (*gorm.DB, error)` — no longer panics, no longer hardcodes.
  - `testutil.NewDB(t *testing.T) *gorm.DB` — a migrated, empty MySQL for one test. Assigns `database.DB` and registers cleanup.

- [ ] **Step 1: Write docker-compose.yml**

```yaml
services:
  db:
    image: mysql:8.4
    restart: unless-stopped
    environment:
      MYSQL_ROOT_PASSWORD: root
      MYSQL_DATABASE: go_admin
      MYSQL_USER: go_admin
      MYSQL_PASSWORD: go_admin
    ports:
      - "3306:3306"
    volumes:
      - dbdata:/var/lib/mysql
    healthcheck:
      # Without this, `make dev` races the database and the API dies on boot.
      test: ["CMD", "mysqladmin", "ping", "-h", "127.0.0.1", "-ugo_admin", "-pgo_admin"]
      interval: 3s
      timeout: 5s
      retries: 20

volumes:
  dbdata:
```

- [ ] **Step 2: Change `database.Connect` to take a DSN and return an error**

Replace the body of `database/db.go`:

```go
package database

import (
	"fmt"

	"github.com/loloDawit/go-admin/models"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// DB is a package-level global. This is a known wart inherited from the
// original code: it prevents dependency injection and forces tests to assign
// to it directly. M1 replaces it with an injected store. Do not build new
// code that depends on it being global.
var DB *gorm.DB

// Connect opens a connection using the supplied DSN and runs AutoMigrate.
// It returns an error instead of panicking so main can report it cleanly.
//
// AutoMigrate is itself a known wart (ASSESSMENT 4w) — it cannot be reviewed,
// rolled back, or ordered. M1 replaces it with golang-migrate. M0 keeps it so
// that this milestone changes no schema semantics.
func Connect(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := Migrate(db); err != nil {
		return nil, err
	}

	DB = db
	return db, nil
}

// Migrate applies the schema. Exported so the test harness can build a
// database without going through Connect's global assignment.
func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&models.User{},
		&models.Role{},
		&models.Permission{},
		&models.Product{},
		&models.Order{},
		&models.OrderItem{},
	); err != nil {
		return fmt.Errorf("automigrate: %w", err)
	}
	return nil
}
```

- [ ] **Step 3: Rewrite `main.go` to fail fast**

```go
package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/joho/godotenv"
	"github.com/loloDawit/go-admin/database"
	"github.com/loloDawit/go-admin/internal/config"
	"github.com/loloDawit/go-admin/routes"
	"github.com/loloDawit/go-admin/utils"
)

func main() {
	// .env is a local-development convenience. In production the platform
	// supplies the environment directly, so a missing file is not an error.
	if err := godotenv.Load(".env"); err != nil && !os.IsNotExist(err) {
		log.Printf("config: .env not loaded (%v); reading environment directly", err)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	utils.SecretKey = cfg.SessionSecret

	if _, err := database.Connect(cfg.DBDSN); err != nil {
		log.Fatalf("database: %v", err)
	}

	app := fiber.New(fiber.Config{
		BodyLimit: int(cfg.MaxUploadBytes) + (1 << 20), // upload cap plus headroom
	})

	// An explicit origin, never "*". Fiber rejects wildcard-with-credentials
	// at runtime, and it would be a CSRF hole regardless (ASSESSMENT 4e).
	app.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.AllowedOrigin,
		AllowCredentials: true,
		AllowHeaders:     "Content-Type",
	}))

	routes.SetupRoutes(app, cfg)

	log.Printf("listening on :%s", cfg.Port)
	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Fatalf("server: %v", err)
	}
}
```

> Note: `routes.SetupRoutes` gains a `cfg` parameter here. Update its signature in `routes/routes.go` to `func SetupRoutes(app *fiber.App, cfg *config.Config)` now; the body changes in Task 7.

- [ ] **Step 4: Add the testcontainers dependency**

```bash
go get github.com/testcontainers/testcontainers-go@latest
go get github.com/testcontainers/testcontainers-go/modules/mysql@latest
go mod tidy
```

- [ ] **Step 5: Write the test harness**

Create `internal/testutil/db.go`:

```go
// Package testutil provides integration-test infrastructure. It spins a real
// MySQL container so tests exercise the same driver, dialect, and constraint
// behaviour as production — sqlite-in-memory would hide exactly the GORM
// bugs this milestone exists to fix.
package testutil

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/loloDawit/go-admin/database"
	tcmysql "github.com/testcontainers/testcontainers-go/modules/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	once     sync.Once
	sharedDB *gorm.DB
	initErr  error
)

// NewDB returns a migrated, empty database. The container is started once per
// `go test` process and reused; each call truncates every table so tests do
// not leak state into one another.
func NewDB(t *testing.T) *gorm.DB {
	t.Helper()

	once.Do(func() {
		ctx := context.Background()

		container, err := tcmysql.Run(ctx, "mysql:8.4",
			tcmysql.WithDatabase("go_admin_test"),
			tcmysql.WithUsername("test"),
			tcmysql.WithPassword("test"),
		)
		if err != nil {
			initErr = err
			return
		}

		dsn, err := container.ConnectionString(ctx, "parseTime=true", "charset=utf8mb4", "loc=UTC")
		if err != nil {
			initErr = err
			return
		}

		// The container reports ready before MySQL accepts connections;
		// retry briefly rather than failing the whole suite on a cold start.
		var db *gorm.DB
		for i := 0; i < 30; i++ {
			db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
				Logger: logger.Default.LogMode(logger.Silent),
			})
			if err == nil {
				break
			}
			time.Sleep(time.Second)
		}
		if err != nil {
			initErr = err
			return
		}

		if err := database.Migrate(db); err != nil {
			initErr = err
			return
		}
		sharedDB = db
	})

	if initErr != nil {
		t.Fatalf("testutil: could not start MySQL container: %v", initErr)
	}

	truncateAll(t, sharedDB)

	// M0 wart: controllers read the package-level global. Assign it so
	// handlers under test hit the container. Removed in M1.
	database.DB = sharedDB

	return sharedDB
}

func truncateAll(t *testing.T, db *gorm.DB) {
	t.Helper()

	tables := []string{
		"role_permissions", "order_items", "orders",
		"products", "users", "roles", "permissions",
	}

	if err := db.Exec("SET FOREIGN_KEY_CHECKS = 0").Error; err != nil {
		t.Fatalf("testutil: disable FK checks: %v", err)
	}
	for _, table := range tables {
		if err := db.Exec("TRUNCATE TABLE " + table).Error; err != nil {
			t.Fatalf("testutil: truncate %s: %v", table, err)
		}
	}
	if err := db.Exec("SET FOREIGN_KEY_CHECKS = 1").Error; err != nil {
		t.Fatalf("testutil: re-enable FK checks: %v", err)
	}
}
```

- [ ] **Step 6: Write a harness smoke test**

Create `internal/testutil/db_test.go`:

```go
package testutil

import (
	"testing"

	"github.com/loloDawit/go-admin/models"
)

func TestNewDBIsMigratedAndEmpty(t *testing.T) {
	db := NewDB(t)

	var count int64
	if err := db.Model(&models.User{}).Count(&count).Error; err != nil {
		t.Fatalf("users table should exist and be queryable: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected an empty users table, got %d rows", count)
	}
}

func TestNewDBTruncatesBetweenTests(t *testing.T) {
	db := NewDB(t)

	if err := db.Create(&models.Permission{Name: "view_users"}).Error; err != nil {
		t.Fatalf("insert: %v", err)
	}

	// A second NewDB in the same process must wipe the row above.
	db = NewDB(t)

	var count int64
	db.Model(&models.Permission{}).Count(&count)
	if count != 0 {
		t.Fatalf("expected truncation between NewDB calls, got %d rows", count)
	}
}
```

- [ ] **Step 7: Run the harness tests**

Run: `go test ./internal/testutil/ -v`
Expected: PASS. First run pulls `mysql:8.4` and takes 30–60s; subsequent runs are fast.

- [ ] **Step 8: Rewrite the Makefile**

```makefile
.PHONY: help up down dev api web seed test fmt gofmtcheck lint tidy

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

up: ## Start MySQL and wait for it to accept connections
	docker compose up -d --wait

down: ## Stop MySQL (data is preserved in the dbdata volume)
	docker compose down

dev: up seed ## Start the database, seed it, then run the API
	$(MAKE) api

api: ## Run the API server
	go run .

web: ## Run the React dev server
	cd clients && npm start

seed: ## Create permissions, roles, and the owner account (idempotent)
	go run ./cmd/seed

test: ## Run all Go tests (starts its own MySQL via testcontainers)
	go test ./... -count=1

fmt: ## Format all Go source
	gofmt -w $$(find . -type f -name '*.go' -not -path './vendor/*')

gofmtcheck: ## Fail if any Go file is unformatted
	@need_fmt=$$(gofmt -l $$(find . -type f -name '*.go' -not -path './vendor/*'));\
	if [ "$$need_fmt" = "" ]; then echo "hooray"; else echo "files that need formatting:"; echo $$need_fmt; exit 1; fi

lint: gofmtcheck ## Vet and format-check
	go vet ./...

tidy: ## Tidy module dependencies
	go mod tidy
```

- [ ] **Step 9: Verify the whole thing boots**

```bash
cp .env.example .env
sed -i '' "s|^SESSION_SECRET=.*|SESSION_SECRET=$(openssl rand -hex 32)|" .env
make up
make test
```
Expected: containers healthy, `go build ./...` clean, config and testutil tests pass.

- [ ] **Step 10: Commit**

```bash
git add docker-compose.yml Makefile database/db.go main.go internal/testutil go.mod go.sum
git commit -m "feat(dev): docker-compose database, testcontainers harness, real Makefile

database.Connect now takes a DSN and returns an error instead of hardcoding
credentials and panicking. main fails fast on bad config. make test spins its
own MySQL so tests need no manual setup.

Refs ASSESSMENT 6.2."
```

---

## Task 3: Fix the password bug (§4a — CRITICAL)

`models/user.go:21` hashes the literal `"test"` instead of the supplied password, so every account's password is `test`. This is a total authentication bypass.

**Files:**
- Create: `internal/auth/password.go`
- Create: `internal/auth/password_test.go`
- Modify: `models/user.go`

**Interfaces:**
- Consumes: nothing.
- Produces: `auth.HashPassword(plain string) (string, error)` and `auth.CheckPassword(hash, plain string) error`. `models.User.SetPassword(plain string) error` now returns an error.

- [ ] **Step 1: Write the failing test**

Create `internal/auth/password_test.go`:

```go
package auth

import "testing"

func TestHashPasswordThenCheckSucceeds(t *testing.T) {
	hash, err := HashPassword("correct-horse-battery-staple")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if err := CheckPassword(hash, "correct-horse-battery-staple"); err != nil {
		t.Fatalf("the correct password must verify, got: %v", err)
	}
}

// This is the regression test for the defect. Before the fix, SetPassword
// hashed the literal "test", so EVERY wrong password that happened to be
// "test" verified, and the real password did not.
func TestCheckPasswordRejectsWrongPassword(t *testing.T) {
	hash, err := HashPassword("correct-horse-battery-staple")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if err := CheckPassword(hash, "test"); err == nil {
		t.Fatal("the literal \"test\" must NOT verify against a different password")
	}
	if err := CheckPassword(hash, "wrong"); err == nil {
		t.Fatal("a wrong password must not verify")
	}
}

func TestHashPasswordRejectsEmpty(t *testing.T) {
	if _, err := HashPassword(""); err == nil {
		t.Fatal("an empty password must be rejected")
	}
}

func TestHashPasswordIsSalted(t *testing.T) {
	a, _ := HashPassword("same-password")
	b, _ := HashPassword("same-password")
	if a == b {
		t.Fatal("two hashes of the same password must differ (bcrypt salts each)")
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/auth/ -v`
Expected: FAIL — `undefined: HashPassword`.

- [ ] **Step 3: Implement**

Create `internal/auth/password.go`:

```go
// Package auth holds password hashing. It is deliberately separate from the
// models package so it can be tested without a database — the original code
// buried this logic in models.User, which is why the bug below survived.
package auth

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

// cost 12 is ~250ms on current hardware: slow enough to resist offline
// cracking, fast enough not to be a login DoS vector. The original code used
// 14, which is ~1s per login.
const cost = 12

var ErrEmptyPassword = errors.New("password must not be empty")

func HashPassword(plain string) (string, error) {
	if plain == "" {
		return "", ErrEmptyPassword
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(plain), cost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

func CheckPassword(hash, plain string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/auth/ -v`
Expected: PASS — all four tests.

- [ ] **Step 5: Rewire the model**

In `models/user.go`, replace `SetPassword` and `CompareHashAndPassword`:

```go
// SetPassword hashes plain and stores it. It returns an error; the previous
// implementation discarded bcrypt's error AND ignored its argument, hashing
// the literal string "test" for every user (ASSESSMENT 4a).
func (user *User) SetPassword(plain string) error {
	hashed, err := auth.HashPassword(plain)
	if err != nil {
		return err
	}
	user.Password = hashed
	return nil
}

func (user *User) CompareHashAndPassword(plain string) error {
	return auth.CheckPassword(user.Password, plain)
}
```

Add `"github.com/loloDawit/go-admin/internal/auth"` to the imports.

- [ ] **Step 6: Fix every caller to handle the new error**

`controllers/auth_controller.go` (`Register`, `UpdatePassword`) and
`controllers/user_controller.go` (`CreateUser`) all call `SetPassword`.

**The compiler will NOT find these.** A discarded return value in statement
position is legal Go, so `go build` stays green while three call sites
silently ignore the error. Find them by grep instead:

Run: `grep -rn 'SetPassword' --include='*.go' . | grep -v _test.go`
Expected: three call sites. Handle each error; never `_ =`.

- [ ] **Step 7: Also fix `Preload("role")` while in this file (§4m)**

`models/user.go:40` — GORM preloads by *field* name, so the lowercase string silently never loads the association:

```go
db.Preload("Role").Offset(offset).Limit(limit).Find(&users)
```

- [ ] **Step 8: Verify everything builds and passes**

Run: `go build ./... && go test ./... -count=1`
Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add internal/auth models/user.go controllers/
git commit -m "fix(auth): hash the actual password, not the literal \"test\"

models.User.SetPassword called bcrypt.GenerateFromPassword([]byte(\"test\"))
and discarded the error, so every account ever created had the password
\"test\" regardless of what the user typed. Total authentication bypass.

Extracts hashing to internal/auth so it is testable without a database,
lowers cost 14 -> 12, and propagates the error instead of swallowing it.
Also fixes Preload(\"role\") -> Preload(\"Role\"), which silently never
loaded the association on the user list.

Fixes ASSESSMENT 4a, 4m."
```

---

## Task 4: Typed request DTOs and a single error shape

Closes §4x (`map[string]string` request parsing, and the `firstname`/`firstName` casing bug that silently blanks user names on every profile save).

**Files:**
- Create: `internal/httpx/dto.go`
- Create: `internal/httpx/respond.go`
- Create: `controllers/auth_controller_test.go`
- Modify: `controllers/auth_controller.go`

**Interfaces:**
- Consumes: `testutil.NewDB` (Task 2), `auth` (Task 3).
- Produces: `httpx.LoginRequest`, `httpx.UpdateUserInfoRequest`, `httpx.UpdatePasswordRequest`, `httpx.CreateUserRequest`, `httpx.RoleRequest`. (`httpx.Fail` and the `errs` registry already exist — built in Task 3.)

- [ ] **Step 1: Write the failing test**

Create `controllers/auth_controller_test.go`:

```go
package controllers_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/loloDawit/go-admin/internal/testutil"
	"github.com/loloDawit/go-admin/models"
)

// Regression test for the camelCase/lowercase mismatch: Register read
// data["firstName"] but UpdateUserInfo read data["firstname"], and the
// frontend sends camelCase. Saving your profile silently blanked your name.
func TestUpdateUserInfoPreservesCamelCaseNames(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	user := testutil.SeedUser(t, db, "owner@example.com", "s3cret-password", "owner")
	cookie := testutil.Login(t, app, "owner@example.com", "s3cret-password")

	body := `{"firstName":"Ada","lastName":"Lovelace","email":"ada@example.com"}`
	req := testutil.NewRequest(http.MethodPut, "/api/v1/user/info", strings.NewReader(body), cookie)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}

	var reloaded models.User
	db.First(&reloaded, user.Id)

	if reloaded.FirstName != "Ada" {
		t.Errorf("FirstName: want \"Ada\", got %q (the casing bug blanks this)", reloaded.FirstName)
	}
	if reloaded.LastName != "Lovelace" {
		t.Errorf("LastName: want \"Lovelace\", got %q", reloaded.LastName)
	}
}

func TestLoginRejectsWrongPassword(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)
	testutil.SeedUser(t, db, "owner@example.com", "s3cret-password", "owner")

	body := `{"email":"owner@example.com","password":"test"}`
	req := testutil.NewRequest(http.MethodPost, "/api/v1/login", strings.NewReader(body), nil)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if resp.StatusCode == http.StatusOK {
		t.Fatal("login with the literal \"test\" must fail (this passed before the 4a fix)")
	}
}

// Unknown email and wrong password must be indistinguishable, or the API
// becomes a user-enumeration oracle (ASSESSMENT 4i).
func TestLoginDoesNotLeakAccountExistence(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)
	testutil.SeedUser(t, db, "owner@example.com", "s3cret-password", "owner")

	unknown := testutil.NewRequest(http.MethodPost, "/api/v1/login",
		strings.NewReader(`{"email":"nobody@example.com","password":"whatever"}`), nil)
	wrongPass := testutil.NewRequest(http.MethodPost, "/api/v1/login",
		strings.NewReader(`{"email":"owner@example.com","password":"whatever"}`), nil)

	r1, _ := app.Test(unknown, -1)
	r2, _ := app.Test(wrongPass, -1)

	if r1.StatusCode != r2.StatusCode {
		t.Fatalf("status codes must match: unknown=%d wrong-password=%d", r1.StatusCode, r2.StatusCode)
	}

	var b1, b2 map[string]any
	json.NewDecoder(r1.Body).Decode(&b1)
	json.NewDecoder(r2.Body).Decode(&b2)
	if b1["message"] != b2["message"] {
		t.Fatalf("messages must match: %q vs %q", b1["message"], b2["message"])
	}
}

// The JWT must never be returned in the response body — that defeats the
// point of setting it HTTPOnly (ASSESSMENT 4e).
func TestLoginDoesNotReturnTokenInBody(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)
	testutil.SeedUser(t, db, "owner@example.com", "s3cret-password", "owner")

	req := testutil.NewRequest(http.MethodPost, "/api/v1/login",
		strings.NewReader(`{"email":"owner@example.com","password":"s3cret-password"}`), nil)
	resp, _ := app.Test(req, -1)

	var body map[string]any
	json.NewDecoder(resp.Body).Decode(&body)
	if _, present := body["token"]; present {
		t.Fatal("the response body must not contain the token; the cookie is the only channel")
	}
}
```

- [ ] **Step 2: Extend the test harness with the helpers the test needs**

Create `internal/testutil/app.go`:

```go
package testutil

import (
	"io"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/loloDawit/go-admin/internal/auth"
	"github.com/loloDawit/go-admin/internal/config"
	"github.com/loloDawit/go-admin/models"
	"github.com/loloDawit/go-admin/routes"
	"github.com/loloDawit/go-admin/utils"
	"gorm.io/gorm"
)

// TestConfig mirrors a valid production config with test-safe values.
func TestConfig() *config.Config {
	return &config.Config{
		DBDSN:          "unused-in-tests",
		SessionSecret:  "0123456789abcdef0123456789abcdef",
		Port:           "0",
		AllowedOrigin:  "http://localhost:3000",
		PublicBaseURL:  "http://localhost:8080",
		UploadDir:      "./testdata/uploads",
		MaxUploadBytes: 5 << 20,
		AppEnv:         "test",
	}
}

// NewApp builds the real routed application. Tests exercise the same routes
// and middleware chain as production — no test-only handler wiring.
func NewApp(t *testing.T) *fiber.App {
	t.Helper()

	cfg := TestConfig()
	utils.SecretKey = cfg.SessionSecret

	app := fiber.New()
	routes.SetupRoutes(app, cfg)
	return app
}

func NewRequest(method, target string, body io.Reader, cookie *http.Cookie) *http.Request {
	req := httptest.NewRequest(method, target, body)
	req.Header.Set("Content-Type", "application/json")
	if cookie != nil {
		req.AddCookie(cookie)
	}
	return req
}

// SeedUser creates a user with the named role, creating the role if needed.
// The password is hashed through the real hasher so login tests are honest.
func SeedUser(t *testing.T, db *gorm.DB, email, password, roleName string) *models.User {
	t.Helper()

	var role models.Role
	if err := db.Where("name = ?", roleName).FirstOrCreate(&role, models.Role{Name: roleName}).Error; err != nil {
		t.Fatalf("seed role: %v", err)
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}

	user := &models.User{
		FirstName: "Test",
		LastName:  "User",
		Email:     email,
		Password:  hash,
		RoleId:    role.Id,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return user
}

// GrantPermission attaches a permission to a user's role, creating both if
// needed. Used by authorization tests.
func GrantPermission(t *testing.T, db *gorm.DB, roleName, permission string) {
	t.Helper()

	var role models.Role
	if err := db.Where("name = ?", roleName).FirstOrCreate(&role, models.Role{Name: roleName}).Error; err != nil {
		t.Fatalf("role: %v", err)
	}
	var perm models.Permission
	if err := db.Where("name = ?", permission).FirstOrCreate(&perm, models.Permission{Name: permission}).Error; err != nil {
		t.Fatalf("permission: %v", err)
	}
	if err := db.Model(&role).Association("Permissions").Append(&perm); err != nil {
		t.Fatalf("attach permission: %v", err)
	}
}

// Login performs a real login and returns the session cookie.
func Login(t *testing.T, app *fiber.App, email, password string) *http.Cookie {
	t.Helper()

	req := NewRequest(http.MethodPost, "/api/v1/login",
		strings.NewReader(`{"email":"`+email+`","password":"`+password+`"}`), nil)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("login request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login failed with status %d", resp.StatusCode)
	}

	for _, c := range resp.Cookies() {
		if c.Name == "jwt" {
			return c
		}
	}
	t.Fatal("login response carried no jwt cookie")
	return nil
}
```

> Full import block for `internal/testutil/app.go`:
> ```go
> import (
> 	"io"
> 	"net/http"
> 	"net/http/httptest"
> 	"strings"
> 	"testing"
>
> 	"github.com/gofiber/fiber/v2"
> 	"github.com/loloDawit/go-admin/internal/auth"
> 	"github.com/loloDawit/go-admin/internal/config"
> 	"github.com/loloDawit/go-admin/models"
> 	"github.com/loloDawit/go-admin/routes"
> 	"github.com/loloDawit/go-admin/utils"
> 	"gorm.io/gorm"
> )
> ```

- [ ] **Step 3: Run the tests to verify they fail**

Run: `go test ./controllers/ -run 'TestLogin|TestUpdateUserInfo' -v`
Expected: FAIL. `TestUpdateUserInfoPreservesCamelCaseNames` fails on the casing bug; `TestLoginDoesNotReturnTokenInBody` fails because the token is in the body; `TestLoginDoesNotLeakAccountExistence` fails on 404-vs-400.

- [ ] **Step 4: Write the DTOs**

Create `internal/httpx/dto.go`:

```go
// Package httpx holds request DTOs and response helpers. Typed DTOs replace
// the original map[string]string parsing, which had no schema and allowed a
// key-casing mismatch between two handlers to silently blank user names.
package httpx

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UpdateUserInfoRequest struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
}

type UpdatePasswordRequest struct {
	Password        string `json:"password"`
	PasswordConfirm string `json:"passwordConfirm"`
}

type CreateUserRequest struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	RoleId    uint   `json:"roleId"`
}

type UpdateUserRequest struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
	RoleId    uint   `json:"roleId"`
}

// RoleRequest replaces the unchecked fiber.Map type assertions in
// role_controller.go, which panicked whenever a client sent permission IDs
// as JSON numbers instead of strings (ASSESSMENT 4n).
type RoleRequest struct {
	Name        string `json:"name"`
	Permissions []uint `json:"permissions"`
}

type ProductRequest struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Image       string  `json:"image"`
	Price       float64 `json:"price"`
}
```

- [ ] **Step 5: Write the response helper**

Create `internal/httpx/respond.go`:

```go
package httpx

import "github.com/gofiber/fiber/v2"

// ErrorBody is the single error shape for the whole API. The original code
// returned {"error": ...}, {"msg": ...}, and raw GORM error objects from
// different handlers; clients could not parse errors reliably, and one path
// leaked driver internals to the caller.
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func Error(ctx *fiber.Ctx, status int, code, message string) error {
	return ctx.Status(status).JSON(ErrorBody{Code: code, Message: message})
}
```

- [ ] **Step 6: Rewrite the auth handlers**

In `controllers/auth_controller.go`: delete `Register` entirely (per ADR 0001 — Task 6 removes its route), and replace the remaining handlers:

```go
// Login is a closure over config so the cookie's Secure flag comes from
// configuration rather than a stray os.Getenv — config.Load is the only
// place this application reads the environment.
func Login(cfg *config.Config) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
	var req httpx.LoginRequest
	if err := ctx.BodyParser(&req); err != nil {
		return httpx.Fail(ctx, errs.InvalidBody)
	}

	var user models.User
	result := database.DB.Where("email = ?", req.Email).First(&user)

	// Unknown email and wrong password return an identical response, so the
	// endpoint cannot be used to enumerate accounts (ASSESSMENT 4i).
	// errs.InvalidCredentials is deliberately the SAME error for both cases.
	if result.Error != nil || user.Id == 0 {
		return httpx.Fail(ctx, errs.InvalidCredentials)
	}
	if err := user.CompareHashAndPassword(req.Password); err != nil {
		return httpx.Fail(ctx, errs.InvalidCredentials)
	}

	token, err := utils.GenerateJWT(strconv.Itoa(user.Id))
	if err != nil {
		return httpx.Fail(ctx, errs.TokenIssueFailed)
	}

	ctx.Cookie(&fiber.Cookie{
		Name:     "jwt",
		Value:    token,
		Expires:  time.Now().Add(24 * time.Hour),
		HTTPOnly: true,
		SameSite: "Lax",
		Secure:   cfg.IsProduction(),
		Path:     "/",
	})

	// The token is deliberately NOT in the body. The HTTPOnly cookie is the
	// only channel; returning it here would hand it to any XSS on the page.
	return ctx.JSON(fiber.Map{"message": "ok"})
	}
}

func UpdateUserInfo(ctx *fiber.Ctx) error {
	var req httpx.UpdateUserInfoRequest
	if err := ctx.BodyParser(&req); err != nil {
		return httpx.Fail(ctx, errs.InvalidBody)
	}

	userId, err := currentUserId(ctx)
	if err != nil {
		return httpx.Fail(ctx, errs.Unauthenticated)
	}

	// Updates with a map, not a struct: GORM's struct form skips zero values,
	// so a struct could never clear a field (ASSESSMENT 4p).
	result := database.DB.Model(&models.User{Id: userId}).Updates(map[string]any{
		"first_name": req.FirstName,
		"last_name":  req.LastName,
		"email":      req.Email,
	})
	if result.Error != nil {
		return httpx.Fail(ctx, errs.Database.Wrap(err))
	}

	var user models.User
	database.DB.First(&user, userId)
	return ctx.JSON(user)
}

func UpdatePassword(ctx *fiber.Ctx) error {
	var req httpx.UpdatePasswordRequest
	if err := ctx.BodyParser(&req); err != nil {
		return httpx.Fail(ctx, errs.InvalidBody)
	}
	if req.Password != req.PasswordConfirm {
		return httpx.Fail(ctx, errs.PasswordMismatch)
	}

	userId, err := currentUserId(ctx)
	if err != nil {
		return httpx.Fail(ctx, errs.Unauthenticated)
	}

	var user models.User
	if err := user.SetPassword(req.Password); err != nil {
		return httpx.Fail(ctx, errs.PasswordEmpty)
	}

	if err := database.DB.Model(&models.User{Id: userId}).
		Update("password", user.Password).Error; err != nil {
		return httpx.Fail(ctx, errs.Database.Wrap(err))
	}

	return ctx.JSON(fiber.Map{"message": "ok"})
}

// currentUserId reads the authenticated user's id from the session cookie.
// Replaces four copies of the same cookie-parse-and-ignore-errors block.
func currentUserId(ctx *fiber.Ctx) (int, error) {
	issuer, err := utils.ParseJWT(ctx.Cookies("jwt"))
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(issuer)
}
```

- [ ] **Step 7: Run the tests to verify they pass**

Run: `go test ./controllers/ -run 'TestLogin|TestUpdateUserInfo' -v`
Expected: PASS — all four.

- [ ] **Step 8: Commit**

```bash
git add internal/httpx internal/testutil controllers/
git commit -m "fix(api): typed request DTOs and one error shape

Replaces map[string]string parsing in the auth handlers. This fixes the
firstName/firstname key mismatch between Register and UpdateUserInfo that
silently blanked a user's name on every profile save.

Also: login no longer returns the token in the body (the HTTPOnly cookie is
the only channel), unknown-email and wrong-password now return identical
responses so the endpoint cannot enumerate accounts, and the session cookie
gains SameSite=Lax plus Secure in production.

Fixes ASSESSMENT 4x, 4i, 4e (partial)."
```

---

## Task 5: Fix pagination arithmetic (§4l)

`models/pagination.go:17` computes `math.Ceil(float64(int(total) / limit))` — integer division happens first, so `Ceil` is a no-op and the final partial page is unreachable.

**Files:**
- Create: `models/pagination_test.go`
- Modify: `models/pagination.go`

**Interfaces:**
- Produces: `models.Paginate(db *gorm.DB, entity Entity, page, perPage int) fiber.Map`. Note the new `perPage` parameter — every caller in `controllers/` must be updated.

- [ ] **Step 1: Write the failing test**

Create `models/pagination_test.go`:

```go
package models

import "testing"

func TestLastPageRoundsUpOnPartialFinalPage(t *testing.T) {
	cases := []struct {
		total, perPage, want int
	}{
		{total: 12, perPage: 5, want: 3}, // the bug: integer division gives 2
		{total: 10, perPage: 5, want: 2},
		{total: 1, perPage: 5, want: 1},
		{total: 0, perPage: 5, want: 1}, // an empty list still has one page
		{total: 26, perPage: 25, want: 2},
	}

	for _, c := range cases {
		if got := lastPage(int64(c.total), c.perPage); got != c.want {
			t.Errorf("lastPage(%d, %d) = %d, want %d", c.total, c.perPage, got, c.want)
		}
	}
}

func TestNormalizePerPageIsBounded(t *testing.T) {
	cases := []struct{ in, want int }{
		{in: 0, want: defaultPerPage},
		{in: -5, want: defaultPerPage},
		{in: 10, want: 10},
		{in: 5000, want: maxPerPage}, // must not let a client request everything
	}
	for _, c := range cases {
		if got := normalizePerPage(c.in); got != c.want {
			t.Errorf("normalizePerPage(%d) = %d, want %d", c.in, got, c.want)
		}
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./models/ -run 'TestLastPage|TestNormalize' -v`
Expected: FAIL — `undefined: lastPage`.

- [ ] **Step 3: Implement**

Replace `models/pagination.go`:

```go
package models

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

const (
	defaultPerPage = 25
	maxPerPage     = 100
)

// lastPage returns the 1-based index of the final page.
//
// The original was math.Ceil(float64(int(total) / limit)), which performs
// INTEGER division before converting to float — so Ceil always received a
// whole number and did nothing. With 12 rows at 5 per page it reported 2
// pages instead of 3, making the last two records unreachable in any UI
// that trusted the value (ASSESSMENT 4l).
func lastPage(total int64, perPage int) int {
	if total <= 0 {
		return 1
	}
	pages := int((total + int64(perPage) - 1) / int64(perPage))
	if pages < 1 {
		return 1
	}
	return pages
}

// normalizePerPage clamps a client-supplied page size. Unbounded page sizes
// let one request pull the entire table.
func normalizePerPage(perPage int) int {
	switch {
	case perPage <= 0:
		return defaultPerPage
	case perPage > maxPerPage:
		return maxPerPage
	default:
		return perPage
	}
}

func normalizePage(page int) int {
	if page < 1 {
		return 1
	}
	return page
}

func Paginate(db *gorm.DB, entity Entity, page, perPage int) fiber.Map {
	page = normalizePage(page)
	perPage = normalizePerPage(perPage)
	offset := (page - 1) * perPage

	data := entity.Take(db, perPage, offset)
	total := entity.Count(db)

	return fiber.Map{
		"data": data,
		"meta": fiber.Map{
			"total":     total,
			"page":      page,
			"perPage":   perPage,
			"last_page": lastPage(total, perPage),
		},
	}
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./models/ -run 'TestLastPage|TestNormalize' -v`
Expected: PASS.

- [ ] **Step 5: Update the three callers**

In `controllers/user_controller.go`, `product_controller.go`, and `order_controller.go`, replace each `models.Paginate(database.DB, &models.X{}, page)` with:

```go
	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	perPage, _ := strconv.Atoi(ctx.Query("perPage", "0")) // 0 -> default
	return ctx.JSON(models.Paginate(database.DB, &models.Product{}, page, perPage))
```

- [ ] **Step 6: Verify the build and full suite**

Run: `go build ./... && go test ./... -count=1`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add models/pagination.go models/pagination_test.go controllers/
git commit -m "fix(pagination): ceil after division, and bound the page size

math.Ceil(float64(int(total)/limit)) divided as integers first, so Ceil was
a no-op and the final partial page was unreachable — 12 records at 5 per
page reported 2 pages instead of 3.

Page size moves from a hardcoded 5 to a client-supplied value clamped to
[1, 100] with a default of 25.

Fixes ASSESSMENT 4l."
```

---

## Task 6: Invite-only access — remove registration, add the seed command

Implements ADR 0001. Closes §4c (public registration grants admin) and §6.2 (no bootstrap path).

**Read `docs/decisions/0001-remove-public-registration.md` before starting.**

**Files:**
- Create: `cmd/seed/main.go`
- Create: `internal/seed/seed.go`
- Create: `internal/seed/seed_test.go`
- Modify: `controllers/auth_controller.go` (delete `Register` — done in Task 4)
- Modify: `controllers/user_controller.go` (`CreateUser` takes an admin-supplied password)
- Delete: `clients/src/pages/Register.tsx`, `clients/src/pages/Register.css`

**Interfaces:**
- Produces: `seed.Run(db *gorm.DB, ownerEmail, ownerPassword string) error` — idempotent.
- Permission vocabulary (referenced by Task 7's middleware): `view_users`, `edit_users`, `view_products`, `edit_products`, `view_orders`, `edit_orders`, `view_roles`, `edit_roles`.
- Roles: `owner` (all permissions), `admin` (all but `edit_roles`), `staff` (all `view_*` plus `edit_products`, `edit_orders`).

- [ ] **Step 1: Write the failing test**

Create `internal/seed/seed_test.go`:

```go
package seed_test

import (
	"testing"

	"github.com/loloDawit/go-admin/internal/auth"
	"github.com/loloDawit/go-admin/internal/seed"
	"github.com/loloDawit/go-admin/internal/testutil"
	"github.com/loloDawit/go-admin/models"
)

func TestRunCreatesPermissionsRolesAndOwner(t *testing.T) {
	db := testutil.NewDB(t)

	if err := seed.Run(db, "owner@example.com", "s3cret-password"); err != nil {
		t.Fatalf("seed: %v", err)
	}

	var permCount int64
	db.Model(&models.Permission{}).Count(&permCount)
	if permCount != 8 {
		t.Errorf("want 8 permissions, got %d", permCount)
	}

	var owner models.Role
	if err := db.Preload("Permissions").Where("name = ?", "owner").First(&owner).Error; err != nil {
		t.Fatalf("owner role missing: %v", err)
	}
	if len(owner.Permissions) != 8 {
		t.Errorf("owner must hold all 8 permissions, got %d", len(owner.Permissions))
	}

	var user models.User
	if err := db.Where("email = ?", "owner@example.com").First(&user).Error; err != nil {
		t.Fatalf("owner user missing: %v", err)
	}
	if user.RoleId != owner.Id {
		t.Errorf("owner user must hold the owner role")
	}
	if err := auth.CheckPassword(user.Password, "s3cret-password"); err != nil {
		t.Errorf("owner password must verify: %v", err)
	}
}

func TestRunIsIdempotent(t *testing.T) {
	db := testutil.NewDB(t)

	for i := 0; i < 3; i++ {
		if err := seed.Run(db, "owner@example.com", "s3cret-password"); err != nil {
			t.Fatalf("seed run %d: %v", i, err)
		}
	}

	var users, perms, roles int64
	db.Model(&models.User{}).Count(&users)
	db.Model(&models.Permission{}).Count(&perms)
	db.Model(&models.Role{}).Count(&roles)

	if users != 1 || perms != 8 || roles != 3 {
		t.Fatalf("re-running seed must not duplicate: users=%d perms=%d roles=%d", users, perms, roles)
	}
}

func TestRunRejectsEmptyOwnerCredentials(t *testing.T) {
	db := testutil.NewDB(t)

	if err := seed.Run(db, "", "s3cret-password"); err == nil {
		t.Error("an empty owner email must be rejected")
	}
	if err := seed.Run(db, "owner@example.com", ""); err == nil {
		t.Error("an empty owner password must be rejected")
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/seed/ -v`
Expected: FAIL — `undefined: seed.Run`.

- [ ] **Step 3: Implement the seeder**

Create `internal/seed/seed.go`:

```go
// Package seed creates the permission vocabulary, the built-in roles, and the
// first owner account. It exists because the application previously had no
// bootstrap path at all: registration hardcoded RoleId 1, nothing created
// role 1, and so a clean database could not produce a usable account.
//
// See docs/decisions/0001-remove-public-registration.md.
package seed

import (
	"errors"
	"fmt"

	"github.com/loloDawit/go-admin/internal/auth"
	"github.com/loloDawit/go-admin/models"
	"gorm.io/gorm"
)

// The authorization middleware derives its checks from these names.
var AllPermissions = []string{
	"view_users", "edit_users",
	"view_products", "edit_products",
	"view_orders", "edit_orders",
	"view_roles", "edit_roles",
}

// roleGrants maps each built-in role to the permissions it holds.
var roleGrants = map[string][]string{
	"owner": AllPermissions,
	"admin": {
		"view_users", "edit_users",
		"view_products", "edit_products",
		"view_orders", "edit_orders",
		"view_roles",
	},
	"staff": {
		"view_users",
		"view_products", "edit_products",
		"view_orders", "edit_orders",
	},
}

// Idempotent: safe to run on every deploy.
func Run(db *gorm.DB, ownerEmail, ownerPassword string) error {
	if ownerEmail == "" {
		return errors.New("owner email is required (set OWNER_EMAIL)")
	}
	if ownerPassword == "" {
		return errors.New("owner password is required (set OWNER_PASSWORD)")
	}

	return db.Transaction(func(tx *gorm.DB) error {
		permissions := make(map[string]models.Permission, len(AllPermissions))
		for _, name := range AllPermissions {
			var p models.Permission
			if err := tx.Where(models.Permission{Name: name}).
				FirstOrCreate(&p, models.Permission{Name: name}).Error; err != nil {
				return fmt.Errorf("permission %q: %w", name, err)
			}
			permissions[name] = p
		}

		roles := make(map[string]models.Role, len(roleGrants))
		for roleName, granted := range roleGrants {
			var role models.Role
			if err := tx.Where(models.Role{Name: roleName}).
				FirstOrCreate(&role, models.Role{Name: roleName}).Error; err != nil {
				return fmt.Errorf("role %q: %w", roleName, err)
			}

			attach := make([]models.Permission, 0, len(granted))
			for _, name := range granted {
				attach = append(attach, permissions[name])
			}
			// Replace, not Append: re-running must converge, not accumulate.
			if err := tx.Model(&role).Association("Permissions").Replace(attach); err != nil {
				return fmt.Errorf("grant permissions to %q: %w", roleName, err)
			}
			roles[roleName] = role
		}

		var existing models.User
		err := tx.Where("email = ?", ownerEmail).First(&existing).Error
		if err == nil {
			// Never reset an existing owner's password: that would make every
			// deploy an account takeover.
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("look up owner: %w", err)
		}

		hash, err := auth.HashPassword(ownerPassword)
		if err != nil {
			return fmt.Errorf("hash owner password: %w", err)
		}

		owner := models.User{
			FirstName: "Owner",
			LastName:  "Account",
			Email:     ownerEmail,
			Password:  hash,
			RoleId:    roles["owner"].Id,
		}
		if err := tx.Create(&owner).Error; err != nil {
			return fmt.Errorf("create owner: %w", err)
		}
		return nil
	})
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/seed/ -v`
Expected: PASS — all three tests.

- [ ] **Step 5: Write the CLI entrypoint**

Create `cmd/seed/main.go`:

```go
// Command seed bootstraps a database with the permission vocabulary, the
// built-in roles, and the owner account. It is idempotent — safe to run on
// every deploy.
//
//	OWNER_EMAIL=you@example.com OWNER_PASSWORD=... go run ./cmd/seed
package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/loloDawit/go-admin/database"
	"github.com/loloDawit/go-admin/internal/config"
	"github.com/loloDawit/go-admin/internal/seed"
)

func main() {
	if err := godotenv.Load(".env"); err != nil && !os.IsNotExist(err) {
		log.Printf("config: .env not loaded (%v); reading environment directly", err)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	db, err := database.Connect(cfg.DBDSN)
	if err != nil {
		log.Fatalf("database: %v", err)
	}

	ownerEmail := os.Getenv("OWNER_EMAIL")
	ownerPassword := os.Getenv("OWNER_PASSWORD")

	if err := seed.Run(db, ownerEmail, ownerPassword); err != nil {
		log.Fatalf("seed: %v", err)
	}

	log.Printf("seeded: permissions, roles, and owner %s", ownerEmail)
	log.Print("change the owner password after first sign-in, then remove OWNER_PASSWORD from .env")
}
```

- [ ] **Step 6: Make `CreateUser` take an admin-supplied password**

In `controllers/user_controller.go`, replace `CreateUser`. Note `user.SetPassword("124")` in the original — a hardcoded password, which combined with §4a meant the value was ignored anyway.

```go
func CreateUser(ctx *fiber.Ctx) error {
	var req httpx.CreateUserRequest
	if err := ctx.BodyParser(&req); err != nil {
		return httpx.Fail(ctx, errs.InvalidBody)
	}

	user := models.User{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     req.Email,
		RoleId:    req.RoleId,
	}

	if err := user.Validate(); err != nil {
		return httpx.Fail(ctx, errs.ValidationFailed.WithMessage("%s", err))
	}
	if err := user.SetPassword(cfg.Hasher(), req.Password); err != nil {
		return httpx.Fail(ctx, err) // already an errs value
	}

	if err := database.DB.Create(&user).Error; err != nil {
		return httpx.Fail(ctx, errs.EmailTaken)
	}

	return ctx.Status(fiber.StatusCreated).JSON(user)
}
```

> `Validate` drops its unused `action string` parameter (ASSESSMENT §4z) **and**
> its password check, because `CreateUser` now validates the password
> separately via `SetPassword`. Replace it in `models/user.go` with:
>
> ```go
> // Validate checks the fields a User carries. It deliberately does NOT check
> // Password: the plaintext never reaches this struct (SetPassword stores only
> // the hash), so a password check here could only ever inspect a bcrypt digest.
> func (user *User) Validate() error {
> 	if user.FirstName == "" || user.LastName == "" {
> 		return errors.New("first and last name are required")
> 	}
> 	if user.Email == "" {
> 		return errors.New("email is required")
> 	}
> 	if err := checkmail.ValidateFormat(user.Email); err != nil {
> 		return errors.New("invalid email")
> 	}
> 	if user.RoleId == 0 {
> 		return errors.New("a role is required")
> 	}
> 	return nil
> }
> ```
>
> The only other caller was `Register`, which Task 4 deleted, so no other
> call site needs updating.

- [ ] **Step 7: Delete the frontend registration page**

```bash
git rm clients/src/pages/Register.tsx clients/src/pages/Register.css
```

In `clients/src/App.tsx`, remove the `Register` import and the `<Route path="/register" ... />` line.

- [ ] **Step 8: Verify build and full suite**

Run: `go build ./... && go test ./... -count=1 && cd clients && npx tsc --noEmit`
Expected: PASS; TypeScript reports no missing-module error for `Register`.

- [ ] **Step 9: Commit**

```bash
git add -A
git commit -m "feat(auth): invite-only access; seed the owner, drop public registration

Public POST /register hardcoded RoleId 1 (the admin role), so anyone who
could reach the API could mint an administrator. Nothing ever created role 1
either, so a clean database could not produce a working account at all.

Removes the endpoint and its page. The owner is now created by an idempotent
`make seed`; all other staff are created by an admin via POST /users with an
admin-supplied initial password.

Implements docs/decisions/0001-remove-public-registration.md.
Fixes ASSESSMENT 4c, 6.2."
```

---

## Task 7: Authorization as route-group middleware (§4b — CRITICAL)

`IsAuthorized` is called from five places, all in `user_controller.go`. Products, orders, roles, permissions, and upload have **no authorization at all** — any authenticated user can rewrite the catalog and grant themselves any permission.

**Files:**
- Create: `middlewares/permission_middleware_test.go`
- Modify: `middlewares/permission_middleware.go`
- Modify: `routes/routes.go`
- Modify: `controllers/user_controller.go` (remove the five manual calls)

**Interfaces:**
- Consumes: the permission vocabulary from Task 6.
- Produces: `middlewares.RequirePermission(resource string) fiber.Handler`.

- [ ] **Step 1: Write the failing test**

Create `middlewares/permission_middleware_test.go`:

```go
package middlewares_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/loloDawit/go-admin/internal/testutil"
)

// The headline regression test: before this task, a user with NO permissions
// could create products, because no product route consulted IsAuthorized.
func TestWriteRoutesRejectUserWithoutEditPermission(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	testutil.SeedUser(t, db, "viewer@example.com", "s3cret-password", "viewer")
	testutil.GrantPermission(t, db, "viewer", "view_products")
	cookie := testutil.Login(t, app, "viewer@example.com", "s3cret-password")

	writes := []struct {
		method, path, body string
	}{
		{http.MethodPost, "/api/v1/products", `{"title":"Pwned","price":1}`},
		{http.MethodPut, "/api/v1/product/1", `{"title":"Pwned"}`},
		{http.MethodDelete, "/api/v1/product/1", ``},
		{http.MethodPost, "/api/v1/orders", `{"email":"x@example.com"}`},
		{http.MethodPost, "/api/v1/roles", `{"name":"superuser","permissions":[1]}`},
		{http.MethodPost, "/api/v1/permissions", `{"name":"edit_everything"}`},
	}

	for _, w := range writes {
		req := testutil.NewRequest(w.method, w.path, strings.NewReader(w.body), cookie)
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("%s %s: %v", w.method, w.path, err)
		}
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("%s %s: want 403, got %d (unauthorized write is possible)",
				w.method, w.path, resp.StatusCode)
		}
	}
}

func TestReadRouteAllowsViewPermission(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	testutil.SeedUser(t, db, "viewer@example.com", "s3cret-password", "viewer")
	testutil.GrantPermission(t, db, "viewer", "view_products")
	cookie := testutil.Login(t, app, "viewer@example.com", "s3cret-password")

	req := testutil.NewRequest(http.MethodGet, "/api/v1/products", nil, cookie)
	resp, _ := app.Test(req, -1)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("view_products must permit GET /products, got %d", resp.StatusCode)
	}
}

func TestEditPermissionImpliesRead(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	testutil.SeedUser(t, db, "editor@example.com", "s3cret-password", "editor")
	testutil.GrantPermission(t, db, "editor", "edit_products")
	cookie := testutil.Login(t, app, "editor@example.com", "s3cret-password")

	req := testutil.NewRequest(http.MethodGet, "/api/v1/products", nil, cookie)
	resp, _ := app.Test(req, -1)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("edit_products must imply read access, got %d", resp.StatusCode)
	}
}

func TestUnauthenticatedRequestIsRejected(t *testing.T) {
	testutil.NewDB(t)
	app := testutil.NewApp(t)

	req := testutil.NewRequest(http.MethodGet, "/api/v1/products", nil, nil)
	resp, _ := app.Test(req, -1)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("want 401 without a cookie, got %d", resp.StatusCode)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./middlewares/ -v`
Expected: FAIL — `TestWriteRoutesRejectUserWithoutEditPermission` reports 200s where it wants 403s, proving the missing authorization.

- [ ] **Step 3: Rewrite the middleware**

Replace `middlewares/permission_middleware.go`:

```go
package middlewares

import (
	"github.com/gofiber/fiber/v2"
	"github.com/loloDawit/go-admin/database"
	"github.com/loloDawit/go-admin/internal/httpx"
	"github.com/loloDawit/go-admin/models"
	"github.com/loloDawit/go-admin/utils"
	"strconv"
)

// RequirePermission enforces the view_/edit_ convention: safe methods accept
// either, mutating methods require edit_. Applied to route groups so
// authorization is the default rather than opt-in.
func RequirePermission(resource string) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		issuer, err := utils.ParseJWT(ctx.Cookies("jwt"))
		if err != nil {
			return httpx.Fail(ctx, errs.Unauthenticated)
		}

		userId, err := strconv.Atoi(issuer)
		if err != nil {
			return httpx.Fail(ctx, errs.Unauthenticated)
		}

		var user models.User
		if err := database.DB.First(&user, userId).Error; err != nil {
			return httpx.Fail(ctx, errs.Unauthenticated)
		}

		var role models.Role
		if err := database.DB.Preload("Permissions").First(&role, user.RoleId).Error; err != nil {
			return httpx.Fail(ctx, errs.Forbidden)
		}

		required := "edit_" + resource
		alsoAccepted := ""
		if isSafeMethod(ctx.Method()) {
			alsoAccepted = "view_" + resource
		}

		for _, p := range role.Permissions {
			if p.Name == required || (alsoAccepted != "" && p.Name == alsoAccepted) {
				ctx.Locals("userId", userId)
				return ctx.Next()
			}
		}

		return httpx.Fail(ctx, errs.Forbidden)
	}
}

func isSafeMethod(method string) bool {
	return method == fiber.MethodGet || method == fiber.MethodHead || method == fiber.MethodOptions
}
```

> Note the removal of `fmt.Println(permission.Name, page)` from the hot path (§4t), and of the second redundant user query.

- [ ] **Step 4: Rewrite the routes with groups**

Replace `routes/routes.go`:

```go
package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/loloDawit/go-admin/controllers"
	"github.com/loloDawit/go-admin/internal/config"
	"github.com/loloDawit/go-admin/middlewares"
)

func SetupRoutes(app *fiber.App, cfg *config.Config) {
	api := app.Group("/api/v1")

	// --- Public ---
	// No /register by design; see docs/decisions/0001-remove-public-registration.md.
	api.Post("/login", controllers.Login(cfg))

	// --- Authenticated ---
	authed := api.Group("", middlewares.IsAuthenticated)

	// Self-service: any signed-in user may read and edit their own account.
	// These need no resource permission — the handler scopes to the caller.
	authed.Get("/user", controllers.User)
	authed.Post("/logout", controllers.Logout)
	authed.Put("/user/info", controllers.UpdateUserInfo)
	authed.Put("/user/password", controllers.UpdatePassword)

	// --- Permission-gated resources ---
	// A route added to one of these groups inherits its check.
	users := authed.Group("", middlewares.RequirePermission("users"))
	users.Get("/users", controllers.GetAllUsers)
	users.Post("/users", controllers.CreateUser)
	users.Get("/user/:id", controllers.GetUser)
	users.Put("/user/:id", controllers.UpdateUser)
	users.Delete("/user/:id", controllers.DeleteUser)

	products := authed.Group("", middlewares.RequirePermission("products"))
	products.Get("/products", controllers.GetAllProducts)
	products.Post("/products", controllers.CreateProduct)
	products.Get("/product/:id", controllers.GetProduct)
	products.Put("/product/:id", controllers.UpdateProduct)
	products.Delete("/product/:id", controllers.DeleteProduct)
	products.Post("/upload", controllers.Upload) // becomes Upload(cfg) in Task 10

	orders := authed.Group("", middlewares.RequirePermission("orders"))
	orders.Get("/orders", controllers.GetAllOrders)
	orders.Post("/orders", controllers.CreateOrder)
	orders.Get("/order/:id", controllers.GetOrder)
	orders.Put("/order/:id", controllers.UpdateOrder)
	orders.Delete("/order/:id", controllers.DeleteOrder)
	orders.Get("/export", controllers.Export)  // was POST; a download is a GET
	orders.Get("/chart", controllers.Chart)

	roles := authed.Group("", middlewares.RequirePermission("roles"))
	roles.Get("/roles", controllers.GetAllRoles)
	roles.Post("/roles", controllers.CreateRole)
	roles.Get("/role/:id", controllers.GetRole)
	roles.Put("/role/:id", controllers.UpdateRole)
	roles.Delete("/role/:id", controllers.DeleteRole)
	roles.Get("/permissions", controllers.GetAllPermissions)
	roles.Post("/permissions", controllers.CreatePermission)

	// Uploaded files. Served from the configured directory rather than a
	// hardcoded path.
	app.Static("/api/v1/uploads", cfg.UploadDir)
}
```

- [ ] **Step 5: Remove the five manual `IsAuthorized` calls**

In `controllers/user_controller.go`, delete every `if err := middlewares.IsAuthorized(ctx, "users"); err != nil { return err }` block (lines 13, 21, 36, 55, 72 in the original) and the now-unused `middlewares` import. The route group enforces this now; leaving both would double-query on every request.

- [ ] **Step 6: Fix the `CreatPermissons` typo while here**

Rename `controllers.CreatPermissons` → `controllers.CreatePermission` in `controllers/permission_controller.go`.

- [ ] **Step 7: Run to verify it passes**

Run: `go test ./middlewares/ -v`
Expected: PASS — all four tests.

- [ ] **Step 8: Run the full suite**

Run: `go test ./... -count=1`
Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add middlewares/ routes/ controllers/
git commit -m "fix(authz): enforce permissions on every resource route

IsAuthorized was a function each handler had to remember to call, and only
user_controller.go ever called it. Products, orders, roles, permissions, and
upload were completely unguarded: any authenticated user could rewrite the
catalog, edit any order, and grant themselves any permission via POST /roles.

Authorization is now route-group middleware, so it is the default rather than
an opt-in. Also removes the fmt.Println from the authorization hot path, drops
a redundant per-request user query, and moves /export from POST to GET.

Fixes ASSESSMENT 4b, 4t."
```

---

## Task 8: Mass assignment, missing records, and unchecked database errors

Closes §4g (client-supplied `id`/`roleId` override the path parameter), §4o (write errors reported as success; missing records return 200), and §4q (ignored `Atoi` errors).

**Files:**
- Create: `controllers/product_controller_test.go`
- Modify: `controllers/product_controller.go`, `user_controller.go`, `order_controller.go`, `role_controller.go`

**Interfaces:**
- Produces: `controllers.pathId(ctx) (int, error)` — one shared, validated path-parameter parser.

- [ ] **Step 1: Write the failing test**

Create `controllers/product_controller_test.go`:

```go
package controllers_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/loloDawit/go-admin/internal/testutil"
	"github.com/loloDawit/go-admin/models"
)

// Handlers built a struct with the path id preset, then BodyParser'd into the
// SAME struct — so an "id" in the body silently overwrote the path parameter.
func TestUpdateProductIgnoresIdInBody(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	testutil.SeedUser(t, db, "editor@example.com", "s3cret-password", "editor")
	testutil.GrantPermission(t, db, "editor", "edit_products")
	cookie := testutil.Login(t, app, "editor@example.com", "s3cret-password")

	target := models.Product{Title: "Target", Price: 10}
	victim := models.Product{Title: "Victim", Price: 20}
	db.Create(&target)
	db.Create(&victim)

	// Update `target` by path, but claim `victim`'s id in the body.
	body := `{"id":` + itoa(int(victim.Id)) + `,"title":"Hijacked","price":99}`
	req := testutil.NewRequest(http.MethodPut, "/api/v1/product/"+itoa(int(target.Id)),
		strings.NewReader(body), cookie)

	if _, err := app.Test(req, -1); err != nil {
		t.Fatalf("request: %v", err)
	}

	var reloadedVictim models.Product
	db.First(&reloadedVictim, victim.Id)
	if reloadedVictim.Title != "Victim" {
		t.Errorf("the body's id must not redirect the update; victim became %q", reloadedVictim.Title)
	}

	var reloadedTarget models.Product
	db.First(&reloadedTarget, target.Id)
	if reloadedTarget.Title != "Hijacked" {
		t.Errorf("the path id must decide the target; got %q", reloadedTarget.Title)
	}
}

func TestGetMissingProductReturns404(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	testutil.SeedUser(t, db, "viewer@example.com", "s3cret-password", "viewer")
	testutil.GrantPermission(t, db, "viewer", "view_products")
	cookie := testutil.Login(t, app, "viewer@example.com", "s3cret-password")

	req := testutil.NewRequest(http.MethodGet, "/api/v1/product/999999", nil, cookie)
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("want 404 for a missing product, got %d (Find returns an empty struct with 200)",
			resp.StatusCode)
	}
}

func TestNonNumericIdReturns400(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	testutil.SeedUser(t, db, "viewer@example.com", "s3cret-password", "viewer")
	testutil.GrantPermission(t, db, "viewer", "view_products")
	cookie := testutil.Login(t, app, "viewer@example.com", "s3cret-password")

	req := testutil.NewRequest(http.MethodGet, "/api/v1/product/abc", nil, cookie)
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("want 400 for a non-numeric id, got %d (Atoi's error was discarded, giving id 0)",
			resp.StatusCode)
	}
}

func itoa(i int) string { return strconv.Itoa(i) }
```

> Full import block for `controllers/product_controller_test.go`:
> ```go
> import (
> 	"net/http"
> 	"strconv"
> 	"strings"
> 	"testing"
>
> 	"github.com/loloDawit/go-admin/internal/testutil"
> 	"github.com/loloDawit/go-admin/models"
> )
> ```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./controllers/ -run 'TestUpdateProduct|TestGetMissing|TestNonNumeric' -v`
Expected: FAIL on all three.

- [ ] **Step 3: Add the shared path-id helper**

Create `controllers/helpers.go`:

```go
package controllers

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/loloDawit/go-admin/internal/httpx"
	"gorm.io/gorm"
)


func pathId(ctx *fiber.Ctx) (int, error) {
	id, err := strconv.Atoi(ctx.Params("id"))
	if err != nil || id <= 0 {
		return 0, errs.InvalidID
	}
	return id, nil
}

// Callers must use First, not Find: Find returns a zero-valued struct and no
// error for a missing row.
func notFoundOrDBError(ctx *fiber.Ctx, err error, resource string) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return httpx.Fail(ctx, errs.NotFound.WithMessage("%s not found", resource))
	}
	// Wrap, so the driver's text reaches the log but never the client.
	return httpx.Fail(ctx, errs.Database.Wrap(err))
}
```

- [ ] **Step 4: Rewrite the product handlers as the pattern for the rest**

Replace `controllers/product_controller.go`:

```go
package controllers

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/loloDawit/go-admin/database"
	"github.com/loloDawit/go-admin/internal/httpx"
	"github.com/loloDawit/go-admin/models"
)

func GetAllProducts(ctx *fiber.Ctx) error {
	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	perPage, _ := strconv.Atoi(ctx.Query("perPage", "0"))
	return ctx.JSON(models.Paginate(database.DB, &models.Product{}, page, perPage))
}

func GetProduct(ctx *fiber.Ctx) error {
	id, err := pathId(ctx)
	if err != nil {
		return httpx.Fail(ctx, err)
	}

	var product models.Product
	// First, not Find: First returns ErrRecordNotFound for a missing row.
	if err := database.DB.First(&product, id).Error; err != nil {
		return notFoundOrDBError(ctx, err, "product")
	}
	return ctx.JSON(product)
}

func CreateProduct(ctx *fiber.Ctx) error {
	var req httpx.ProductRequest
	if err := ctx.BodyParser(&req); err != nil {
		return httpx.Fail(ctx, errs.InvalidBody)
	}
	if req.Title == "" {
		return httpx.Fail(ctx, errs.MissingField.WithMessage("title is required"))
	}
	if req.Price < 0 {
		return httpx.Fail(ctx, errs.ValidationFailed.WithMessage("price must not be negative"))
	}

	// Built from the DTO so the client cannot set Id.
	product := models.Product{
		Title:       req.Title,
		Description: req.Description,
		Image:       req.Image,
		Price:       req.Price,
	}

	if err := database.DB.Create(&product).Error; err != nil {
		return httpx.Fail(ctx, errs.Database.Wrap(err))
	}
	return ctx.Status(fiber.StatusCreated).JSON(product)
}

func UpdateProduct(ctx *fiber.Ctx) error {
	id, err := pathId(ctx)
	if err != nil {
		return httpx.Fail(ctx, err)
	}

	var req httpx.ProductRequest
	if err := ctx.BodyParser(&req); err != nil {
		return httpx.Fail(ctx, errs.InvalidBody)
	}

	var product models.Product
	if err := database.DB.First(&product, id).Error; err != nil {
		return notFoundOrDBError(ctx, err, "product")
	}

	// Must be a map: the struct form skips zero values. Id comes from the
	// path only.
	if err := database.DB.Model(&product).Updates(map[string]any{
		"title":       req.Title,
		"description": req.Description,
		"image":       req.Image,
		"price":       req.Price,
	}).Error; err != nil {
		return httpx.Fail(ctx, errs.Database.Wrap(err))
	}

	return ctx.JSON(product)
}

func DeleteProduct(ctx *fiber.Ctx) error {
	id, err := pathId(ctx)
	if err != nil {
		return httpx.Fail(ctx, err)
	}

	result := database.DB.Delete(&models.Product{}, id)
	if result.Error != nil {
		return httpx.Fail(ctx, errs.Database.Wrap(err))
	}
	if result.RowsAffected == 0 {
		return httpx.Fail(ctx, errs.NotFound)
	}

	return ctx.SendStatus(fiber.StatusNoContent)
}
```

- [ ] **Step 5: Apply the same pattern to users, orders, and roles**

Repeat for `user_controller.go`, `order_controller.go`, and `role_controller.go`:
- `pathId` instead of `strconv.Atoi(...)` with a discarded error
- `First` instead of `Find` for single-record lookups, with `notFoundOrDBError`
- build the model from a DTO; never `BodyParser` into a struct holding the path id
- check every `.Error` on `Create` / `Updates` / `Delete`
- `role_controller.go`: use `httpx.RoleRequest` — this removes the six unchecked type assertions that panic on numeric permission IDs (§4n)

- [ ] **Step 6: Run to verify it passes**

Run: `go test ./controllers/ -v`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add controllers/
git commit -m "fix(api): block mass assignment, return 404s, check database errors

Handlers preset the path id on a struct and then BodyParser'd into that same
struct, so a client-supplied \"id\" overwrote the path parameter — PUT
/user/5 with {\"id\":1} updated user 1, and on UpdateUser the body could also
set roleId. Requests now build models from typed DTOs.

Also: Find -> First so missing records return 404 instead of 200 with an
empty object; every Create/Updates/Delete error is checked instead of
reporting success; non-numeric ids return 400 rather than silently becoming
id 0; and role_controller's six unchecked type assertions (which panicked on
numeric permission ids) become a typed DTO.

Fixes ASSESSMENT 4g, 4n, 4o, 4p, 4q."
```

---

## Task 9: Order timestamps and the broken chart (§4j, §4k)

`models/order.go` declares `CreatedAt`/`UpdatedAt` as `string`, so GORM never populates them and `Chart`'s `DATE_FORMAT(o.created_at, ...)` groups every order under `""`. The flagship reporting feature has never worked.

**Files:**
- Create: `controllers/order_controller_test.go`
- Modify: `models/order.go`, `controllers/order_controller.go`

- [ ] **Step 1: Write the failing test**

Create `controllers/order_controller_test.go`:

```go
package controllers_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/loloDawit/go-admin/internal/testutil"
	"github.com/loloDawit/go-admin/models"
)

func TestOrderTimestampsArePopulated(t *testing.T) {
	db := testutil.NewDB(t)

	order := models.Order{FirstName: "Ada", LastName: "Lovelace", Email: "ada@example.com"}
	if err := db.Create(&order).Error; err != nil {
		t.Fatalf("create order: %v", err)
	}

	var reloaded models.Order
	db.First(&reloaded, order.Id)

	if reloaded.CreatedAt.IsZero() {
		t.Fatal("CreatedAt must be populated; as a string field GORM never set it")
	}
	if time.Since(reloaded.CreatedAt) > time.Minute {
		t.Fatalf("CreatedAt looks wrong: %v", reloaded.CreatedAt)
	}
}

func TestChartGroupsRevenueByRealDate(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	testutil.SeedUser(t, db, "viewer@example.com", "s3cret-password", "viewer")
	testutil.GrantPermission(t, db, "viewer", "view_orders")
	cookie := testutil.Login(t, app, "viewer@example.com", "s3cret-password")

	order := models.Order{FirstName: "Ada", LastName: "Lovelace", Email: "ada@example.com"}
	db.Create(&order)
	db.Create(&models.OrderItem{OrderId: order.Id, ProductTitle: "Widget", Price: 10.50, Quantity: 2})
	db.Create(&models.OrderItem{OrderId: order.Id, ProductTitle: "Gizmo", Price: 5.25, Quantity: 4})

	req := testutil.NewRequest(http.MethodGet, "/api/v1/chart", nil, cookie)
	resp, _ := app.Test(req, -1)

	var sales []struct {
		Date string  `json:"date"`
		Sum  float64 `json:"sum"`
	}
	json.NewDecoder(resp.Body).Decode(&sales)

	if len(sales) != 1 {
		t.Fatalf("want one day of sales, got %d", len(sales))
	}
	if sales[0].Date == "" {
		t.Error("date must not be empty (it was, because created_at was a string)")
	}
	want := 10.50*2 + 5.25*4
	if sales[0].Sum != want {
		t.Errorf("sum: want %.2f, got %.2f", want, sales[0].Sum)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./controllers/ -run 'TestOrderTimestamps|TestChart' -v`
Expected: FAIL — compile error on `.IsZero()` against a string, proving the type is wrong.

- [ ] **Step 3: Fix the model**

In `models/order.go`:

```go
type Order struct {
	Id        uint   `json:"id"`
	FirstName string `json:"firstName"` // was json:"-", which made it unsettable via the API (4k)
	LastName  string `json:"lastName"`  // was json:"-"
	Email     string `json:"email"`

	// Computed, not stored.
	Name  string  `json:"name" gorm:"-"`
	Total float64 `json:"total" gorm:"-"`

	// Must be time.Time: GORM only auto-populates these when they are.
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	OrderItems []OrderItem `json:"orderItems" gorm:"foreignKey:OrderId"`
}
```

Add a shared computation so the detail path matches the list path (the original computed `Name`/`Total` only inside `Take`, so `GetOrder` always returned `total: 0`):

```go
// Compute fills the derived fields. Call it on every path that returns an
// Order, not just the list — the original omitted it from GetOrder.
func (order *Order) Compute() {
	order.Name = strings.TrimSpace(order.FirstName + " " + order.LastName)

	var total float64
	for _, item := range order.OrderItems {
		total += item.Price * float64(item.Quantity)
	}
	order.Total = total
}
```

Update `Take` to call `orders[i].Compute()`, and call it in `GetOrder` too. Change `OrderItem.Price` from `float32` to `float64` for consistency with `Product.Price`.

- [ ] **Step 4: Fix the chart query**

In `controllers/order_controller.go`:

```go
type Sales struct {
	Date string  `json:"date"`
	Sum  float64 `json:"sum"` // was string, which stringified every total
}

func Chart(ctx *fiber.Ctx) error {
	var sales []Sales

	// Requires created_at to be a real DATETIME.
	err := database.DB.Raw(`
		SELECT DATE(o.created_at) AS date,
		       SUM(oi.price * oi.quantity) AS sum
		FROM orders o
		JOIN order_items oi ON o.id = oi.order_id
		GROUP BY DATE(o.created_at)
		ORDER BY date
	`).Scan(&sales).Error
	if err != nil {
		return httpx.Fail(ctx, errs.Database.Wrap(err))
	}

	return ctx.JSON(sales)
}
```

- [ ] **Step 5: Fix the CSV export (§4r)**

```go
func Export(ctx *fiber.Ctx) error {
	// Per-request: a shared path corrupts concurrent exports.
	file, err := os.CreateTemp("", "orders-*.csv")
	if err != nil {
		return httpx.Fail(ctx, errs.ExportFailed)
	}
	defer os.Remove(file.Name())
	defer file.Close()

	if err := writeOrdersCSV(file); err != nil {
		return httpx.Fail(ctx, errs.ExportFailed)
	}

	ctx.Set("Content-Disposition", `attachment; filename="orders.csv"`)
	return ctx.SendFile(file.Name())
}

func writeOrdersCSV(w io.Writer) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	var orders []models.Order
	if err := database.DB.Preload("OrderItems").Find(&orders).Error; err != nil {
		return err
	}

	if err := writer.Write([]string{"ID", "Name", "Email", "Product Title", "Price", "Quantity"}); err != nil {
		return err
	}

	for _, order := range orders {
		if err := writer.Write([]string{
			strconv.FormatUint(uint64(order.Id), 10),
			strings.TrimSpace(order.FirstName + " " + order.LastName),
			order.Email, "", "", "",
		}); err != nil {
			return err
		}
		for _, item := range order.OrderItems {
			if err := writer.Write([]string{
				"", "", "", item.ProductTitle,

				strconv.FormatFloat(item.Price, 'f', 2, 64),
				strconv.FormatUint(uint64(item.Quantity), 10),
			}); err != nil {
				return err
			}
		}
	}
	return writer.Error()
}
```

- [ ] **Step 6: Run to verify it passes**

Run: `go test ./controllers/ -v`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add models/order.go controllers/order_controller.go controllers/order_controller_test.go
git commit -m "fix(orders): real timestamps, working chart, safe CSV export

Order.CreatedAt/UpdatedAt were declared as string, so GORM never populated
them and Chart's DATE_FORMAT(created_at) grouped every order under the empty
string. The revenue chart has never returned a correct result.

Also: FirstName/LastName drop json:\"-\" so an order can actually receive a
customer name through the API; Total/Name are computed on the detail path as
well as the list; and Export writes a per-request temp file with %.2f prices
instead of a shared ./csv/orders.csv with truncated integers.

Fixes ASSESSMENT 4j, 4k, 4r."
```

---

## Task 10: Upload hardening (§4f)

`file.Filename` goes straight into a path, with no sanitization, no MIME check, no size limit, and a hardcoded `localhost:3000` in the returned URL.

**Files:**
- Create: `controllers/image_controller_test.go`
- Modify: `controllers/image_controller.go`

**Interfaces:**
- Produces: `controllers.Upload(cfg *config.Config) fiber.Handler` — now a closure over config (the route in Task 7 already calls it this way).

- [ ] **Step 1: Write the failing test**

Create `controllers/image_controller_test.go`:

```go
package controllers_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/loloDawit/go-admin/internal/testutil"
)

func uploadRequest(t *testing.T, filename, contentType string, content []byte, cookie *http.Cookie) *http.Request {
	t.Helper()

	var body bytes.Buffer
	w := multipart.NewWriter(&body)

	h := make(map[string][]string)
	h["Content-Disposition"] = []string{`form-data; name="image"; filename="` + filename + `"`}
	h["Content-Type"] = []string{contentType}

	part, err := w.CreatePart(h)
	if err != nil {
		t.Fatalf("multipart: %v", err)
	}
	part.Write(content)
	w.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/upload", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.AddCookie(cookie)
	return req
}

// A PNG header, so the MIME sniffer sees a real image.
var pngBytes = []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 0, 0, 0, 0}

func TestUploadRejectsPathTraversal(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	testutil.SeedUser(t, db, "editor@example.com", "s3cret-password", "editor")
	testutil.GrantPermission(t, db, "editor", "edit_products")
	cookie := testutil.Login(t, app, "editor@example.com", "s3cret-password")

	req := uploadRequest(t, "../../../../tmp/pwned.png", "image/png", pngBytes, cookie)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}

	if resp.StatusCode == http.StatusOK {
		// If it succeeded, make sure at least nothing escaped the directory.
		if _, err := os.Stat("/tmp/pwned.png"); err == nil {
			os.Remove("/tmp/pwned.png")
			t.Fatal("traversal escaped the upload directory")
		}
	}

	entries, _ := os.ReadDir(testutil.TestConfig().UploadDir)
	for _, e := range entries {
		if strings.Contains(e.Name(), "..") || strings.Contains(e.Name(), "/") {
			t.Fatalf("stored filename is unsafe: %q", e.Name())
		}
	}
}

func TestUploadRejectsNonImage(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	testutil.SeedUser(t, db, "editor@example.com", "s3cret-password", "editor")
	testutil.GrantPermission(t, db, "editor", "edit_products")
	cookie := testutil.Login(t, app, "editor@example.com", "s3cret-password")

	req := uploadRequest(t, "shell.php", "application/x-php", []byte("<?php system($_GET['c']); ?>"), cookie)
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("want 400 for a non-image upload, got %d", resp.StatusCode)
	}
}

func TestUploadReturnsConfiguredBaseURL(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	testutil.SeedUser(t, db, "editor@example.com", "s3cret-password", "editor")
	testutil.GrantPermission(t, db, "editor", "edit_products")
	cookie := testutil.Login(t, app, "editor@example.com", "s3cret-password")

	req := uploadRequest(t, "photo.png", "image/png", pngBytes, cookie)
	resp, _ := app.Test(req, -1)

	var body map[string]string
	json.NewDecoder(resp.Body).Decode(&body)

	want := testutil.TestConfig().PublicBaseURL
	if !strings.HasPrefix(body["url"], want) {
		t.Fatalf("url must start with the configured base %q, got %q (was hardcoded to :3000)",
			want, body["url"])
	}
}

func TestMain(m *testing.M) {
	os.MkdirAll(testutil.TestConfig().UploadDir, 0o755)
	code := m.Run()
	os.RemoveAll("./testdata")
	os.Exit(code)
}
```

> Full import block for `controllers/image_controller_test.go`:
> ```go
> import (
> 	"bytes"
> 	"encoding/json"
> 	"mime/multipart"
> 	"net/http"
> 	"net/http/httptest"
> 	"os"
> 	"strings"
> 	"testing"
>
> 	"github.com/loloDawit/go-admin/internal/testutil"
> )
> ```
> `path/filepath` is not needed after all — the traversal assertion uses
> `strings.Contains` on the stored name.

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./controllers/ -run TestUpload -v`
Expected: FAIL — the handler is not yet a closure, and none of the checks exist.

- [ ] **Step 3: Implement**

Replace `controllers/image_controller.go`:

```go
package controllers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/loloDawit/go-admin/internal/config"
	"github.com/loloDawit/go-admin/internal/httpx"
)

// Checked against sniffed content, not the client's Content-Type header.
var allowedImageTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

// Upload stores one image and returns its public URL.
func Upload(cfg *config.Config) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		form, err := ctx.MultipartForm()
		if err != nil {
			return httpx.Fail(ctx, errs.UploadMalformed)
		}

		files := form.File["image"]
		if len(files) != 1 {
			return httpx.Fail(ctx, errs.UploadMalformed)
		}
		header := files[0]

		if header.Size > cfg.MaxUploadBytes {
			return httpx.Fail(ctx, errs.UploadTooLarge.WithMessage(
				"the file exceeds the %d byte limit", cfg.MaxUploadBytes))
		}

		ext, err := sniffImageType(header)
		if err != nil {
			return httpx.Fail(ctx, errs.UploadUnsupportedType)
		}

		// Random, so no client-supplied bytes reach the filesystem.
		name, err := randomName(ext)
		if err != nil {
			return httpx.Fail(ctx, errs.UploadFailed)
		}

		dest := filepath.Join(cfg.UploadDir, name)

		// Defence in depth; the name is already generated.
		absDir, _ := filepath.Abs(cfg.UploadDir)
		absDest, _ := filepath.Abs(dest)
		if !strings.HasPrefix(absDest, absDir+string(os.PathSeparator)) {
			return httpx.Fail(ctx, errs.UploadFailed)
		}

		if err := ctx.SaveFile(header, dest); err != nil {
			return httpx.Fail(ctx, errs.UploadFailed)
		}

		return ctx.JSON(fiber.Map{
			"url": strings.TrimRight(cfg.PublicBaseURL, "/") + "/api/v1/uploads/" + name,
		})
	}
}

// The client's Content-Type is ignored: it is attacker-controlled.
func sniffImageType(header *multipart.FileHeader) (string, error) {
	f, err := header.Open()
	if err != nil {
		return "", err
	}
	defer f.Close()

	buf := make([]byte, 512)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return "", err
	}

	ext, ok := allowedImageTypes[strings.SplitN(http.DetectContentType(buf[:n]), ";", 2)[0]]
	if !ok {
		return "", fmt.Errorf("unsupported content type")
	}
	return ext, nil
}

func randomName(ext string) (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b) + ext, nil
}
```

> Add the `os` import for `os.PathSeparator`.

- [ ] **Step 4: Switch the route to the closure form**

`Upload` is now `Upload(cfg) fiber.Handler` rather than a bare handler, so
`routes/routes.go` must change with it (Task 7 deliberately left it as
`controllers.Upload` so the tree kept compiling in between):

```go
	products.Post("/upload", controllers.Upload(cfg))
```

- [ ] **Step 5: Run to verify it passes**

Run: `go build ./... && go test ./controllers/ -run TestUpload -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add controllers/image_controller.go controllers/image_controller_test.go
git commit -m "fix(upload): sanitize names, allowlist types, cap size, config the URL

file.Filename went straight into \"./uploads/\"+filename with no traversal
check, no MIME check, no size limit, and no collision handling — and the
route was unauthorized until the previous commit.

Stored names are now random hex plus an extension derived from SNIFFED
content (the client's Content-Type is ignored), size is capped from config,
and the returned URL uses PUBLIC_BASE_URL instead of a hardcoded
localhost:3000 that never matched the API's own port.

Fixes ASSESSMENT 4f."
```

---

## Task 11: Frontend contract alignment

Limited to correcting what is provably wrong. No new screens — that is M3.

**Files:**
- Modify: `clients/src/interfaces/user.ts`, `clients/src/pages/Login.tsx`, `clients/src/App.tsx`, `clients/src/Components/Layout.tsx`, `clients/src/Components/Nav.tsx`
- Create: `clients/src/api/client.ts`

- [ ] **Step 1: Fix the type contract**

Replace `clients/src/interfaces/user.ts`:

```ts
export interface Permission {
  id: number;
  name: string;
}

export interface Role {
  id: number;
  name: string;
  // Was `permissions: string` — the backend sends an array of objects
  // (models/role.go), so this never matched reality.
  permissions: Permission[];
}

export interface UserInfo {
  id: number;
  firstName: string;
  lastName: string;
  email: string;
  // Was the literal type `1`, so only the value 1 type-checked.
  roleId: number;
  role: Role;
}

export interface ApiError {
  code: string;
  message: string;
}
```

- [ ] **Step 2: Add a single API client**

Create `clients/src/api/client.ts`:

```ts
import { ApiError } from '../interfaces/user';

const BASE_URL = process.env.REACT_APP_API_URL ?? 'http://localhost:8080';

export class ApiRequestError extends Error {
  constructor(public status: number, public code: string, message: string) {
    super(message);
  }
}

/** Turns a non-2xx response into a typed error. */
export async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  const response = await fetch(`${BASE_URL}/api/v1${path}`, {
    ...init,
    credentials: 'include',
    headers: { 'Content-Type': 'application/json', ...init.headers }
  });

  if (!response.ok) {
    let body: ApiError = { code: 'unknown', message: response.statusText };
    try {
      body = await response.json();
    } catch {
      // A non-JSON body (a proxy error page) keeps the default.
    }
    throw new ApiRequestError(response.status, body.code, body.message);
  }

  if (response.status === 204) {
    return undefined as T;
  }
  return response.json() as Promise<T>;
}
```

- [ ] **Step 3: Rewrite Login with error handling and controlled inputs**

In `clients/src/pages/Login.tsx`: remove axios and `//@ts-ignore`, add `value` props to both inputs, add `<label>` elements, and handle failure — the original threw an unhandled rejection and left the form silently inert.

```tsx
const submit = async (e: SyntheticEvent) => {
  e.preventDefault();
  setError(null);
  setSubmitting(true);
  try {
    await api('/login', { method: 'POST', body: JSON.stringify({ email, password }) });
    setRedirect(true);
  } catch (err) {
    setError(err instanceof ApiRequestError ? err.message : 'Something went wrong. Please try again.');
  } finally {
    setSubmitting(false);
  }
};
```

Render `{error && <div className="alert alert-danger" role="alert">{error}</div>}` above the form, and disable the button while `submitting`.

Add below the form, per ADR 0001:

```tsx
<p className="mt-3 text-muted">
  Accounts are created by an administrator. Contact yours to request access.
</p>
```

- [ ] **Step 4: Remove the registration route**

In `clients/src/App.tsx`, delete the `Register` import and its `<Route>`.

- [ ] **Step 5: Remove `//@ts-ignore` from Layout and Nav**

Convert both to `api<UserInfo>('/user')`. In `Nav.tsx`, wrap the fetch in try/catch — the original had none, so every signed-out render produced an unhandled rejection. Fix the invalid markup while here: wrap each `<Link>` in an `<li>`, and change `<a href="#">` to a `<Link to="/">`.

- [ ] **Step 6: Verify**

```bash
cd clients && npx tsc --noEmit && npm run build
```
Expected: no type errors; build succeeds.

- [ ] **Step 7: Commit**

```bash
git add clients/src
git commit -m "fix(web): correct the API contract, centralize the client, handle errors

interfaces/user.ts declared Role.permissions as a string (the backend sends
an array of objects) and roleId as the literal type 1, so the hand-written
contract had already drifted from the API.

Adds a single typed api() client with the base URL in one place, removes the
//@ts-ignore above every axios call, gives Login a visible error state
instead of an unhandled rejection, and removes the registration route per
ADR 0001."
```

---

## Task 12: CI, README, and the definition of done

**Files:**
- Create: `.github/workflows/ci.yml`
- Modify: `README.md`

- [ ] **Step 1: Write the CI workflow**

```yaml
name: CI

on:
  push:
    branches: [main]
  pull_request:

jobs:
  api:
    name: API
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.27'
          cache: true

      - name: Format check
        run: make gofmtcheck

      - name: Vet
        run: go vet ./...

      - name: Build
        run: go build ./...

      # testcontainers uses the Docker daemon that ubuntu-latest provides.
      - name: Test
        run: go test ./... -count=1

  web:
    name: Web
    runs-on: ubuntu-latest
    defaults:
      run:
        working-directory: clients
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: '22'
          cache: npm
          cache-dependency-path: clients/package-lock.json

      - run: npm ci
      - run: npx tsc --noEmit
      - run: npm run build
```

- [ ] **Step 2: Write the README**

````markdown
# go-admin

A staff back-office for a small e-commerce shop: catalog, orders, customers,
staff roles and permissions, and a sales dashboard.

**Status:** under active revival. See [`docs/ASSESSMENT.md`](docs/ASSESSMENT.md)
for the full technical assessment and the milestone plan. This is M0 —
the application boots safely and the critical security defects are closed,
but most screens do not exist yet (M3).

## Requirements

- Go 1.27+
- Docker (for MySQL, and for the test suite)
- Node 22+

## Quick start

```bash
cp .env.example .env
# Generate a real signing key — the app refuses to start without one.
sed -i '' "s|^SESSION_SECRET=.*|SESSION_SECRET=$(openssl rand -hex 32)|" .env

make dev     # starts MySQL, seeds it, runs the API on :8080
make web     # in a second terminal: React dev server on :3000
```

Sign in with the `OWNER_EMAIL` / `OWNER_PASSWORD` from your `.env`.
**Change that password immediately, then remove `OWNER_PASSWORD` from `.env`.**

## Accounts

There is no public registration — see
[ADR 0001](docs/decisions/0001-remove-public-registration.md). The first
account comes from `make seed`; every other account is created by an admin
through the Users screen (or `POST /api/v1/users`).

Built-in roles: `owner` (everything), `admin` (everything but editing roles),
`staff` (catalog and orders).

## Commands

| Command | What it does |
|---|---|
| `make dev` | Database up, seeded, API running |
| `make up` / `make down` | Start / stop MySQL |
| `make seed` | Create permissions, roles, and the owner (idempotent) |
| `make test` | Full Go suite; starts its own MySQL via testcontainers |
| `make lint` | `gofmt` check plus `go vet` |
| `make web` | React dev server |

## Configuration

Every setting comes from the environment; see `.env.example`. The application
**refuses to start** if `DB_DSN`, `SESSION_SECRET` (min 32 bytes), or
`ALLOWED_ORIGIN` is missing — a misconfigured deploy fails loudly at boot
rather than silently issuing empty session cookies.

## Layout

```
main.go              API entrypoint
cmd/seed/            Bootstrap command
routes/              Route table; permission groups live here
controllers/         HTTP handlers
models/              GORM models
middlewares/         Authentication and authorization
internal/config/     Typed configuration
internal/auth/       Password hashing
internal/httpx/      Request DTOs and the shared error shape
internal/seed/       Idempotent seeding
internal/testutil/   Integration-test harness (testcontainers)
clients/             React + TypeScript frontend
docs/                Assessment, decisions, plans
```

## Known limitations in M0

These are tracked, not forgotten — see `docs/ASSESSMENT.md` §10:

- `database.DB` is still a package-level global (M1 injects it)
- Schema is still `AutoMigrate`, not versioned migrations (M1)
- Sessions are JWTs and cannot be revoked (M4)
- No password reset or email (M4)
- Orders have no status/lifecycle (M2)
- Products, Orders, Roles, and Customers have no UI yet (M3)
````

- [ ] **Step 3: Verify the definition of done, end to end, from scratch**

```bash
docker compose down -v          # destroy the volume: simulate a clean clone
rm -f .env
cp .env.example .env
sed -i '' "s|^SESSION_SECRET=.*|SESSION_SECRET=$(openssl rand -hex 32)|" .env
make up
make seed
make test
make lint
go build ./...
(cd clients && npm ci && npx tsc --noEmit && npm run build)
```

Every command must succeed. Then start the API and confirm by hand:

```bash
make api &
# Wrong password must fail:
curl -si -X POST localhost:8080/api/v1/login -H 'Content-Type: application/json' \
  -d '{"email":"owner@example.com","password":"test"}' | head -1   # expect 401

# Correct password must succeed and set an HttpOnly cookie with no token in the body:
curl -si -X POST localhost:8080/api/v1/login -H 'Content-Type: application/json' \
  -d "{\"email\":\"$OWNER_EMAIL\",\"password\":\"$OWNER_PASSWORD\"}"

# Registration must be gone:
curl -s -o /dev/null -w '%{http_code}\n' -X POST localhost:8080/api/v1/register  # expect 404
```

- [ ] **Step 4: Commit**

```bash
git add .github/workflows/ci.yml README.md
git commit -m "ci: lint, vet, test, and build on every PR; write a real README

Replaces the one-line README with setup instructions, the command table, the
configuration contract, and an explicit list of what M0 does NOT fix."
```

---

## Definition of done for M0

- [ ] A fresh clone plus `cp .env.example .env`, a generated secret, and `make dev` yields a running, seeded, signed-in-capable application.
- [ ] `make test` passes with no manual database setup.
- [ ] `make lint` and `go build ./...` are clean.
- [ ] `npx tsc --noEmit` and `npm run build` are clean.
- [ ] CI runs all of the above on every pull request.
- [ ] No credential appears in any tracked file.
- [ ] The application refuses to start on missing or invalid configuration.
- [ ] These specific tests pass, each of which **failed against the code at the start of M0**:
  - `TestCheckPasswordRejectsWrongPassword` (§4a)
  - `TestWriteRoutesRejectUserWithoutEditPermission` (§4b)
  - `TestLoginDoesNotReturnTokenInBody` / `TestLoginDoesNotLeakAccountExistence` (§4e, §4i)
  - `TestUpdateUserInfoPreservesCamelCaseNames` (§4x)
  - `TestLastPageRoundsUpOnPartialFinalPage` (§4l)
  - `TestUpdateProductIgnoresIdInBody` (§4g)
  - `TestGetMissingProductReturns404` / `TestNonNumericIdReturns400` (§4o, §4q)
  - `TestOrderTimestampsArePopulated` / `TestChartGroupsRevenueByRealDate` (§4j)
  - `TestUploadRejectsPathTraversal` / `TestUploadRejectsNonImage` (§4f)
  - `TestRunIsIdempotent` (§6.2)
- [ ] `POST /api/v1/register` returns 404.
- [ ] ADR 0001 is committed and referenced from the README.

## Explicitly NOT in M0

Deferred deliberately; do not let scope creep pull them in:

- Package restructure to `cmd/` + `internal/{domain,http,store}` → **M1**
- `golang-migrate` replacing `AutoMigrate` → **M1**
- Removing the `database.DB` global via injection → **M1**
- Postgres, `sqlc`, or the Fiber → `net/http` decision → **M1**
- Order status and lifecycle, `Customer`, `Payment`, audit log → **M2**
- Products / Orders / Roles / Customers screens → **M3**
- Server-side revocable sessions, CSRF tokens, password reset, email → **M4**
