# Task 12 report: README for the new stack

## Commands run to verify the README, and their actual output

### Clean-state teardown

```
$ docker compose -f deploy/compose/docker-compose.yml down -v
 Container go-admin-platform-gateway-1  Removed
 Container go-admin-platform-orders-1  Removed
 Container go-admin-platform-catalog-1  Removed
 Container go-admin-platform-catalog-migrate-1  Removed
 Container go-admin-platform-orders-migrate-1  Removed
 Container go-admin-platform-identity-1  Removed
 Container go-admin-platform-identity-migrate-1  Removed
 Container go-admin-platform-postgres-1  Removed
 Volume go-admin-platform_pgdata  Removed
 Network go-admin-platform_default  Removed
```

### `make up` (clean volume — Postgres init scripts actually ran)

```
$ make up
docker compose -f deploy/compose/docker-compose.yml up -d --build --wait
... builds catalog, identity, orders, gateway images ...
 Container go-admin-platform-postgres-1  Healthy
 Container go-admin-platform-identity-migrate-1  Exited
 Container go-admin-platform-orders-migrate-1  Exited
 Container go-admin-platform-catalog-migrate-1  Exited
 Container go-admin-platform-identity-1  Healthy
 Container go-admin-platform-catalog-1  Healthy
 Container go-admin-platform-orders-1  Healthy
 Container go-admin-platform-gateway-1  Healthy
```

All three migrate jobs exited 0 (compose only proceeds past
`condition: service_completed_successfully` on success); postgres,
identity, catalog, orders, gateway all reported healthy.

### Manual curl through the gateway, all three services

```
$ curl -s -i -H 'X-Request-Id: check' localhost:8080/_platform/identity
HTTP/1.1 200 OK
Content-Type: application/json
X-Request-Id: check

{"service":"identity","schemaVersion":1,"requestId":"check"}

$ curl -s -i -H 'X-Request-Id: check' localhost:8080/_platform/catalog
HTTP/1.1 200 OK
...
{"service":"catalog","schemaVersion":1,"requestId":"check"}

$ curl -s -i -H 'X-Request-Id: check' localhost:8080/_platform/orders
HTTP/1.1 200 OK
...
{"service":"orders","schemaVersion":1,"requestId":"check"}
```

Correct `service` name, non-zero `schemaVersion`, and the `X-Request-Id`
echoed back, for all three services.

### `make test`

```
$ make test
go test ./services/... ./platform/... ./test/arch/... -count=1
ok  	.../services/catalog/cmd/catalog	0.376s
ok  	.../services/catalog/internal/config	0.646s
ok  	.../services/catalog/internal/httperr	0.881s
ok  	.../services/catalog/internal/platformcheck	1.174s
ok  	.../services/gateway/cmd/gateway	1.445s
ok  	.../services/gateway/internal/config	1.608s
?   	.../services/gateway/internal/httperr	[no test files]
ok  	.../services/gateway/internal/routing	1.917s
ok  	.../services/identity/cmd/identity	2.127s
ok  	.../services/identity/internal/config	2.340s
ok  	.../services/identity/internal/httperr	2.647s
ok  	.../services/identity/internal/platformcheck	2.794s
ok  	.../services/orders/cmd/orders	3.054s
ok  	.../services/orders/internal/config	3.145s
ok  	.../services/orders/internal/httperr	3.177s
ok  	.../services/orders/internal/platformcheck	3.162s
?   	.../platform/healthcheck	[no test files]
ok  	.../platform/httpx	3.141s
ok  	.../platform/observability	3.106s
ok  	.../platform/pgx	3.191s
ok  	.../platform/requestid	3.088s
ok  	.../test/arch	3.113s
... docker compose up -d --build --wait (re-boots the stack) ...
go test -tags integration ./test/integration/... -count=1
ok  	github.com/loloDawit/go-admin/test/integration	0.411s
```

All unit/arch packages `ok`; integration suite `ok` against the live stack.

### Verifying the tag-gating claims in the README

