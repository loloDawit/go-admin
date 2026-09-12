# M1 — Platform Skeleton and Walking Skeleton

**Date:** 2026-09-11
**Status:** Draft for review
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
capability package appearing under `services/*/internal/` in this milestone is
out of scope (`ArchitectureDecisionClarifications.md` §5).

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
│   │       ├── config/
│   │       ├── routing/                service registry, reverse proxy
│   │       └── httperr/                client-facing error mapping
│   │
│   ├── identity/
│   │   ├── cmd/identity/main.go
│   │   ├── internal/
│   │   │   ├── config/
│   │   │   ├── httperr/                sentinel -> {code,message,status}
│   │   │   └── platformcheck/          the walking-skeleton capability
│   │   ├── migrations/
│   │   ├── openapi/identity.yaml
│   │   └── Dockerfile
│   │
│   ├── catalog/                        same shape
│   └── orders/                         same shape
│
├── platform/                           shared TECHNICAL infrastructure only
│   ├── observability/                  slog setup, log fields
│   ├── requestid/                      generate, propagate, read
│   ├── httpx/                          error envelope, JSON helpers, middleware
│   └── pgx/                            pool construction, health probe
│
├── deploy/compose/
│   ├── docker-compose.yml
│   └── postgres/init/                  db + role creation, isolated grants
│
├── apps/web/                           empty until M5
├── Makefile
└── docs/
```

**One root `go.mod`** (`ArchitectureDecisionClarifications.md` §4). A single module
is a dependency-management choice, not a statement that the services are one
application: each remains a separate binary, image, migration set, and database
owner.

**Everything inside a service lives under `internal/` except `cmd/`.** Go's own
rule then makes `services/identity/internal/...` importable only from under
`services/identity/`. Catalog *cannot compile* against Identity's packages. The
boundary is enforced by the compiler, not by convention.

`platform/` holds technical infrastructure only. No business or domain model, and
no permission list.

---

## 3. Service anatomy

Every service is the same shape, so the third is boring to build. Within a
capability the flow is handler → service → repository
(`GoApplicationArchitectureandCodingConventions.md` §1), with all of a
capability's files in one package (§11).

M1 creates exactly one capability package per service: `platformcheck`. It is
deliberately not a domain concept — it is the walking skeleton, and it is
deleted or absorbed when the service gets real capabilities in M2–M4.

```
internal/platformcheck/
├── platformcheck.go     domain: SchemaVersion, sentinel errors
├── handler.go           HTTP boundary
├── service.go           orchestration
├── repository.go        interface, owned by the consumer
└── postgres.go          implementation
```

---

## 4. The walking skeleton

```
GET /api/v1/{service}/_platform
```

routed by the gateway to the named service, which reads its **own** database and
returns:

```json
{
  "service": "identity",
  "schemaVersion": 1,
  "requestId": "01JB..."
}
```

`schemaVersion` is read from `golang-migrate`'s `schema_migrations` table. That
choice is deliberate: it is a real query, against a table only migrations create,
using credentials scoped to that one database. A successful response proves
migrations ran, the pool works, and the credentials are right — without inventing
a fake domain entity.

What the path proves, per `ArchitectureDecisionClarifications.md` §5:

| Concern | Proven by |
|---|---|
| Gateway routing | request reaches the right service |
| Container networking | gateway resolves services by compose DNS |
| Request ID propagation | `requestId` in the body matches the response header, and appears in both services' logs |
| Configuration | services fail to boot on missing config |
| Database credentials | query succeeds as that service's own role |
| Service-owned DB access | credential-isolation test (§7) |
| Migrations | `schema_migrations` exists and has a version |
| Standardized HTTP errors | DB unreachable yields the standard envelope, not a stack trace |
| End-to-end networking | the test drives the real running stack |

---

## 5. Gateway (M1 subset)

`services/gateway` is a thin Go service, not proxy configuration
(`SmallScaleMicroservicesArchitecture.md` §8.1).

In M1 it does only:

* originate a request ID when absent; propagate it downstream and echo it back
* route `/api/v1/{service}/*` to the matching upstream by config
* return the standard error envelope for unknown routes and unreachable upstreams
* enforce a per-request deadline on every upstream call

**Not in M1:** session validation, principal signing, rate limiting, auth of any
kind. Those arrive in M2 with Identity, because there is nothing to authenticate
against until then.

---

## 6. Database topology

One PostgreSQL container, three databases, three roles
(`SmallScaleMicroservicesArchitecture.md` §6.2):

```
postgres
├── identity_db   owner: identity_user
├── catalog_db    owner: catalog_user
└── orders_db     owner: orders_user
```

`deploy/compose/postgres/init/` creates each database and role, grants each role
rights only to its own database, and **revokes** the default `PUBLIC` connect
privilege on the others. Each service's DSN carries only its own credentials.

Migrations run per service via `golang-migrate`, as a compose init step, not from
application code. A service never migrates another service's database.

---

## 7. Tests

M1's tests are about topology, not behaviour.

**Integration smoke (the definition of done).** Boot the stack; `GET` the
skeleton path for each of the three services through the gateway; assert 200, the
correct `service`, a non-zero `schemaVersion`, and that the `requestId` echoed
back matches the one sent.

**Credential isolation.** Connect as `identity_user`; assert connecting to
`catalog_db` is refused. The PRD's M1 says *"verify technically that each service
cannot access another service's database"* — this is what that means, and it must
be a test rather than an assertion in a document.

**Import boundary.** Walk the import graph; fail if any file under
`services/<a>/` imports `services/<b>/`. Go's `internal/` rule already prevents
this; the test catches someone "fixing" a compile error by moving a package out
of `internal/`.

**Error envelope.** Point a service at a dead database; assert the response is
the standard `{code, message}` with no driver text, SQL, or host name.

**Config fail-fast.** Each service refuses to start on missing or invalid
configuration, per the convention carried from M0.

**Request ID propagation.** Assert the ID appears in the gateway's log line and
the service's log line for the same request.

---

## 8. Error model

Per `ArchitectureDecisionClarifications.md` §6:

* domain and service layers return sentinel errors, with no HTTP knowledge
* the HTTP boundary owns the mapping to `{code, message, status}`, one place per
  service, in `internal/httperr`
* `platform/httpx` supplies the envelope type and the writer, never the catalogue

The AST guard from M0 is re-added in its **narrowed** form: it fails when an
`http`-layer package constructs a client-facing error outside the central
mapping. It does not ban `errors.New` in domain code.

---

## 9. Observability (M1 subset)

Structured logging via `log/slog`, with `request_id`, `service`, `route`,
`status`, and `duration` on every request log line.

**OpenTelemetry is M7.** M1 lays the request-ID groundwork that tracing later
builds on, and nothing more.

---

## 10. CI

One workflow, matrixed over the four services: `gofmt`, `go vet`, `go build`,
`go test`. The integration smoke test runs against a booted compose stack. CI
must fail if any service's Docker image fails to build.

---

## 11. Definition of done

```bash
git clone
make dev     # whole stack boots: gateway, identity, catalog, orders, postgres
make test    # all tests, including the integration smoke
```

* the walking skeleton returns 200 for all three services through the gateway
* a request ID set by the client survives to the service's log and back
* credential isolation is proven by a passing test
* the import-boundary test passes
* each service has `/healthz`, `/readyz`, a Dockerfile, migrations, and an
  OpenAPI file
* CI builds and tests every service
* no domain functionality exists

---

## 12. Explicitly not in M1

Deferred with the milestone that owns each, so none of these leaks in early:

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

---

## 13. Open question

**Does the gateway resolve upstreams from static config, or from a small service
registry?** Static config (an env var per service) is simpler and sufficient for
three fixed services; a registry is a distributed-systems concept worth
practising but adds a dependency M1 does not otherwise need. Recommendation:
**static config in M1**, revisit if M8's Kubernetes work makes it interesting.
