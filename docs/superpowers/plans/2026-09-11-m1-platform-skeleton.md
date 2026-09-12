# M1 — Platform Skeleton Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Boot four services and three isolated databases with `make dev`, and prove the topology with an automated test that drives a real request through the gateway, into a service, into that service's own database, and back.

**Architecture:** One Go module, four binaries under `services/`. Shared *technical* infrastructure in `platform/` — never domain. Each service owns a database with its own restricted role. A temporary `platformcheck` capability per service reads `golang-migrate`'s `schema_migrations` table, proving migrations, connectivity, credentials, routing, and the error path in one request. No domain functionality exists in M1.

**Tech Stack:** Go 1.27, chi, pgx/v5, golang-migrate, `log/slog`, PostgreSQL 17, Docker Compose.

**Spec:** `docs/superpowers/specs/2026-09-11-m1-platform-skeleton-design.md`

## Global Constraints

- **Module is `github.com/loloDawit/go-admin`.** One root `go.mod`. Do not create per-service modules or `go.work`.
- **All service implementation Go packages live under `internal/`, except executable entry points under `cmd/`.** `migrations/`, `openapi/`, and `Dockerfile` sit at the service root.
- **`platform/` is technical infrastructure only.** No business or domain model, no permission list.
- **No domain functionality in M1.** No login, product, order, staff, session, or customer. A domain capability package under `services/*/internal/` is out of scope.
- **The legacy app still exists** at the repo root (`main.go`, `controllers/`, `models/`, `routes/`, `middlewares/`, `database/`, `utils/`, `internal/`, `clients/`, `cmd/seed`). It is deleted in M2–M5, **not now**. New code must never import it; Task 10 enforces this.
- **Errors split at the HTTP boundary.** Domain and service layers return sentinel errors with no HTTP knowledge. `internal/httperr` per service maps sentinel → `{code, message, status}`. `platform/httpx` supplies the envelope and writer, never the catalogue.
- **No 5xx response body may contain** SQL, driver text, a host name, a DSN, or a credential.
- **Comments state constraints, not changes.** A comment earns its place only if it names something a reader would break by tidying the code. No restating code, no narrating history. Most functions need none. Applies to tests, YAML, and SQL.
- **Tests must discriminate.** Write the failing test, run it, confirm it fails for the stated reason, then implement.
- **Gateway owns no database and gets no migrations.** Do not create empty migration files to make a checklist uniform.
- **`/_platform/*` is reserved for infrastructure** and is never a product endpoint.
- Green means: `gofmt -l`, `go vet ./...`, `go build ./...`, and the scoped test command all clean.

---

## File Structure

**Created — shared platform**

| Path | Responsibility |
|---|---|
| `platform/httpx/envelope.go` | `ErrorBody`, `WriteJSON`, `WriteError`. The one wire shape for errors. |
| `platform/requestid/requestid.go` | Header constant, middleware, context accessors. |
| `platform/observability/logger.go` | `slog` construction; request-logging middleware. |
| `platform/observability/captured.go` | `CapturedHandler` — a `slog.Handler` that records records, for deterministic log assertions. |
| `platform/pgx/pool.go` | Pool construction and a liveness probe. |

**Created — per service** (`identity`, `catalog`, `orders`)

| Path | Responsibility |
|---|---|
| `services/<svc>/cmd/<svc>/main.go` | Wiring and lifecycle only. |
| `services/<svc>/internal/config/config.go` | Typed env config, fail-fast. |
| `services/<svc>/internal/httperr/httperr.go` | Sentinel → `{code, message, status}`. |
| `services/<svc>/internal/platformcheck/*.go` | **Temporary.** Walking-skeleton capability. |
| `services/<svc>/migrations/000001_init.{up,down}.sql` | First migration; creates `schema_migrations`. |
| `services/<svc>/openapi/<svc>.yaml` | Published contract. |
| `services/<svc>/Dockerfile` | Multi-stage build. |

**Created — gateway**

| Path | Responsibility |
|---|---|
| `services/gateway/cmd/gateway/main.go` | Wiring and lifecycle. |
| `services/gateway/internal/config/config.go` | Upstream URLs, timeouts. |
| `services/gateway/internal/routing/routing.go` | Fixed route → upstream mapping, reverse proxy. |
| `services/gateway/internal/httperr/httperr.go` | Gateway-side error mapping. |
| `services/gateway/Dockerfile` | Multi-stage build. **No migrations.** |

**Created — infrastructure**

| Path | Responsibility |
|---|---|
| `deploy/compose/docker-compose.yml` | The whole stack. |
| `deploy/compose/postgres/init/01-roles-and-databases.sql` | Databases, restricted roles, revocations. |
| `test/integration/smoke_test.go` | The definition-of-done test. |
| `test/integration/isolation_test.go` | Credential isolation. |
| `test/arch/imports_test.go` | Import-boundary walk. |
| `.github/workflows/ci-services.yml` | Matrixed build/test. |

**Modified**

| Path | Change |
|---|---|
| `Makefile` | `dev`/`test`/`up`/`down` repoint to the new stack, scoped to `./services/... ./platform/...`. |
| `go.mod` | Add chi, pgx/v5, golang-migrate. |

---

## Task 1: Platform error envelope and request IDs

**Files:**
- Create: `platform/httpx/envelope.go`, `platform/httpx/envelope_test.go`
- Create: `platform/requestid/requestid.go`, `platform/requestid/requestid_test.go`
- Modify: `go.mod`

**Interfaces:**
- Consumes: nothing.
- Produces:
  - `httpx.ErrorBody{Code, Message string}`
  - `httpx.WriteJSON(w http.ResponseWriter, status int, v any)`
  - `httpx.WriteError(w http.ResponseWriter, status int, code, message string)`
  - `requestid.Header = "X-Request-Id"`
  - `requestid.Middleware(next http.Handler) http.Handler`
  - `requestid.FromContext(ctx context.Context) string`

- [ ] **Step 1: Write the failing tests**

`platform/httpx/envelope_test.go`:

```go
package httpx_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/loloDawit/go-admin/platform/httpx"
)

func TestWriteErrorUsesTheEnvelope(t *testing.T) {
	rec := httptest.NewRecorder()

	httpx.WriteError(rec, http.StatusServiceUnavailable, "database_unavailable", "the service is not ready")

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status: want 503, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("content-type: want application/json, got %q", ct)
	}

	var body httpx.ErrorBody
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Code != "database_unavailable" {
		t.Errorf("code: got %q", body.Code)
	}
	if body.Message != "the service is not ready" {
		t.Errorf("message: got %q", body.Message)
	}
}

func TestWriteJSONSetsStatusAndBody(t *testing.T) {
	rec := httptest.NewRecorder()

	httpx.WriteJSON(rec, http.StatusOK, map[string]string{"service": "identity"})

	if rec.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", rec.Code)
	}
	var body map[string]string
	json.NewDecoder(rec.Body).Decode(&body)
	if body["service"] != "identity" {
		t.Errorf("body: got %v", body)
	}
}
```

`platform/requestid/requestid_test.go`:

