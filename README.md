# go-admin

A staff back-office for a small e-commerce shop, being rebuilt as a small
set of Go microservices: a **gateway** in front of **Identity**, **Catalog**,
and **Orders**, each owning its own PostgreSQL database with its own
restricted database role.

**Status: M1 — platform skeleton.** The stack boots, each service has its
own database and migrations, and the gateway proxies to all three. **There
is no domain functionality yet** — no login, no products, no orders, no
staff or customer records. See "Known limitations in M1" below before you
go looking for a working admin panel.

The architecture, milestone plan, and the assessment behind the decision to
rebuild rather than repair are kept as working documents outside this
repository.

## The legacy application

The original monolith still lives at the repo root (`main.go`,
`controllers/`, `models/`, `clients/`, its own `docker-compose.yml`, etc.).
It is tagged `legacy-v1` and is **not** touched by the instructions below —
it has its own MySQL-based stack. It is kept only as a behavioural
reference while the new services are built, **not a migration source**,
and is deleted milestone by milestone through M5. Do not confuse its files
with the new services under `services/`.

## Requirements

- Go 1.27 (see `go.mod`)
- Docker

## Quick start

There is no `.env` step. Configuration for local development lives directly
in `deploy/compose/docker-compose.yml`.

```bash
make up
curl -s -H 'X-Request-Id: check' localhost:8080/_platform/identity
```

`make up` boots postgres, the three one-shot migrate jobs, identity,
catalog, orders, and the gateway, and waits for all of them to report
healthy. The curl above returned:

```
{"service":"identity","schemaVersion":1,"requestId":"check"}
```

The same call against `/_platform/catalog` and `/_platform/orders` returns
the equivalent body for each service, with `requestId` echoing whatever
`X-Request-Id` the client sent.

## Ports

| Service | Port |
|---|---|
| gateway | 8080 |
| identity | 8081 |
| catalog | 8082 |
| orders | 8083 |
| postgres | 5433 (bound to `127.0.0.1` only, deliberately — see below) |

## `make` targets

| Command | What it does |
|---|---|
| `make help` | List the available targets |
| `make up` | Boot the stack (`docker compose up -d --build --wait`) |
| `make down` | Stop the stack (volumes preserved) |
| `make dev` | `up`, then tail gateway logs |
| `make logs` | Tail all service logs |
| `make test` | `test-unit` + `test-integration` |
| `make test-unit` | Unit and architecture tests; no stack required |
| `make test-integration` | Boots the stack (`up`), then runs the `-tags integration` suite against it |
| `make fmt` | `gofmt -w` over `services`, `platform`, `test` |
| `make lint` | gofmt check + `go vet` (including the integration build tag) |
| `make tidy` | `go mod tidy` |

Integration tests carry `//go:build integration` and need `-tags
integration` to run at all. Without the tag, `go test ./test/integration/...`
matches zero files and fails outright — a mistake you can't miss:

```
go: warning: "./test/integration/..." matched no packages
no packages to test
```

But without the tag against a wider path, such as `go test ./test/...`,
the command still succeeds — it just silently runs 0 integration tests
(only `test/arch` has non-tagged files there) and reports `ok`, which is
why `make test-integration` pins `-tags integration` to the exact
`test/integration/...` path rather than something broader. Also note
`go test ./...` from the repo root pulls in the legacy packages at the
root, which need MySQL and their own setup — that's why the Makefile's
`GO_PKGS` excludes the repo root and scopes to `./services/...` and
`./platform/...` instead. `make test` runs the tag-gated integration
suite plus the plain unit/arch tests, against a stack it boots itself.

## `/_platform/*` is temporary

`GET /_platform/{service}` (proxied through the gateway, and served
directly by each service) reports `{service, schemaVersion, requestId}` and
exists solely so this milestone has something to boot and verify against.
It is **not a product API**. It is removed service by service as each one
gains its real capabilities in M2–M4. Do not build a client against it.

## Credentials are dev-only, on purpose

The database roles and passwords in `deploy/compose/docker-compose.yml`
(for example `dev_only_identity`) are committed in plain text so the stack
runs from a clean clone with no setup step. Each role is created
`NOSUPERUSER NOCREATEDB NOCREATEROLE` and scoped to its own database only
(`deploy/compose/postgres/init/01-roles-and-databases.sql` revokes
`CONNECT` on every database from `PUBLIC`, then grants it back only to
that database's own role — `identity_user` cannot reach `catalog_db`).
Postgres is bound to `127.0.0.1:5433` on the host, not a LAN-reachable
address, so this is tolerable for a local, throwaway stack.

It is still not something to copy into a deployed configuration. Each
service's DSN currently sits in its container's `environment:` block in
the compose file, which means it is visible to `docker inspect` on that
container — acceptable on a laptop, not acceptable once the stack is
deployed. **A deployed configuration must not inherit this pattern:**
secrets need a real secret store, not compose `environment:`, and deployed
passwords must not be the ones checked into this repository.

## Known limitations in M1

- No domain functionality: no login, no products, no orders, no staff or
  customer records. That arrives across M2–M4.
- No sessions, authentication, or principal signing — arrives in M2.
- No OpenTelemetry, metrics, or rate limiting — arrives in M7.
- No messaging (NATS, outbox) — arrives in M6.
- No Kubernetes — arrives in M8.
- The `platformcheck` package in each service and the `/_platform/*` routes
  are M1 scaffolding, not a stable API, and are deleted as each service's
  real capabilities land.
- CI (`.github/workflows/ci-services.yml`) builds, unit-tests, and
  integration-tests the four services on every PR, but has not yet run
  on GitHub Actions itself — only its individual commands have been run
  and verified locally.
