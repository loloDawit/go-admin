# go-admin

A staff back-office for a small e-commerce shop: product catalog, orders,
and staff roles and permissions. Customer records and the sales dashboard
are planned but not yet built — see "Known limitations in M0" below.

**Status:** under active revival. See [`docs/ASSESSMENT.md`](docs/ASSESSMENT.md)
for the full technical assessment and the milestone plan. This is milestone
M0 — the application boots safely from a clean clone and the critical
security defects are closed, but most screens do not exist yet (rebuilt in PRD M5, not repaired). See
"Known limitations in M0" below.

## Requirements

- Go 1.27
- Docker (for MySQL, and for the test suite)
- Node 22

## Quick start

```bash
cp .env.example .env
# Generate a real signing key — the app refuses to start without one.
# macOS/BSD sed:
sed -i '' "s|^SESSION_SECRET=.*|SESSION_SECRET=$(openssl rand -hex 32)|" .env
# GNU/Linux sed:
sed -i "s|^SESSION_SECRET=.*|SESSION_SECRET=$(openssl rand -hex 32)|" .env

make dev     # starts MySQL, seeds it, runs the API on :8080
make web     # in a second terminal: React dev server on :3000
```

Sign in with the `OWNER_EMAIL` / `OWNER_PASSWORD` from your `.env`.
**Change that password immediately, then remove `OWNER_PASSWORD` from `.env`.**
`make seed` never resets an existing owner's password — but it does replace
every built-in role's permission grants (`owner`/`admin`/`staff`) with the
hardcoded defaults on every run, so any grant customized through the role
endpoints is reverted on the next `make dev` or deploy.

## Accounts and registration

There is no public registration ([ADR 0001](docs/decisions/0001-remove-public-registration.md)).
`POST /api/v1/register` returns **401, not 404**: the authenticated route
group has an empty path prefix and matches every path under `/api/v1`
before route lookup runs, so an unregistered path fails auth before it can
404 — deliberately, so the status code can't be used to enumerate routes.

The first account is created by `make seed`, which reads `OWNER_EMAIL` and
`OWNER_PASSWORD` from the environment. Every other account is created by an
authenticated admin through `POST /api/v1/users`.

### Built-in roles and permissions

Three roles are seeded: `owner` (every permission), `admin` (everything
except `edit_roles`), and `staff` (view users, view/edit products, view/edit
orders — no role or user management).

The permission vocabulary is `view_<resource>` / `edit_<resource>` for
`users`, `products`, `orders`, and `roles`. A route guarded by
`RequirePermission("products")` accepts `edit_products` on any method, and
also accepts `view_products` on safe methods (GET/HEAD/OPTIONS) — holding
edit implies read.

## `make` targets

| Command | What it does |
|---|---|
| `make up` | Start MySQL (`docker compose up -d --wait`) |
| `make down` | Stop MySQL; the `dbdata` volume is preserved |
| `make seed` | Create permissions, roles, and the owner account (idempotent) |
| `make dev` | `up` + `seed`, then run the API |
| `make api` | Run the API server (`go run .`) |
| `make web` | Run the React dev server |
| `make test` | Full Go test suite; starts its own MySQL via testcontainers |
| `make fmt` | Format all Go source |
| `make gofmtcheck` | Fail if any Go file is unformatted |
| `make lint` | `gofmtcheck` + `go vet` |
| `make tidy` | `go mod tidy` |

## Configuration

Every setting comes from the environment; see `.env.example`. The
application **refuses to start** if any of the following is missing or
invalid, so a misconfigured deploy fails loudly at boot instead of silently
issuing broken session cookies:

- `DB_DSN` — must contain `parseTime=true` (or any value the Go MySQL driver
  parses as true, e.g. `True`); GORM cannot scan `DATETIME` columns into
  `time.Time` without it.
- `SESSION_SECRET` — at least 32 bytes. Generate with `openssl rand -hex 32`.
- `ALLOWED_ORIGIN` — must be non-empty and must be the frontend's exact
  origin, never `*`: the API sends credentialed cookies, and a wildcard
  origin combined with credentials is a CSRF hole.

`APP_ENV=production` turns on `Secure` cookies (HTTPS only); leave it as
`development` for local HTTP.

## Repository layout

```
main.go              API entrypoint
cmd/seed/             Seed command (make seed)
database/             GORM connection setup (database.DB, a package global)
routes/               Route table; permission groups are declared here
controllers/          HTTP handlers
models/               GORM models
middlewares/          Authentication and authorization
utils/                JWT issuing and parsing
internal/config/      Typed, validated configuration
internal/auth/        Password hashing
internal/errs/        The error registry — every client-facing failure
internal/httpx/       Request DTOs and the {code,message} response shape
internal/seed/        Idempotent permission/role/owner seeding
internal/testutil/    Integration-test harness (testcontainers + MySQL)
clients/              React + TypeScript frontend
docs/                 Assessment, ADRs, and milestone plans
```

## Known limitations in M0

See `docs/ASSESSMENT.md` for the full milestone plan.

- `database.DB` is still a package-level global, not injected (rebuilt in PRD M1–M4, not repaired).
- The schema is still created with GORM `AutoMigrate`, not versioned
  migrations (rebuilt in PRD M1–M4, not repaired).
- Sessions are plain JWTs with no server-side store, so a session cannot be
  revoked before it expires (rebuilt in PRD M2, not repaired).
- There is no password reset flow and no email delivery (rebuilt in PRD M2, not repaired).
- Orders have no status or lifecycle (rebuilt in PRD M4, not repaired).
- Products, Orders, Roles, and Customers have no UI. The Users and Dashboard
  pages that exist in `clients/src/pages` are placeholder stubs with static
  markup, not working screens (rebuilt in PRD M5, not repaired).
- `internal/testutil.NewApp` builds the real route table but does not
  install the CORS middleware `main.go` installs — a CORS regression would
  not be caught by the test suite.
- Uploaded files are readable by any signed-in user regardless of
  permissions: the route sweep that checks every route carries the right
  `RequirePermission` deliberately skips the `config.UploadsPath` static
  mount. Uploads are authenticated at all only because that mount is
  registered in `routes.SetupRoutes` after the `authed` group, under the
  same `/api/v1` prefix the `authed` group's empty-path routes match first —
  moving the static mount above it, or off `/api/v1`, would silently make
  uploads public.
- `staff` can enumerate the full role and permission model through
  `GET /api/v1/users`, which preloads `Role.Permissions` on every user in
  the page — the same payload `GET /api/v1/roles` withholds from `staff`.
- The JWT expiry (`utils/jwt.go`) and the session cookie expiry
  (`controllers/auth_controller.go`'s `sessionTTL`) are two separate
  hardcoded 24-hour literals; nothing enforces that they agree, so changing
  one without the other silently produces a cookie that outlives its token
  or a token that outlives its cookie.