```go
package requestid_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/loloDawit/go-admin/platform/requestid"
)

func TestMiddlewarePreservesAClientSuppliedID(t *testing.T) {
	var seen string
	h := requestid.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = requestid.FromContext(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(requestid.Header, "client-supplied-id")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if seen != "client-supplied-id" {
		t.Errorf("context: want client-supplied-id, got %q", seen)
	}
	if echoed := rec.Header().Get(requestid.Header); echoed != "client-supplied-id" {
		t.Errorf("response header: want client-supplied-id, got %q", echoed)
	}
}

func TestMiddlewareGeneratesAnIDWhenAbsent(t *testing.T) {
	var seen string
	h := requestid.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = requestid.FromContext(r.Context())
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if seen == "" {
		t.Fatal("an ID must be generated when the client sends none")
	}
	if rec.Header().Get(requestid.Header) != seen {
		t.Error("the generated ID must be echoed in the response header")
	}
}

func TestFromContextIsEmptyWithoutMiddleware(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if got := requestid.FromContext(req.Context()); got != "" {
		t.Errorf("want empty, got %q", got)
	}
}
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./platform/... -v`
Expected: FAIL — packages do not exist.

- [ ] **Step 3: Implement**

`platform/httpx/envelope.go`:

```go
// Package httpx supplies the wire shape for HTTP responses. It owns the
// envelope, never the catalogue of errors: each service maps its own sentinels
// in internal/httperr.
package httpx

import (
	"encoding/json"
	"net/http"
)

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// Message must be safe to show a client: no SQL, driver text, host names, or
// credentials. Log the cause instead.
func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, ErrorBody{Code: code, Message: message})
}
```

`platform/requestid/requestid.go`:

```go
package requestid

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

const Header = "X-Request-Id"

type ctxKey struct{}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(Header)
		if id == "" {
			id = generate()
		}
		w.Header().Set(Header, id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, id)))
	})
}

func FromContext(ctx context.Context) string {
	id, _ := ctx.Value(ctxKey{}).(string)
	return id
}

func generate() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(b)
}
```

- [ ] **Step 4: Run to verify pass**

Run: `go test ./platform/... -v`
Expected: PASS — all five tests.

- [ ] **Step 5: Commit**

```bash
git add platform/httpx platform/requestid go.mod go.sum
git commit -m "feat(platform): error envelope and request ID propagation"
```

---

## Task 2: Structured logging with deterministic test capture

**Files:**
- Create: `platform/observability/logger.go`, `platform/observability/captured.go`, `platform/observability/logger_test.go`

**Interfaces:**
- Consumes: `requestid.FromContext` (Task 1).
- Produces:
  - `observability.NewLogger(service string, w io.Writer) *slog.Logger`
  - `observability.RequestLogger(logger *slog.Logger) func(http.Handler) http.Handler`
  - `observability.NewCaptured() (*slog.Logger, *Captured)`
  - `(*Captured).Records() []slog.Record`, `(*Captured).Attr(i int, key string) (slog.Value, bool)`

`Captured` exists so log assertions read structured attributes in-process. Tests must never grep rendered container logs — that couples the assertion to log formatting and is flaky.

- [ ] **Step 1: Write the failing test**

`platform/observability/logger_test.go`:

```go
package observability_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/loloDawit/go-admin/platform/observability"
	"github.com/loloDawit/go-admin/platform/requestid"
)

func TestRequestLoggerRecordsTheExpectedAttributes(t *testing.T) {
	logger, captured := observability.NewCaptured()

	h := requestid.Middleware(
		observability.RequestLogger(logger)(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusTeapot)
			}),
		),
	)

	req := httptest.NewRequest(http.MethodGet, "/_platform", nil)
	req.Header.Set(requestid.Header, "known-id")
	h.ServeHTTP(httptest.NewRecorder(), req)

	records := captured.Records()
	if len(records) != 1 {
		t.Fatalf("want exactly one log record, got %d", len(records))
	}

	for _, tc := range []struct{ key, want string }{
		{"request_id", "known-id"},
		{"route", "/_platform"},
		{"method", "GET"},
	} {
		v, ok := captured.Attr(0, tc.key)
		if !ok {
			t.Errorf("attribute %q missing", tc.key)
			continue
		}
		if v.String() != tc.want {
			t.Errorf("%s: want %q, got %q", tc.key, tc.want, v.String())
		}
	}

	if v, ok := captured.Attr(0, "status"); !ok || v.Int64() != http.StatusTeapot {
		t.Errorf("status: want 418, got %v", v)
	}
	if _, ok := captured.Attr(0, "duration_ms"); !ok {
		t.Error("duration_ms missing")
	}
}
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./platform/observability/ -v`
Expected: FAIL — `undefined: observability.NewCaptured`.

- [ ] **Step 3: Implement**

`platform/observability/logger.go`:

```go
package observability

import (
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/loloDawit/go-admin/platform/requestid"
)

func NewLogger(service string, w io.Writer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{Level: slog.LevelInfo})).
		With(slog.String("service", service))
}

func RequestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(rec, r)

			logger.InfoContext(r.Context(), "request",
				slog.String("request_id", requestid.FromContext(r.Context())),
				slog.String("method", r.Method),
				slog.String("route", r.URL.Path),
				slog.Int("status", rec.status),
				slog.Int64("duration_ms", time.Since(start).Milliseconds()),
			)
		})
	}
}

// statusRecorder captures the status for the log line. WriteHeader may not be
// called at all, so the zero value must be 200.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}
```

`platform/observability/captured.go`:

```go
package observability

import (
	"context"
	"log/slog"
	"sync"
)

// Captured records log records in memory so tests can assert structured
// attributes rather than parsing rendered output.
type Captured struct {
	mu      sync.Mutex
	records []slog.Record
}

func NewCaptured() (*slog.Logger, *Captured) {
	c := &Captured{}
	return slog.New(c), c
}

func (c *Captured) Records() []slog.Record {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]slog.Record(nil), c.records...)
}

func (c *Captured) Attr(i int, key string) (slog.Value, bool) {
	records := c.Records()
	if i >= len(records) {
		return slog.Value{}, false
	}
	var (
		found slog.Value
		ok    bool
	)
	records[i].Attrs(func(a slog.Attr) bool {
		if a.Key == key {
			found, ok = a.Value, true
			return false
		}
		return true
	})
	return found, ok
}

func (c *Captured) Enabled(context.Context, slog.Level) bool { return true }

func (c *Captured) Handle(_ context.Context, r slog.Record) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.records = append(c.records, r.Clone())
	return nil
}

func (c *Captured) WithAttrs([]slog.Attr) slog.Handler { return c }
func (c *Captured) WithGroup(string) slog.Handler      { return c }
```

- [ ] **Step 4: Run to verify pass**

Run: `go test ./platform/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add platform/observability
git commit -m "feat(platform): structured request logging with in-process capture for tests"
```

---

## Task 3: Postgres pool

**Files:**
- Create: `platform/pgx/pool.go`, `platform/pgx/pool_test.go`
- Modify: `go.mod`

**Interfaces:**
- Produces: `pgx.NewPool(ctx context.Context, dsn string) (*pgxpool.Pool, error)`, `pgx.ErrEmptyDSN`

- [ ] **Step 1: Add the dependency**

```bash
go get github.com/jackc/pgx/v5@latest
go get github.com/go-chi/chi/v5@latest
go mod tidy
```

- [ ] **Step 2: Write the failing test**

`platform/pgx/pool_test.go`:

