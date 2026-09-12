# Task 8 report: full stack in Compose, repointed Makefile

## Corrections applied (per brief overrides)

- Left `name: go-admin-platform` in `deploy/compose/docker-compose.yml` untouched — did not revert to `go-admin`.
- Did not touch the existing `postgres` service, its three roles/databases, its end-state healthcheck, or the `127.0.0.1:5433` bind. Only appended `identity-migrate`/`identity`, `catalog-migrate`/`catalog`, `orders-migrate`/`orders`, and `gateway` under `services:`.
- `gateway` has no migrate job and no `DATABASE_URL` — it owns no database.
- `test-integration` runs `go test -tags integration ./test/integration/...`, not the brief's untagged `./test/...` (which would have silently run 0 integration tests and reported `ok`). Verified the difference empirically — see below.
- `GO_PKGS` scoped to `./services/... ./platform/...`, excluding the repo root (legacy code, needs MySQL).

## Verifying the tag defect

```
$ go test ./test/integration/... -count=1
go: warning: "./test/integration/..." matched no packages
no packages to test
exit=1
```

Against the narrower target actually used in the Makefile, dropping the tag fails loudly (no packages matched), not silently. The silent-pass failure mode described in the brief's correction #3 applies to the wider `./test/...` path (which would resolve to `test/arch` only and report `ok`) — that's the reason the tag stays pinned to `./test/integration/...` specifically rather than a broader path. Documented as a comment in the Makefile.

## Boot sequence (from a genuinely clean state)

```
$ make down
$ docker compose -f deploy/compose/docker-compose.yml down -v
 Volume go-admin-platform_pgdata  Removing
 Volume go-admin-platform_pgdata  Removed
$ make up
... (build + up --wait) ...
```

`docker compose -f deploy/compose/docker-compose.yml ps -a`:

```
NAME                                   STATUS
go-admin-platform-catalog-1            Up
go-admin-platform-catalog-migrate-1    Exited (0)
go-admin-platform-gateway-1            Up
go-admin-platform-identity-1           Up
go-admin-platform-identity-migrate-1   Exited (0)
go-admin-platform-orders-1             Up
go-admin-platform-orders-migrate-1     Exited (0)
go-admin-platform-postgres-1           Up (healthy)
```

All three `*-migrate` jobs exited 0; `postgres`, `identity`, `catalog`, `orders`, `gateway` all up. The legacy `go-admin` project/container was never listed or touched by any `down`/`down -v` in this run — confirms correction #1 (project name `go-admin-platform`) holds.

## Manual curl check (Step 4), all three services through the gateway

```
$ curl -s -H 'X-Request-Id: manual-check' localhost:8080/_platform/identity
{"service":"identity","schemaVersion":1,"requestId":"manual-check"}

$ curl -s -H 'X-Request-Id: manual-check' localhost:8080/_platform/catalog
{"service":"catalog","schemaVersion":1,"requestId":"manual-check"}

$ curl -s -H 'X-Request-Id: manual-check' localhost:8080/_platform/orders
{"service":"orders","schemaVersion":1,"requestId":"manual-check"}
```

Correct service name, non-zero `schemaVersion`, and the same request ID echoed back, for all three services. This was re-verified after a second clean-volume `down -v` + `up` cycle (results identical).

## `make test`

Unit (`go test ./services/... ./platform/... ./test/arch/... -count=1`): all packages `ok` (20 test packages, one `[no test files]` for `gateway/internal/httperr`), no failures.

Integration (`go test -tags integration ./test/integration/... -count=1`): `ok`. Verified with `-v` that this is not a silent zero-test pass:

```
--- PASS: TestServiceRoleCannotReachAnotherServiceDatabase (0.01s)
--- PASS: TestServiceRoleCannotReachAnotherServiceDatabaseCrossCheck (0.01s)
--- PASS: TestServiceRoleCannotReachMaintenanceDatabase (0.01s)
--- PASS: TestServiceRoleReachesItsOwnDatabase (0.01s)
--- PASS: TestServiceRoleHasNoElevatedAttributes (0.03s)
    --- PASS: TestServiceRoleHasNoElevatedAttributes/identity_user
    --- PASS: TestServiceRoleHasNoElevatedAttributes/catalog_user
    --- PASS: TestServiceRoleHasNoElevatedAttributes/orders_user
PASS
ok  	github.com/loloDawit/go-admin/test/integration	0.310s
```

5 top-level tests (7 including subtests) actually ran and passed.

`make lint`: clean (gofmt + `go vet` on `GO_PKGS`/`test/arch` and, tagged, on `test/integration`).

## Legacy untouched

`docker-compose.yml`, `main.go`, `controllers/`, etc. at the repo root are unmodified. `git status --short` before commit showed only `Makefile` and `deploy/compose/docker-compose.yml` as modified.

## Concerns (not addressed, flagged for awareness)

- The four Go services are distroless images with no `HEALTHCHECK`; `docker compose up --wait` treats them as ready once the container is *running*, not once the process is actually accepting connections — compose reports them "Healthy" but that's `service_started` semantics, not a real health probe. Only `postgres` has a genuine healthcheck. `gateway`'s `depends_on: [identity, catalog, orders]` is therefore also just "started," not "ready" — acceptable in practice because the reverse proxy dials lazily per-request (proven by the curl checks succeeding), but worth tightening with real healthchecks in a later milestone if flakiness shows up under slower cold starts.
- Service HTTP ports (8080-8083) bind `0.0.0.0`, asymmetric with postgres's `127.0.0.1:5433` bind. No credentials ride on the HTTP ports so this isn't a defect, just an asymmetry worth naming.
- `make dev`'s `logs -f gateway` is interactive/foregrounded and wasn't exercised non-interactively beyond confirming `up` + curl work; only `make up`, `make test`, and manual curls were run end-to-end.