```
$ go test ./test/integration/... -count=1
go: warning: "./test/integration/..." matched no packages
no packages to test
exit status: 1
```

Confirms: without `-tags integration`, the exact `test/integration` path
fails outright (matches no packages) rather than passing silently.

```
$ go test ./test/... -count=1
ok  	github.com/loloDawit/go-admin/test/arch	0.328s
```

Confirms: without the tag, a wider path (`./test/...`) reports plain `ok`
— it silently ran 0 integration tests (only `test/arch`'s untagged files
matched) rather than failing. This is exactly why `make test-integration`
pins `-tags integration` to `./test/integration/...` specifically.

### Facts checked directly against the repo before writing them into the README

- `Makefile`: read in full for the target table and `GO_PKGS` scoping
  (`./services/... ./platform/...`, root excluded because legacy code
  needs MySQL).
- `deploy/compose/docker-compose.yml`: read in full for ports, service
  names, `DATABASE_URL`s, and the `127.0.0.1:5433` postgres bind. Confirmed
  `gateway` has no migrate job and no database of its own; confirmed
  `services/gateway` has no `migrations` directory (`find services/gateway
  -iname migrations` returned nothing).
- `deploy/compose/postgres/init/01-roles-and-databases.sql`: read in full.
  Confirms each of `identity_user`/`catalog_user`/`orders_user` is created
  `NOSUPERUSER NOCREATEDB NOCREATEROLE`, and that `CONNECT` on every
  database is revoked from `PUBLIC` and granted back only to that
  database's own role — the basis for the "cannot reach a sibling
  database" claim in the README.
- `go.mod`: `go 1.27`, confirming the requirements section.
- `.github/workflows/ci-services.yml`: read in full for the CI description
  (unit job + integration job, both matrix-free, building/testing all four
  services).
- `docs/ASSESSMENT.md` and `docs/PRD/SmallScaleMicroservicesArchitecture.md`
  confirmed present at the paths linked from the README.
- Checked the PRD for an explicit per-service M2/M3/M4 mapping of
  `/_platform` removal — none found stated that granularly, so the README
  says "removed service by service as each one gains its real capabilities
  in M2–M4" rather than asserting a specific service-to-milestone pairing
  that isn't written down anywhere.
- `git tag` confirms `legacy-v1` exists.

## Corrections made after a first draft (self-review before commit)

- Removed session-relative language ("verified in this session") from the
  quick-start and CI-limitations sections — replaced with plain statements
  of fact, since a reader of the README wasn't in this session.
- Rewrote the `go test ./...` paragraph, which in the first draft inverted
  which case is the silent pass vs. the hard failure; re-ran both commands
  above to pin down the correct direction before rewriting.
- Added "not a migration source" to the legacy-application section (named
  explicitly in the task brief, missing from the first draft).
- Added the missing `make help` target to the make-target table.
- Fixed the postgres-network wording: other containers on the compose
  network *can* reach postgres by design (that's how the services connect
  to it) — the actual isolation fact is the `127.0.0.1` host bind, not
  network isolation.
- Restructured the credentials section so the safety rationale (scoped
  roles + host-only bind) and the deploy warning (docker-inspect
  visibility, don't copy this pattern) aren't run together as one bullet
  list.
- Backed the "restricted role" and "cannot reach a sibling database" claim
  with the actual init SQL rather than paraphrasing from the task brief.

## Legacy untouched

No files at the repo root (`main.go`, `controllers/`, `models/`,
`clients/`, root `docker-compose.yml`) were modified. Only `README.md` was
changed.

## Concerns

- The README states CI "has not yet run on GitHub Actions itself" — this
  is accurate as of this session (only local command-by-command
  verification was done) but will go stale the first time a PR actually
  runs `ci-services.yml`; whoever merges the first PR against this branch
  should update that line.
- The per-service `/_platform` removal mapping (which milestone removes
  which service's scaffolding) is asserted only at the M2–M4 range, not a
  specific service-to-milestone pairing, because the PRD doesn't commit to
  one at that granularity as far as I found. If a more specific mapping
  exists elsewhere, the README should be tightened.