```go
package pgx_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	pgxplatform "github.com/loloDawit/go-admin/platform/pgx"
)

func TestNewPoolRejectsAnEmptyDSN(t *testing.T) {
	_, err := pgxplatform.NewPool(context.Background(), "")
	if !errors.Is(err, pgxplatform.ErrEmptyDSN) {
		t.Fatalf("want ErrEmptyDSN, got %v", err)
	}
}

// A malformed DSN must fail at construction, not at first query.
func TestNewPoolRejectsAMalformedDSN(t *testing.T) {
	_, err := pgxplatform.NewPool(context.Background(), "://not-a-dsn")
	if err == nil {
		t.Fatal("a malformed DSN must be rejected")
	}
	if strings.Contains(err.Error(), "password") {
		t.Error("the error must not echo credential material")
	}
}
```

- [ ] **Step 3: Run to verify failure**

Run: `go test ./platform/pgx/ -v`
Expected: FAIL — package does not exist.

- [ ] **Step 4: Implement**

`platform/pgx/pool.go`:

```go
// Package pgx constructs PostgreSQL connection pools.
package pgx

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrEmptyDSN = errors.New("database DSN is empty")

func NewPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	if dsn == "" {
		return nil, ErrEmptyDSN
	}

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		// ParseConfig's error can echo the DSN, credentials included.
		return nil, fmt.Errorf("parse database DSN: invalid format")
	}

	cfg.MaxConns = 10
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnIdleTime = 30 * time.Minute

	// NewWithConfig does not dial. A database that is down must not stop the
	// process from starting; readiness reports it instead.
	return pgxpool.NewWithConfig(ctx, cfg)
}
```

- [ ] **Step 5: Run to verify pass**

Run: `go test ./platform/... -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add platform/pgx go.mod go.sum
git commit -m "feat(platform): postgres pool construction"
```

---

## Task 4: Postgres with three isolated databases

**Files:**
- Create: `deploy/compose/docker-compose.yml`
- Create: `deploy/compose/postgres/init/01-roles-and-databases.sql`
- Create: `test/integration/isolation_test.go`

**Interfaces:**
- Produces: databases `identity_db`, `catalog_db`, `orders_db`; roles `identity_user`, `catalog_user`, `orders_user`, each password `dev_only_<svc>`; Postgres on host port 5433.

Port 5433 avoids colliding with any local Postgres on 5432.

- [ ] **Step 1: Write the init SQL**

`deploy/compose/postgres/init/01-roles-and-databases.sql`:

```sql
-- A superuser role would make every grant below decorative.
CREATE ROLE identity_user WITH LOGIN PASSWORD 'dev_only_identity' NOSUPERUSER NOCREATEDB NOCREATEROLE;
CREATE ROLE catalog_user  WITH LOGIN PASSWORD 'dev_only_catalog'  NOSUPERUSER NOCREATEDB NOCREATEROLE;
CREATE ROLE orders_user   WITH LOGIN PASSWORD 'dev_only_orders'   NOSUPERUSER NOCREATEDB NOCREATEROLE;

CREATE DATABASE identity_db OWNER identity_user;
CREATE DATABASE catalog_db  OWNER catalog_user;
CREATE DATABASE orders_db   OWNER orders_user;

-- PUBLIC holds CONNECT on every database by default, which would let any role
-- reach a sibling database regardless of the grants above.
REVOKE CONNECT ON DATABASE identity_db FROM PUBLIC;
REVOKE CONNECT ON DATABASE catalog_db  FROM PUBLIC;
REVOKE CONNECT ON DATABASE orders_db   FROM PUBLIC;

GRANT CONNECT ON DATABASE identity_db TO identity_user;
GRANT CONNECT ON DATABASE catalog_db  TO catalog_user;
GRANT CONNECT ON DATABASE orders_db   TO orders_user;
```

- [ ] **Step 2: Write the compose file (Postgres only for now)**

`deploy/compose/docker-compose.yml`:

```yaml
name: go-admin

services:
  postgres:
    image: postgres:17
    restart: unless-stopped
    environment:
      POSTGRES_PASSWORD: dev_only_root
    ports:
      - "5433:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data
      - ./postgres/init:/docker-entrypoint-initdb.d:ro
    healthcheck:
      test: ["CMD", "pg_isready", "-U", "postgres"]
      interval: 3s
      timeout: 5s
      retries: 20

volumes:
  pgdata:
```

- [ ] **Step 3: Write the failing isolation test**

`test/integration/isolation_test.go`:

```go
package integration_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

func dsn(t *testing.T, user, password, database string) string {
	t.Helper()
	host := os.Getenv("PGHOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("PGPORT")
	if port == "" {
		port = "5433"
	}
	return "postgres://" + user + ":" + password + "@" + host + ":" + port + "/" + database + "?sslmode=disable"
}

func TestServiceRoleCannotReachAnotherServiceDatabase(t *testing.T) {
	ctx := context.Background()

	conn, err := pgx.Connect(ctx, dsn(t, "identity_user", "dev_only_identity", "catalog_db"))
	if err == nil {
		conn.Close(ctx)
		t.Fatal("identity_user must not be able to connect to catalog_db")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "permission denied") {
		t.Fatalf("want a permission error, got: %v", err)
	}
}

func TestServiceRoleReachesItsOwnDatabase(t *testing.T) {
	ctx := context.Background()

	conn, err := pgx.Connect(ctx, dsn(t, "identity_user", "dev_only_identity", "identity_db"))
	if err != nil {
		t.Fatalf("identity_user must reach identity_db: %v", err)
	}
	defer conn.Close(ctx)

	var one int
	if err := conn.QueryRow(ctx, "SELECT 1").Scan(&one); err != nil {
		t.Fatalf("query: %v", err)
	}
}

// Grants are only meaningful if the role cannot simply grant itself more.
func TestServiceRoleHasNoElevatedAttributes(t *testing.T) {
	ctx := context.Background()

	conn, err := pgx.Connect(ctx, dsn(t, "identity_user", "dev_only_identity", "identity_db"))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer conn.Close(ctx)

	var super, createdb, createrole bool
	err = conn.QueryRow(ctx,
		`SELECT rolsuper, rolcreatedb, rolcreaterole FROM pg_roles WHERE rolname = current_user`,
	).Scan(&super, &createdb, &createrole)
	if err != nil {
		t.Fatalf("query role attributes: %v", err)
	}

	if super {
		t.Error("service role must be NOSUPERUSER")
	}
	if createdb {
		t.Error("service role must be NOCREATEDB")
	}
	if createrole {
		t.Error("service role must be NOCREATEROLE")
	}
}
```

- [ ] **Step 4: Run to verify failure**

```bash
docker compose -f deploy/compose/docker-compose.yml down -v
go test ./test/integration/ -run TestServiceRole -v
```
Expected: FAIL — nothing is listening on 5433.

- [ ] **Step 5: Boot Postgres and re-run**

```bash
docker compose -f deploy/compose/docker-compose.yml up -d --wait
go test ./test/integration/ -run TestServiceRole -v
```
Expected: PASS — all three.

If `TestServiceRoleCannotReachAnotherServiceDatabase` fails because the connection *succeeded*, the `REVOKE CONNECT` did not apply — confirm the init script ran (`docker compose logs postgres`), and remember init scripts run only on an empty volume, so `down -v` first.

- [ ] **Step 6: Commit**

```bash
git add deploy/compose test/integration
git commit -m "feat(deploy): postgres with three isolated service databases"
```

