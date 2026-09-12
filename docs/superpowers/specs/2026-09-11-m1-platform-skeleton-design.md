# M1 — Platform Skeleton and Walking Skeleton

**Date:** 2026-09-11
**Status:** Approved
**Milestone:** PRD M1 (`docs/PRD/SmallScaleMicroservicesArchitecture.md` §16)

**Governing documents**

| Decision | Source |
|---|---|
| Milestone scope, service boundaries, gateway, auth | `SmallScaleMicroservicesArchitecture.md` |
| Go structure inside a service | `GoApplicationArchitectureandCodingConventions.md` |
| Precedence and the resolved ambiguities | `ArchitectureDecisionClarifications.md` |

---

## 1. Goal

Stand up the service topology and prove it works end to end, before any business
behaviour exists.

M1 succeeds when `make dev` boots the whole stack and an automated test drives a
real request through the gateway, into a service, into that service's own
database, and back — with a request ID that survives the whole path.

**M1 implements no domain functionality.** No login, no product, no order. A
domain capability package appearing under `services/*/internal/` in this
milestone is out of scope (`ArchitectureDecisionClarifications.md` §5).

Milestone ownership is unchanged:

```text
M1 -> platform + technical walking skeleton
M2 -> Identity
M3 -> Catalog
M4 -> Orders + first meaningful Orders -> Catalog business workflow
```

---

## 2. Repository layout

```
go-admin/
├── go.mod                              one root module
│
├── services/
│   ├── gateway/
│   │   ├── cmd/gateway/main.go
│   │   └── internal/
│   │       ├── config/                 upstream URLs, timeouts
│   │       ├── routing/                fixed route -> upstream mapping
│   │       └── httperr/                client-facing error mapping
│   │
│   ├── identity/
│   │   ├── cmd/identity/main.go
│   │   ├── internal/
│   │   │   ├── config/
│   │   │   ├── httperr/                sentinel -> {code,message,status}
│   │   │   └── platformcheck/          TEMPORARY — see §4
│   │   ├── migrations/
│   │   ├── openapi/identity.yaml
│   │   └── Dockerfile
│   │
│   ├── catalog/                        same shape
│   └── orders/                         same shape
│
├── platform/                           shared TECHNICAL infrastructure only
│   ├── observability/                  slog setup, log attributes
│   ├── requestid/                      generate, propagate, read
│   ├── httpx/                          error envelope, JSON helpers, middleware
│   └── pgx/                            pool construction, health probe
│
├── deploy/compose/
│   ├── docker-compose.yml
│   └── postgres/init/                  databases, roles, grants
│
├── apps/web/                           empty until M5
├── Makefile
└── docs/
```

**One root `go.mod`** (`ArchitectureDecisionClarifications.md` §4). A single module
is a dependency-management choice, not a statement that the services are one
application: each remains a separate binary, image, migration set, and database
owner.

**All service implementation Go packages live under `internal/`, except
executable entry points under `cmd/`.** Non-Go assets — `migrations/`,
`openapi/`, `Dockerfile` — sit alongside them at the service root.

That placement is what makes the boundary compiler-enforced: Go's own rule means
`services/identity/internal/...` is importable only from under
`services/identity/`, so Catalog *cannot compile* against Identity's packages.

`platform/` holds technical infrastructure only. No business or domain model, and
no permission list.

---

## 3. Service anatomy

Every service is the same shape, so the third is boring to build. Within a
capability the flow is handler → service → repository
(`GoApplicationArchitectureandCodingConventions.md` §1), with all of a
capability's files in one package (§11).

M1 creates exactly one package per service — `platformcheck` — and it is
deliberately not a domain concept.

```
internal/platformcheck/
├── platformcheck.go     SchemaState type, sentinel errors
├── handler.go           HTTP boundary
├── service.go           orchestration
├── repository.go        interface, owned by the consumer
└── postgres.go          implementation
```

---

## 4. The walking skeleton

### Routes

The skeleton is **infrastructure, not product API**, and its routing says so.
Putting it under `/api/v1/{service}/...` would leak deployment topology into the
product contract — the API should not tell a client which service answered.

```text
gateway                     upstream
────────────────────────    ──────────────────────
GET /_platform/identity  →  identity:  GET /_platform
GET /_platform/catalog   →  catalog:   GET /_platform
GET /_platform/orders    →  orders:    GET /_platform
```

Each service knows only its own local `/_platform`. The `/_platform/*` prefix is
reserved for infrastructure and is never used for product endpoints.