---

## Task 5: The service template — Identity

This task builds one complete service. Tasks 6 replicates it. Get it right here.

**Files:**
- Create: `services/identity/cmd/identity/main.go`
- Create: `services/identity/internal/config/config.go`, `config_test.go`
- Create: `services/identity/internal/httperr/httperr.go`
- Create: `services/identity/internal/platformcheck/{platformcheck.go,repository.go,postgres.go,service.go,handler.go,service_test.go}`
- Create: `services/identity/migrations/000001_init.{up,down}.sql`
- Create: `services/identity/openapi/identity.yaml`
- Create: `services/identity/Dockerfile`

**Interfaces:**
- Consumes: `httpx`, `requestid`, `observability`, `pgx` (Tasks 1–3).
- Produces (replicated verbatim by Task 6):
  - `config.Config{ServiceName, Port, DatabaseURL string}`, `config.Load() (*Config, error)`
  - `platformcheck.SchemaState{Version int, Dirty bool}`
  - `platformcheck.ErrNoMigrations`, `platformcheck.ErrDirtySchema`
  - `platformcheck.Repository` interface with `SchemaState(ctx) (SchemaState, error)`
  - `platformcheck.NewService(Repository) *Service`, `(*Service).Check(ctx) (SchemaState, error)`
  - `platformcheck.NewHandler(*Service, string, func(http.ResponseWriter, error)) *Handler`, `(*Handler).Platform`, `(*Handler).Ready`
  - `httperr.Write(w http.ResponseWriter, err error)` — imports `platformcheck`, never the reverse

- [ ] **Step 1: Write the failing service test**

`services/identity/internal/platformcheck/service_test.go`:

```go
package platformcheck_test

import (
	"context"
	"errors"
	"testing"

	"github.com/loloDawit/go-admin/services/identity/internal/platformcheck"
)

type stubRepo struct {
	state platformcheck.SchemaState
	err   error
}

func (s stubRepo) SchemaState(context.Context) (platformcheck.SchemaState, error) {
	return s.state, s.err
}

func TestCheckAcceptsACleanAppliedSchema(t *testing.T) {
	svc := platformcheck.NewService(stubRepo{state: platformcheck.SchemaState{Version: 3, Dirty: false}})

	state, err := svc.Check(context.Background())
	if err != nil {
		t.Fatalf("want success, got %v", err)
	}
	if state.Version != 3 {
		t.Errorf("version: want 3, got %d", state.Version)
	}
}

// A dirty row means a migration failed partway and the schema is in an unknown
// state. A non-zero version alone is not success.
func TestCheckRejectsADirtySchema(t *testing.T) {
	svc := platformcheck.NewService(stubRepo{state: platformcheck.SchemaState{Version: 3, Dirty: true}})

	if _, err := svc.Check(context.Background()); !errors.Is(err, platformcheck.ErrDirtySchema) {
		t.Fatalf("want ErrDirtySchema, got %v", err)
	}
}

func TestCheckRejectsAnUnmigratedSchema(t *testing.T) {
	svc := platformcheck.NewService(stubRepo{state: platformcheck.SchemaState{Version: 0}})

	if _, err := svc.Check(context.Background()); !errors.Is(err, platformcheck.ErrNoMigrations) {
		t.Fatalf("want ErrNoMigrations, got %v", err)
	}
}

func TestCheckPropagatesRepositoryFailure(t *testing.T) {
	want := errors.New("connection refused")
	svc := platformcheck.NewService(stubRepo{err: want})

	if _, err := svc.Check(context.Background()); !errors.Is(err, want) {
		t.Fatalf("want the repository error, got %v", err)
	}
}
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./services/identity/... -v`
Expected: FAIL — package does not exist.

- [ ] **Step 3: Implement the capability**

`services/identity/internal/platformcheck/platformcheck.go`:

```go
// Package platformcheck is the M1 walking skeleton and is TEMPORARY. It proves
// the gateway -> service -> service-owned database path works. It is removed
// when this service gains its real capabilities in M2. Do not build on it.
package platformcheck

import "errors"

type SchemaState struct {
	Version int
	Dirty   bool
}

var (
	ErrNoMigrations = errors.New("no migrations applied")
	ErrDirtySchema  = errors.New("schema is in a dirty state")
)
```

`services/identity/internal/platformcheck/repository.go`:

```go
package platformcheck

import "context"

type Repository interface {
	SchemaState(ctx context.Context) (SchemaState, error)
}
```

`services/identity/internal/platformcheck/service.go`:

```go
package platformcheck

import "context"

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Check(ctx context.Context) (SchemaState, error) {
	state, err := s.repo.SchemaState(ctx)
	if err != nil {
		return SchemaState{}, err
	}
	if state.Dirty {
		return SchemaState{}, ErrDirtySchema
	}
	if state.Version == 0 {
		return SchemaState{}, ErrNoMigrations
	}
	return state, nil
}
```

`services/identity/internal/platformcheck/postgres.go`:

```go
package platformcheck

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// schema_migrations is created by golang-migrate. Its absence means migrations
// have never run against this database.
func (r *PostgresRepository) SchemaState(ctx context.Context) (SchemaState, error) {
	var state SchemaState

	err := r.pool.QueryRow(ctx, `SELECT version, dirty FROM schema_migrations LIMIT 1`).
		Scan(&state.Version, &state.Dirty)

	if errors.Is(err, pgx.ErrNoRows) {
		return SchemaState{}, nil
	}
	if err != nil {
		return SchemaState{}, err
	}
	return state, nil
}
```

`services/identity/internal/platformcheck/handler.go`:

```go
package platformcheck

import (
	"net/http"

	"github.com/loloDawit/go-admin/platform/httpx"
	"github.com/loloDawit/go-admin/platform/requestid"
)

// writeErr is injected rather than imported: internal/httperr must import this
// package for its sentinels, so importing it back would be a cycle.
type Handler struct {
	svc         *Service
	serviceName string
	writeErr    func(http.ResponseWriter, error)
}

func NewHandler(svc *Service, serviceName string, writeErr func(http.ResponseWriter, error)) *Handler {
	return &Handler{svc: svc, serviceName: serviceName, writeErr: writeErr}
}

type response struct {
	Service       string `json:"service"`
	SchemaVersion int    `json:"schemaVersion"`
	RequestID     string `json:"requestId"`
}

func (h *Handler) Platform(w http.ResponseWriter, r *http.Request) {
	state, err := h.svc.Check(r.Context())
	if err != nil {
		h.writeErr(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, response{
		Service:       h.serviceName,
		SchemaVersion: state.Version,
		RequestID:     requestid.FromContext(r.Context()),
	})
}

// Ready reports whether this process can serve. It checks the same conditions
// as Platform but returns no body.
func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	if _, err := h.svc.Check(r.Context()); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}
```

`services/identity/internal/httperr/httperr.go`:

```go
// Package httperr maps this service's sentinel errors to client-facing
// responses. It is the only place in the service that decides a status code or
// a client-visible message.
package httperr

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/loloDawit/go-admin/platform/httpx"
	"github.com/loloDawit/go-admin/services/identity/internal/platformcheck"
)

func Write(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, platformcheck.ErrDirtySchema):
		httpx.WriteError(w, http.StatusServiceUnavailable, "schema_dirty", "the service is not ready")
	case errors.Is(err, platformcheck.ErrNoMigrations):
		httpx.WriteError(w, http.StatusServiceUnavailable, "schema_not_migrated", "the service is not ready")
	default:
		// The cause reaches the log; the client gets none of it.
		slog.Error("unmapped error", slog.String("error", err.Error()))
		httpx.WriteError(w, http.StatusServiceUnavailable, "database_unavailable", "the service is not ready")
	}
}
```

- [ ] **Step 4: Run to verify pass**

Run: `go test ./services/identity/... -v`
Expected: PASS — four tests.

- [ ] **Step 5: Write config with fail-fast, and its test**

`services/identity/internal/config/config_test.go`:

```go
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
```

`services/identity/internal/config/config.go`:

```go
package config

import (
	"errors"
	"os"
)

const serviceName = "identity"

type Config struct {
	ServiceName string
	Port        string
	DatabaseURL string
}

// Any error here is fatal: a misconfigured service must fail at startup, not at
// the first request.
func Load() (*Config, error) {
	cfg := &Config{
		ServiceName: serviceName,
		Port:        withDefault("PORT", "8081"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
	}
	if cfg.DatabaseURL == "" {
		return nil, errors.New("DATABASE_URL is required")
	}
	return cfg, nil
}

func withDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
```

- [ ] **Step 6: Write the first migration**

`services/identity/migrations/000001_init.up.sql`:

```sql
-- PUBLIC may create objects in the public schema by default.
REVOKE CREATE ON SCHEMA public FROM PUBLIC;

COMMENT ON SCHEMA public IS 'identity service';
```

`services/identity/migrations/000001_init.down.sql`:

```sql
GRANT CREATE ON SCHEMA public TO PUBLIC;

COMMENT ON SCHEMA public IS NULL;
```

- [ ] **Step 7: Write main.go**

`services/identity/cmd/identity/main.go`:

```go
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/loloDawit/go-admin/platform/observability"
	pgxplatform "github.com/loloDawit/go-admin/platform/pgx"
	"github.com/loloDawit/go-admin/platform/requestid"
	"github.com/loloDawit/go-admin/services/identity/internal/config"
	"github.com/loloDawit/go-admin/services/identity/internal/httperr"
	"github.com/loloDawit/go-admin/services/identity/internal/platformcheck"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logger := observability.NewLogger(cfg.ServiceName, os.Stdout)
	slog.SetDefault(logger)

	ctx := context.Background()

	// A database that is down must not stop the process from starting; readiness
	// reports it and the service recovers when Postgres returns.
	pool, err := pgxplatform.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("database pool", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	handler := platformcheck.NewHandler(
		platformcheck.NewService(platformcheck.NewPostgresRepository(pool)),
		cfg.ServiceName,
		httperr.Write,
	)

	r := chi.NewRouter()
	r.Use(requestid.Middleware)
	r.Use(observability.RequestLogger(logger))

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	r.Get("/readyz", handler.Ready)
	r.Get("/_platform", handler.Platform)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("listening", slog.String("port", cfg.Port))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}
```

- [ ] **Step 8: Write the Dockerfile**

`services/identity/Dockerfile`:

```dockerfile
FROM golang:1.27 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/identity ./services/identity/cmd/identity

FROM gcr.io/distroless/static-debian12
COPY --from=build /out/identity /identity
EXPOSE 8081
ENTRYPOINT ["/identity"]
```

- [ ] **Step 9: Write the OpenAPI stub**

`services/identity/openapi/identity.yaml`:

```yaml
openapi: 3.1.0
info:
  title: Identity Service
  version: 0.1.0
  description: >
    M1 exposes only infrastructure endpoints. Domain endpoints arrive in M2.
paths:
  /healthz:
    get:
      summary: Process liveness. Does not touch the database.
      responses:
        "200": { description: The process is alive }
  /readyz:
    get:
      summary: Readiness. Checks database connectivity and clean migration state.
      responses:
        "200": { description: Ready to serve }
        "503": { description: Not ready, $ref: "#/components/responses/Error" }
  /_platform:
    get:
      summary: >
        Temporary M1 walking skeleton. Removed in M2. Not a product endpoint.
      responses:
        "200":
          description: Schema state
          content:
            application/json:
              schema:
                type: object
                required: [service, schemaVersion, requestId]
                properties:
                  service: { type: string }
                  schemaVersion: { type: integer }
                  requestId: { type: string }
        "503": { description: Not ready, $ref: "#/components/responses/Error" }
components:
  responses:
    Error:
      description: Standard error envelope
      content:
        application/json:
          schema:
            type: object
            required: [code, message]
            properties:
              code: { type: string }
              message: { type: string }
```

- [ ] **Step 10: Verify and commit**

Run: `gofmt -l services platform && go vet ./services/... ./platform/... && go test ./services/... ./platform/... -v`
Expected: clean, all tests pass.

```bash
git add services/identity
git commit -m "feat(identity): service template with the walking-skeleton capability"
```

---

## Task 6: Replicate the template to Catalog and Orders

Mechanical. Copy Task 5's files, changing only the service name, port, and import paths. Do **not** improve the template here — if something is wrong, fix it in Identity first and re-copy, so the three stay identical.

**Files:** the full Task 5 file set under `services/catalog/` and `services/orders/`.

**Interfaces:** identical to Task 5, with `config.Config.ServiceName` of `"catalog"` / `"orders"`.

Substitutions:

| | identity | catalog | orders |
|---|---|---|---|
| `serviceName` const | `identity` | `catalog` | `orders` |
| default `PORT` | `8081` | `8082` | `8083` |
| import path | `services/identity/...` | `services/catalog/...` | `services/orders/...` |
| migration comment | `identity service` | `catalog service` | `orders service` |
| Dockerfile binary | `identity` | `catalog` | `orders` |
| OpenAPI title | Identity Service | Catalog Service | Orders Service |

- [ ] **Step 1: Copy and rewrite**

```bash
for svc in catalog orders; do
  mkdir -p services/$svc
  cp -r services/identity/. services/$svc/
  find services/$svc -type f -exec sed -i '' "s/identity/$svc/g" {} +
  mv services/$svc/cmd/identity services/$svc/cmd/$svc
done
sed -i '' 's/"8081"/"8082"/' services/catalog/internal/config/config.go
sed -i '' 's/"8081"/"8083"/' services/orders/internal/config/config.go
sed -i '' 's/EXPOSE 8081/EXPOSE 8082/' services/catalog/Dockerfile
sed -i '' 's/EXPOSE 8081/EXPOSE 8083/' services/orders/Dockerfile
mv services/catalog/openapi/identity.yaml services/catalog/openapi/catalog.yaml 2>/dev/null || true
mv services/orders/openapi/identity.yaml services/orders/openapi/orders.yaml 2>/dev/null || true
gofmt -w services
```

- [ ] **Step 2: Check the rename did not corrupt anything**

Run: `grep -rn 'identity' services/catalog services/orders`
Expected: no matches. `sed` is blunt — confirm no `Identity` in prose survived, and that the OpenAPI titles read correctly.

- [ ] **Step 3: Verify**

Run: `go build ./services/... && go test ./services/... -v`
Expected: PASS — twelve tests (four per service).

- [ ] **Step 4: Commit**

```bash
git add services/catalog services/orders
git commit -m "feat(catalog,orders): replicate the service template"
```