### Response

```json
{
  "service": "identity",
  "schemaVersion": 3,
  "requestId": "01JB..."
}
```

`schemaVersion` is read from `golang-migrate`'s `schema_migrations` table using
the service's own credentials. That choice is deliberate: it is a real query
against a table only migrations create, so a 200 proves migrations ran, the pool
works, and the credentials are correctly scoped — without inventing a fake domain
entity that would then need deleting.

### Dirty migration state is a failure

`schema_migrations` carries both `version` and `dirty`. **A non-zero version is
not sufficient.** A dirty row means a migration failed partway and the schema is
in an unknown state.

```text
version > 0 AND NOT dirty   -> ready
dirty                       -> NOT ready; /_platform returns 503
no row / version 0          -> NOT ready; /_platform returns 503
```

The service must understand `dirty`; the API need not expose it. It belongs in
the log line and in readiness, not in the response body.

### This code is temporary

`platformcheck` and the `/_platform/*` routes exist to prove the topology and are
**removed or absorbed as each service gains its real capability**:

| Milestone | Disposition |
|---|---|
| M2 Identity | `platformcheck` removed from Identity once `staff`/`session` land |
| M3 Catalog | removed from Catalog once `product` lands |
| M4 Orders | removed from Orders once `order` lands |

Readiness checking survives; the *route* and the *package* do not. Nothing in
M2–M4 should build on `platformcheck`.

### What the path proves

| Concern | Proven by |
|---|---|
| Gateway routing | request reaches the right service |
| Container networking | gateway resolves upstreams by configured URL |
| Request ID propagation | echoed header matches, and appears in both services' structured logs |
| Configuration | services refuse to boot on missing config |
| Database credentials | query succeeds as that service's own role |
| Service-owned DB access | credential-isolation test (§8) |
| Migrations | `schema_migrations` present, versioned, and clean |
| Standardized HTTP errors | unreachable database yields the standard envelope |
| End-to-end networking | the test drives the real running stack |

---

## 5. Gateway (M1 subset)

`services/gateway` is a thin Go service, not proxy configuration
(`SmallScaleMicroservicesArchitecture.md` §8.1).

In M1 it does only:

* originate a request ID when absent; propagate it downstream and echo it back
* route by a **fixed** mapping to statically configured upstreams (§13)
* return the standard error envelope for unknown routes and unreachable upstreams
* enforce a per-request deadline on every upstream call

**Not in M1:** session validation, principal signing, rate limiting, or auth of
any kind. Those arrive in M2 with Identity, because there is nothing to
authenticate against until then.

The gateway owns **no database** and therefore has **no migrations**.

---

## 6. Startup and availability contract

Configuration errors and dependency outages are different failures and must
behave differently:

```text
missing / invalid DB configuration
    -> process refuses to start

valid configuration, database temporarily unavailable
    -> process remains running
    -> /healthz   = 200   (the process is healthy)
    -> /readyz    = 503   (it cannot serve)
    -> /_platform = 503   (standard error envelope)
```

A service must **not** exit because Postgres is briefly unavailable. Exiting
turns a recoverable outage into a restart loop; staying up lets the service
recover on its own when the database returns.

`/healthz` answers "is this process alive" and never touches the database.
`/readyz` answers "can this process serve", and therefore checks both
connectivity and clean migration state (§4).

---

## 7. Database topology

One PostgreSQL container, three databases, three roles
(`SmallScaleMicroservicesArchitecture.md` §6.2):

```
postgres
├── identity_db   owner: identity_user
├── catalog_db    owner: catalog_user
└── orders_db     owner: orders_user
```

`deploy/compose/postgres/init/` must:

* create each database and role
* create every service role as `NOSUPERUSER NOCREATEDB NOCREATEROLE` — a
  superuser role makes every other restriction decorative
* grant each role only the privileges it needs on its **own** database
* `REVOKE CONNECT ON DATABASE <other> FROM PUBLIC` so a role cannot reach a
  sibling database through the default `PUBLIC` grant

Each service's DSN carries only its own credentials.

Migrations run per service via `golang-migrate` as a compose step, not from
application code. A service never migrates another service's database.

---

## 8. Tests

M1's tests are about topology, not behaviour.

**Integration smoke (the definition of done).** Boot the stack; `GET
/_platform/{service}` for each of the three services through the gateway; assert
200, the correct `service`, a non-zero `schemaVersion`, and that the `requestId`
echoed back matches the one sent.