---

## Task 7: Gateway

**Files:**
- Create: `services/gateway/cmd/gateway/main.go`
- Create: `services/gateway/internal/config/config.go`, `config_test.go`
- Create: `services/gateway/internal/routing/routing.go`, `routing_test.go`
- Create: `services/gateway/internal/httperr/httperr.go`
- Create: `services/gateway/Dockerfile`

**Interfaces:**
- Produces:
  - `config.Config{Port string, Upstreams map[string]string, UpstreamTimeout time.Duration}`
  - `routing.New(upstreams map[string]string, timeout time.Duration) (http.Handler, error)`

Upstreams are **statically configured** — `IDENTITY_URL`, `CATALOG_URL`, `ORDERS_URL`. No service registry.

- [ ] **Step 1: Write the failing routing test**

`services/gateway/internal/routing/routing_test.go`:

```go
package routing_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/loloDawit/go-admin/platform/httpx"
	"github.com/loloDawit/go-admin/services/gateway/internal/routing"
)

func TestRoutesToTheNamedUpstream(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/_platform" {
			t.Errorf("upstream path: want /_platform, got %q", r.URL.Path)
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"service": "identity"})
	}))
	defer upstream.Close()

	h, err := routing.New(map[string]string{"identity": upstream.URL}, time.Second)
	if err != nil {
		t.Fatalf("routing.New: %v", err)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/_platform/identity", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", rec.Code)
	}
	var body map[string]string
	json.NewDecoder(rec.Body).Decode(&body)
	if body["service"] != "identity" {
		t.Errorf("body: got %v", body)
	}
}

func TestUnknownServiceReturnsTheStandardEnvelope(t *testing.T) {
	h, _ := routing.New(map[string]string{"identity": "http://127.0.0.1:1"}, time.Second)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/_platform/nosuchservice", nil))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status: want 404, got %d", rec.Code)
	}
	var body httpx.ErrorBody
	json.NewDecoder(rec.Body).Decode(&body)
	if body.Code == "" {
		t.Error("an unknown route must still return the standard envelope")
	}
}

// An upstream that is down must not leak its address to the client.
func TestUnreachableUpstreamReturns502WithoutLeakingTheAddress(t *testing.T) {
	h, _ := routing.New(map[string]string{"identity": "http://127.0.0.1:1"}, time.Second)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/_platform/identity", nil))

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status: want 502, got %d", rec.Code)
	}
	body := rec.Body.String()
	for _, leak := range []string{"127.0.0.1", "connection refused", "dial tcp"} {
		if strings.Contains(body, leak) {
			t.Errorf("response leaks %q: %s", leak, body)
		}
	}
}

func TestRejectsAnUnparseableUpstream(t *testing.T) {
	if _, err := routing.New(map[string]string{"identity": "://bad"}, time.Second); err == nil {
		t.Fatal("a malformed upstream URL must be rejected at construction")
	}
}
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./services/gateway/... -v`
Expected: FAIL — package does not exist.

- [ ] **Step 3: Implement routing**

`services/gateway/internal/routing/routing.go`:

```go
package routing

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/loloDawit/go-admin/services/gateway/internal/httperr"
)

// New builds the fixed route-to-upstream mapping. Upstreams are statically
// configured; there is deliberately no service registry.
func New(upstreams map[string]string, timeout time.Duration) (http.Handler, error) {
	proxies := make(map[string]*httputil.ReverseProxy, len(upstreams))

	for name, raw := range upstreams {
		target, err := url.Parse(raw)
		if err != nil || target.Scheme == "" || target.Host == "" {
			return nil, fmt.Errorf("upstream %q is not a valid URL", name)
		}

		proxy := httputil.NewSingleHostReverseProxy(target)
		proxy.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, _ error) {
			// The transport error names the upstream host; the client gets none of it.
			httperr.WriteUpstreamUnavailable(w)
		}
		proxies[name] = proxy
	}

	r := chi.NewRouter()

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	r.Get("/readyz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	r.Get("/_platform/{service}", func(w http.ResponseWriter, req *http.Request) {
		name := chi.URLParam(req, "service")
		proxy, ok := proxies[name]
		if !ok {
			httperr.WriteUnknownRoute(w)
			return
		}

		ctx, cancel := contextWithTimeout(req, timeout)
		defer cancel()

		// The upstream exposes its check at a local /_platform; the service name
		// is a gateway-side routing concern only.
		req = req.WithContext(ctx)
		req.URL.Path = "/_platform"
		proxy.ServeHTTP(w, req)
	})

	r.NotFound(func(w http.ResponseWriter, _ *http.Request) { httperr.WriteUnknownRoute(w) })

	return r, nil
}
```

The `context` import is required; the deadline bounds every upstream call:

```go
func contextWithTimeout(r *http.Request, d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), d)
}
```

`services/gateway/internal/httperr/httperr.go`:

```go
// Package httperr owns the gateway's client-facing errors. Transport failures
// name upstream hosts; nothing from them reaches a response body.
package httperr

import (
	"net/http"

	"github.com/loloDawit/go-admin/platform/httpx"
)

func WriteUnknownRoute(w http.ResponseWriter) {
	httpx.WriteError(w, http.StatusNotFound, "not_found", "no such route")
}

func WriteUpstreamUnavailable(w http.ResponseWriter) {
	httpx.WriteError(w, http.StatusBadGateway, "upstream_unavailable", "the service is temporarily unavailable")
}
```

- [ ] **Step 4: Run to verify pass**

Run: `go test ./services/gateway/... -v`
Expected: PASS — four tests.

- [ ] **Step 5: Config, main, Dockerfile**

`services/gateway/internal/config/config.go`:

```go
package config

import (
	"errors"
	"os"
	"time"
)

type Config struct {
	Port            string
	Upstreams       map[string]string
	UpstreamTimeout time.Duration
}

func Load() (*Config, error) {
	cfg := &Config{
		Port:            withDefault("PORT", "8080"),
		UpstreamTimeout: 5 * time.Second,
		Upstreams: map[string]string{
			"identity": os.Getenv("IDENTITY_URL"),
			"catalog":  os.Getenv("CATALOG_URL"),
			"orders":   os.Getenv("ORDERS_URL"),
		},
	}
	for name, url := range cfg.Upstreams {
		if url == "" {
			return nil, errors.New("upstream URL is required for " + name)
		}
	}
	return cfg, nil
}

func withDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
```

`services/gateway/internal/config/config_test.go`:

```go
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
```

`services/gateway/cmd/gateway/main.go` — same shape as Task 5's `main.go`, with these differences: no database or pool; build the handler with `routing.New(cfg.Upstreams, cfg.UpstreamTimeout)`; wrap it in `requestid.Middleware` and `observability.RequestLogger`; exit non-zero if `routing.New` returns an error.

`services/gateway/Dockerfile` — identical to Task 5's, building `./services/gateway/cmd/gateway`, `EXPOSE 8080`. **No migrations directory.**

- [ ] **Step 6: Verify and commit**

Run: `gofmt -l services && go vet ./services/... && go test ./services/... -v`
Expected: clean, all tests pass.

```bash
git add services/gateway
git commit -m "feat(gateway): static upstream routing with the standard error envelope"
```

---

## Task 8: Full stack in Compose, and Makefile

**Files:**
- Modify: `deploy/compose/docker-compose.yml`
- Modify: `Makefile`

**Interfaces:**
- Produces: `make dev`, `make test`, `make up`, `make down` targeting the new stack.

The legacy `make` targets currently boot the MySQL app. M1 repoints them. The legacy `docker-compose.yml` at the repo root stays until M4 deletes it, which is why every new command passes `-f deploy/compose/docker-compose.yml`.

- [ ] **Step 1: Add the services and migration jobs to compose**

Append to `deploy/compose/docker-compose.yml` under `services:`:

```yaml
  identity-migrate:
    image: migrate/migrate:v4.17.1
    depends_on:
      postgres: { condition: service_healthy }
    volumes:
      - ../../services/identity/migrations:/migrations:ro
    command:
      - "-path=/migrations"
      - "-database=postgres://identity_user:dev_only_identity@postgres:5432/identity_db?sslmode=disable"
      - "up"

  identity:
    build:
      context: ../..
      dockerfile: services/identity/Dockerfile
    depends_on:
      identity-migrate: { condition: service_completed_successfully }
    environment:
      PORT: "8081"
      DATABASE_URL: "postgres://identity_user:dev_only_identity@postgres:5432/identity_db?sslmode=disable"
    ports:
      - "8081:8081"

  gateway:
    build:
      context: ../..
      dockerfile: services/gateway/Dockerfile
    depends_on: [identity, catalog, orders]
    environment:
      PORT: "8080"
      IDENTITY_URL: "http://identity:8081"
      CATALOG_URL: "http://catalog:8082"
      ORDERS_URL: "http://orders:8083"
    ports:
      - "8080:8080"
```

Repeat the `-migrate` and service pair for `catalog` (port 8082, `catalog_db`, `catalog_user`, `dev_only_catalog`) and `orders` (8083, `orders_db`, `orders_user`, `dev_only_orders`).

- [ ] **Step 2: Rewrite the Makefile**

```makefile
COMPOSE := docker compose -f deploy/compose/docker-compose.yml
GO_PKGS := ./services/... ./platform/...

.PHONY: help up down dev logs test test-unit test-integration fmt lint tidy

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

up: ## Boot the stack and wait for it to be healthy
	$(COMPOSE) up -d --build --wait

down: ## Stop the stack (volumes preserved)
	$(COMPOSE) down

dev: up ## Boot the stack and show gateway logs
	$(COMPOSE) logs -f gateway

logs: ## Tail all service logs
	$(COMPOSE) logs -f

test: test-unit test-integration ## Run everything

test-unit: ## Unit tests, no stack required
	go test $(GO_PKGS) -count=1

test-integration: up ## Integration tests against the running stack
	go test ./test/... -count=1

fmt: ## Format
	gofmt -w services platform test

lint: ## Vet and format check
	@unformatted=$$(gofmt -l services platform test); \
	if [ -n "$$unformatted" ]; then echo "needs gofmt:"; echo "$$unformatted"; exit 1; fi
	go vet $(GO_PKGS) ./test/...

tidy: ## Tidy modules
	go mod tidy
```

`GO_PKGS` deliberately excludes the repo root: the legacy packages still live there and their tests need MySQL. They are deleted in M2–M4.

- [ ] **Step 3: Boot the whole stack**

```bash
make down
docker compose -f deploy/compose/docker-compose.yml down -v
make up
docker compose -f deploy/compose/docker-compose.yml ps
```
Expected: `postgres`, `identity`, `catalog`, `orders`, `gateway` all up; the three `*-migrate` jobs exited 0.

- [ ] **Step 4: Check the path by hand before automating it**

```bash
curl -s -H 'X-Request-Id: manual-check' localhost:8080/_platform/identity
```
Expected: `{"service":"identity","schemaVersion":1,"requestId":"manual-check"}`

- [ ] **Step 5: Commit**

```bash
git add deploy/compose Makefile
git commit -m "feat(deploy): full stack in compose; repoint make targets at the new services"
```

---

## Task 9: Integration smoke test — the definition of done

**Files:**
- Create: `test/integration/smoke_test.go`

**Interfaces:**
- Consumes: the running stack from Task 8.

- [ ] **Step 1: Write the test**

`test/integration/smoke_test.go`:

```go
package integration_test

import (
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"
)

func gatewayURL() string {
	if v := os.Getenv("GATEWAY_URL"); v != "" {
		return v
	}
	return "http://localhost:8080"
}

type platformResponse struct {
	Service       string `json:"service"`
	SchemaVersion int    `json:"schemaVersion"`
	RequestID     string `json:"requestId"`
}

func TestWalkingSkeletonThroughTheGateway(t *testing.T) {
	client := &http.Client{Timeout: 10 * time.Second}

	for _, service := range []string{"identity", "catalog", "orders"} {
		t.Run(service, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, gatewayURL()+"/_platform/"+service, nil)
			if err != nil {
				t.Fatalf("build request: %v", err)
			}
			req.Header.Set("X-Request-Id", "smoke-"+service)

			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Fatalf("status: want 200, got %d", resp.StatusCode)
			}

			var body platformResponse
			if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
				t.Fatalf("decode: %v", err)
			}

			if body.Service != service {
				t.Errorf("service: want %q, got %q — the gateway routed to the wrong upstream", service, body.Service)
			}
			if body.SchemaVersion == 0 {
				t.Error("schemaVersion is 0 — migrations did not run against this service's database")
			}
			if body.RequestID != "smoke-"+service {
				t.Errorf("requestId: want %q, got %q — the ID did not survive the hop", "smoke-"+service, body.RequestID)
			}
			if echoed := resp.Header.Get("X-Request-Id"); echoed != "smoke-"+service {
				t.Errorf("response header: want %q, got %q", "smoke-"+service, echoed)
			}
		})
	}
}

func TestUnknownServiceReturnsTheStandardEnvelope(t *testing.T) {
	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get(gatewayURL() + "/_platform/nosuchservice")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: want 404, got %d", resp.StatusCode)
	}

	var body struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Code == "" || body.Message == "" {
		t.Errorf("want the standard envelope, got %+v", body)
	}
}
```

- [ ] **Step 2: Run against the stack**

Run: `make test-integration`
Expected: PASS — three subtests plus the envelope test.

- [ ] **Step 3: Prove the dirty-schema path**

```bash
docker compose -f deploy/compose/docker-compose.yml exec -T postgres \
  psql -U identity_user -d identity_db -c "UPDATE schema_migrations SET dirty = true;"

curl -s -o /dev/null -w '%{http_code}\n' localhost:8080/_platform/identity   # expect 503
curl -s localhost:8080/_platform/identity                                     # expect {"code":"schema_dirty",...}
curl -s -o /dev/null -w '%{http_code}\n' localhost:8081/readyz                # expect 503
curl -s -o /dev/null -w '%{http_code}\n' localhost:8081/healthz               # expect 200 — the process is alive

docker compose -f deploy/compose/docker-compose.yml exec -T postgres \
  psql -U identity_user -d identity_db -c "UPDATE schema_migrations SET dirty = false;"
```

Record the observed outputs in the task report. `/healthz` returning 200 while `/readyz` returns 503 is the contract from spec §6 — if `/healthz` also fails, the liveness check is wrongly touching the database.

- [ ] **Step 4: Commit**

```bash
git add test/integration
git commit -m "test(integration): walking-skeleton smoke test through the gateway"
```