**Credential isolation.** Connect as `identity_user`; assert connecting to
`catalog_db` is refused. Assert the role is not a superuser and cannot create
databases or roles. The PRD says *"verify technically that each service cannot
access another service's database"* — this is what that means, and it must be a
test rather than a claim in a document.

**Import boundary.** Walk the import graph; fail if any file under
`services/<a>/` imports `services/<b>/`. Go's `internal/` rule is the primary
enforcement and makes this redundant *today* — the test's job is to catch someone
weakening the layout later (moving a package out of `internal/` to resolve a
compile error) and thereby removing the compiler's protection.

**Dirty migration state.** Mark `schema_migrations.dirty`; assert `/readyz` and
`/_platform` both fail, and that a clean state passes. A version-only check must
not satisfy this test.

**Error envelope.** Point a service at an unreachable database; assert 503 with
the standard `{code, message}` and no driver text, SQL, host name, or credential.

**Config fail-fast.** Each service refuses to start on missing or invalid
configuration.

**Request ID propagation** — two tests, neither of which scrapes logs:

1. *Through HTTP*: send a known request ID to the gateway; assert the same ID is
   echoed back and appears in the upstream's response body.
2. *In structured logs*: install a captured `slog.Handler` in-process and assert
   the expected attributes (`request_id`, `service`, `route`, `status`,
   `duration`) are present with the right values.

Grepping human-formatted container logs is flaky and couples the test to log
rendering. Do not do it.

---

## 9. Error model

Per `ArchitectureDecisionClarifications.md` §6:

* domain and service layers return sentinel errors, with no HTTP knowledge
* the HTTP boundary owns the mapping to `{code, message, status}`, one place per
  service, in `internal/httperr`
* `platform/httpx` supplies the envelope type and the writer, never the catalogue

The AST guard from M0 is re-added in its **narrowed** form: it fails when an
`http`-layer package constructs a client-facing error outside the central
mapping. It does not ban `errors.New` in domain code.

---

## 10. Observability (M1 subset)

Structured logging via `log/slog`, with `request_id`, `service`, `route`,
`status`, and `duration` on every request log line.

**OpenTelemetry is M7.** M1 lays the request-ID groundwork tracing later builds
on, and nothing more.

---

## 11. CI

One workflow, matrixed over the four services: `gofmt`, `go vet`, `go build`,
`go test`. The integration smoke test runs against a booted compose stack. CI
fails if any service's Docker image fails to build.

---

## 12. Definition of done

```bash
git clone
make dev     # whole stack boots: gateway, identity, catalog, orders, postgres
make test    # all tests, including the integration smoke
```

Per-service requirements differ, and the checklist must not invent work to make
them uniform:

| | Gateway | Identity / Catalog / Orders |
|---|---|---|
| `/healthz`, `/readyz` | required | required |
| Dockerfile | required | required |
| OpenAPI | where applicable | required |
| Migrations | **none** — owns no database | required |
| Database | none | its own, isolated |

Do not create empty gateway migrations to satisfy a uniform checklist.

Also required:

* `/_platform/{service}` returns 200 for all three services through the gateway
* a client-supplied request ID survives to the service's structured log and back
* credential isolation proven by a passing test, including the role attributes
* dirty migration state fails readiness, proven by a test
* the import-boundary test passes
* CI builds and tests every service
* no domain functionality exists

---

## 13. Decision — Gateway Upstream Resolution

Gateway upstreams are **statically configured**. Docker Compose supplies the
Identity, Catalog, and Orders base URLs through configuration:

```text
IDENTITY_URL
CATALOG_URL
ORDERS_URL
```

The gateway contains a fixed route-to-upstream mapping for the established
services. **No dynamic service registry is introduced.**

In Compose these resolve through Compose DNS. In Kubernetes later, the same
configured-upstream abstraction is satisfied by Kubernetes Service DNS —
Kubernetes does not by itself justify changing this decision.

Reconsider dynamic discovery only if a concrete future requirement introduces
dynamically changing service identities that cannot be represented by platform
DNS or configuration.

---

## 14. Explicitly not in M1

| Deferred | Milestone |
|---|---|
| Login, sessions, principal signing, authorization | M2 |
| Staff, roles, permissions | M2 |
| Products, images, `/internal/products/resolve` | M3 |
| Orders, customers, the Orders→Catalog dependency | M4 |
| Frontend beyond an empty `apps/web/` | M5 |
| NATS, outbox, any messaging | M6 |
| OpenTelemetry, metrics, dashboards, rate limiting | M7 |
| Kubernetes | M8 |