---

## Task 10: Import-boundary and error-convention guards

**Files:**
- Create: `test/arch/imports_test.go`

**Interfaces:**
- Consumes: nothing. Reads the source tree.

Go's `internal/` rule already prevents cross-service imports. This test exists to catch someone weakening the layout later — moving a package out of `internal/` to resolve a compile error removes the compiler's protection silently.

- [ ] **Step 1: Write the test**

`test/arch/imports_test.go`:

```go
package arch_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const modulePath = "github.com/loloDawit/go-admin"

var services = []string{"gateway", "identity", "catalog", "orders"}

// Legacy packages at the repo root are deleted in M2-M4. Nothing new may depend
// on them, or deleting them becomes a cross-cutting change.
var legacyRoots = []string{
	modulePath + "/controllers",
	modulePath + "/models",
	modulePath + "/routes",
	modulePath + "/middlewares",
	modulePath + "/database",
	modulePath + "/utils",
	modulePath + "/internal/",
	modulePath + "/cmd/seed",
}

func TestNoServiceImportsAnotherService(t *testing.T) {
	root := repoRoot(t)

	for _, svc := range services {
		svc := svc
		t.Run(svc, func(t *testing.T) {
			forEachImport(t, filepath.Join(root, "services", svc), func(file, imported string) {
				for _, other := range services {
					if other == svc {
						continue
					}
					if strings.HasPrefix(imported, modulePath+"/services/"+other) {
						t.Errorf("%s imports %s.\n\tServices communicate over HTTP through published contracts, never by importing each other.", rel(root, file), imported)
					}
				}
			})
		})
	}
}

func TestNoNewCodeImportsLegacyPackages(t *testing.T) {
	root := repoRoot(t)

	for _, dir := range []string{"services", "platform"} {
		forEachImport(t, filepath.Join(root, dir), func(file, imported string) {
			for _, legacy := range legacyRoots {
				if strings.HasPrefix(imported, legacy) {
					t.Errorf("%s imports legacy package %s.\n\tLegacy code is deleted in M2-M4; new code must not depend on it.", rel(root, file), imported)
				}
			}
		})
	}
}

func TestPlatformHoldsNoDomainConcepts(t *testing.T) {
	root := repoRoot(t)

	banned := []string{"product", "order", "customer", "staff", "permission", "role", "session"}

	err := filepath.Walk(filepath.Join(root, "platform"), func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		name := strings.ToLower(filepath.Base(path))
		for _, word := range banned {
			if strings.Contains(name, word) {
				t.Errorf("%s looks like a domain concept.\n\tplatform/ is technical infrastructure only.", rel(root, path))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk platform: %v", err)
	}
}

func forEachImport(t *testing.T, dir string, check func(file, imported string)) {
	t.Helper()

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return
	}

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		for _, imp := range file.Imports {
			check(path, strings.Trim(imp.Path.Value, `"`))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", dir, err)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	t.Fatal("could not locate the repository root")
	return ""
}

func rel(root, path string) string {
	r, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return r
}
```

- [ ] **Step 2: Run to verify it passes on clean code**

Run: `go test ./test/arch/ -v`
Expected: PASS.

- [ ] **Step 3: Prove the guard catches a violation**

Temporarily add `_ "github.com/loloDawit/go-admin/services/identity/internal/platformcheck"` to a file under `services/catalog/`, run the test, confirm it FAILS naming that file, then revert. Record the output in the task report — a guard nobody has seen fail is not yet a guard.

- [ ] **Step 4: Commit**

```bash
git add test/arch
git commit -m "test(arch): enforce service and legacy import boundaries"
```

---

## Task 11: CI

**Files:**
- Create: `.github/workflows/ci-services.yml`

The existing `.github/workflows/ci.yml` (legacy, MySQL) and `codeql-analysis.yml` stay untouched. Legacy CI is deleted with legacy code in M4.

- [ ] **Step 1: Write the workflow**

```yaml
name: CI (services)

on:
  push:
    branches: [main]
  pull_request:

jobs:
  unit:
    name: Unit
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.27'
          cache: true
      - run: make lint
      - run: go build ./services/... ./platform/...
      - run: go test ./services/... ./platform/... -count=1
      - run: go test ./test/arch/ -count=1

  integration:
    name: Integration
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.27'
          cache: true
      # ubuntu-latest ships a working Docker daemon; the stack and the
      # integration tests both need it.
      - run: docker compose -f deploy/compose/docker-compose.yml up -d --build --wait
      - run: go test ./test/... -count=1
      - if: failure()
        run: docker compose -f deploy/compose/docker-compose.yml logs
```

- [ ] **Step 2: Verify the commands locally**

Run: `make lint && go build ./services/... ./platform/... && go test ./services/... ./platform/... ./test/arch/ -count=1`
Expected: all clean.

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/ci-services.yml
git commit -m "ci: build and test the services on every PR"
```

---

## Task 12: README for the new stack

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Rewrite the run instructions**

Replace the legacy quick-start with the new stack. The README must state:

- what the system is: gateway plus Identity, Catalog, Orders, each owning a database
- requirements: Go 1.27, Docker
- quick start: `make up`, then `curl localhost:8080/_platform/identity`
- the make target table from Task 8
- that `/_platform/*` is **temporary M1 infrastructure**, removed in M2–M4, and not a product API
- that the legacy application still lives at the repo root, is tagged `legacy-v1`, and is deleted in M2–M5
- ports: gateway 8080, identity 8081, catalog 8082, orders 8083, postgres 5433
- that M1 has no domain functionality — no login, products, or orders yet

Every command in it must have been run.

- [ ] **Step 2: Verify from a clean state**

```bash
docker compose -f deploy/compose/docker-compose.yml down -v
make up
make test
```
Record exactly what was run and what happened.

- [ ] **Step 3: Commit**

```bash
git add README.md
git commit -m "docs: README for the service stack"
```

---

## Definition of done

- [ ] `make up` boots postgres, three migrate jobs, identity, catalog, orders, gateway
- [ ] `make test` passes: unit, arch, and integration
- [ ] `GET /_platform/{service}` returns 200 with the right `service`, a non-zero `schemaVersion`, and the client's request ID, for all three services
- [ ] a dirty `schema_migrations` row makes `/readyz` and `/_platform` return 503 while `/healthz` stays 200
- [ ] `identity_user` cannot connect to `catalog_db`, and is NOSUPERUSER/NOCREATEDB/NOCREATEROLE
- [ ] the import-boundary guard has been **seen to fail** on a deliberate violation
- [ ] no 5xx body contains SQL, driver text, a host, or a credential
- [ ] gateway has no migrations directory
- [ ] CI builds and tests every service
- [ ] no domain functionality exists in `services/*/internal/` beyond `platformcheck`

## Explicitly not in M1

| Deferred | Milestone |
|---|---|
| Login, sessions, principal signing, authorization | M2 |
| Staff, roles, permissions | M2 |
| sqlc (no real queries to generate yet) | M2 |
| Products, images, `/internal/products/resolve` | M3 |
| Orders, customers, Orders→Catalog | M4 |
| Frontend | M5 |
| NATS, outbox, messaging | M6 |
| OpenTelemetry, metrics, rate limiting | M7 |
| Kubernetes | M8 |
| Deleting legacy code | M2–M5 |
