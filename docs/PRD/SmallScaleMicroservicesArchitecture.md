# go-admin Revival PRD

**Status:** Draft
**Date:** 2026-09-11
**Project:** `go-admin`
**Document purpose:** Define the product and engineering requirements for reviving the original `go-admin` project while intentionally evolving it into a small-scale microservices system.

---

# 1. Executive Summary

`go-admin` is a 2022 Go + React tutorial-derived back-office application for a small e-commerce shop. The original repository assessment concluded that the implementation should be rebuilt while preserving the useful domain decisions already present in the project.

The original architecture recommendation was:

> **Modular monolith, one deployable binary, one database. No microservices, no event bus, no CQRS.**

That recommendation remains correct if the only objective is to build the simplest production-quality admin application.

However, the project now has a second explicit objective:

> **Use `go-admin` as a practical environment for designing, implementing, testing, deploying, and operating a small microservice architecture.**

This changes the architectural constraint.

The goal is **not** to maximize the number of services or recreate large-enterprise infrastructure. The goal is to introduce only enough distribution to exercise meaningful service boundaries and distributed-system concerns while keeping the application small enough to complete.

The target architecture will therefore use:

* **3 business microservices**

  * Identity
  * Catalog
  * Orders
* React admin frontend
* One API entry point / reverse proxy
* PostgreSQL
* Separate service-owned databases or schemas with isolated credentials
* Synchronous HTTP communication first
* Asynchronous messaging introduced later where justified
* Docker Compose as the initial runtime
* Kubernetes only as a later optional deployment target
* OpenTelemetry and production-style observability after the core system works

The existing domain intent remains the foundation. The implementation will be rebuilt.

---

# 2. Background

The repository assessment established several important facts:

* The application is a staff-facing e-commerce back-office.
* Its main entities are:

  * User
  * Role
  * Permission
  * Product
  * Order
  * OrderItem
* The original implementation contains severe authentication and authorization defects.
* The application cannot be reliably started from a clean clone.
* There are no tests.
* There is no production data or deployed environment requiring migration compatibility.
* The frontend is largely unfinished.
* The existing architecture has no service or repository boundary.
* HTTP handling, business logic, and persistence are mixed inside controllers.
* Package-level mutable globals make testing difficult.
* The current database and API contracts can be replaced without migration risk.

The repository nevertheless contains several useful decisions that should survive:

1. Product, Order, OrderItem, User, Role, and Permission are appropriate domain concepts.
2. `OrderItem` snapshots product title and purchase price.
3. Authorization is permission-based rather than relying only on role names.
4. Read and write permissions are distinct.
5. bcrypt and HTTP-only cookies were directionally correct.
6. The existing route map is useful as a record of intended product functionality.

The rebuild should preserve these **domain decisions**, not necessarily the existing source files.

---

# 3. Product Vision

Build a complete staff-facing commerce administration application that is also a realistic small-scale microservices reference implementation.

The product should allow staff to:

* authenticate securely;
* manage staff roles and permissions;
* manage products;
* manage customers;
* create and manage orders;
* progress orders through a controlled lifecycle;
* inspect revenue and operational metrics;
* search and filter operational data;
* review audit history;
* operate the application locally or in a deployed environment.

The engineering system should additionally demonstrate:

* bounded service ownership;
* database ownership;
* synchronous service-to-service communication;
* timeouts and cancellation;
* request correlation;
* distributed tracing;
* dependency failure handling;
* service contracts;
* transactional event publication;
* idempotent asynchronous consumers;
* health and readiness checks;
* independent service migrations;
* containerized deployment.

---

# 4. Goals

## 4.1 Product goals

The finished product must provide:

* secure staff authentication;
* role and permission administration;
* product CRUD and catalog state management;
* customer records;
* order creation and management;
* immutable historical order-line snapshots;
* a defined order lifecycle;
* inventory-aware fulfillment;
* revenue reporting;
* audit history;
* usable administration screens for all core capabilities.

## 4.2 Engineering goals

The project must provide practical experience implementing real microservice concerns at small scale.

Specifically, the system should eventually demonstrate:

* service ownership;
* isolated persistence;
* API contracts;
* synchronous inter-service calls;
* resilience to dependency failures;
* eventual consistency;
* transactional outbox;
* idempotent message handling;
* observability across service boundaries;
* independent migrations;
* containerized deployment.

## 4.3 Learning constraint

Microservices are an intentional project requirement even where a modular monolith would be operationally simpler.

The implementation must nevertheless avoid unnecessary distribution.

The architecture should be the **smallest system that meaningfully behaves like microservices**.

---

# 5. Non-Goals

The following are explicitly out of scope during the initial milestones:

* dozens of independently deployed services;
* service-per-database-table architecture;
* Kafka as an initial dependency;
* Kubernetes as the initial development runtime;
* CQRS;
* event sourcing;
* distributed transactions;
* custom service mesh infrastructure;
* multi-region deployment;
* independent frontend applications per service;
* multiple backend languages;
* multi-tenant SaaS architecture.

The following should also **not** become separate services initially:

* Customers
* Payments
* Reporting
* Permissions
* Images
* Inventory
* Notifications

These capabilities should remain inside the closest business service until there is a clear reason to extract them.

---

# 6. Architecture Principles

## 6.1 Service boundaries represent business capabilities

A service must not exist simply because a table or model exists.

A new service must have a defensible reason based on one or more of:

* business capability;
* data ownership;
* independent lifecycle;
* failure isolation;
* scalability need;
* security boundary;
* independently useful deployment behavior.

Before creating another service, ask:

> **What business capability, ownership boundary, deployment requirement, or failure boundary does this extraction create?**

If the answer is only:

> "It is a different entity/table,"

the capability should remain inside its current service.

---

## 6.2 Every service owns its data

Services must not directly query another service's tables.

Allowed:

```text
orders-service -> HTTP -> catalog-service
```

Not allowed:

```text
orders-service -> SELECT ... FROM catalog.products
```

The initial physical database topology may use one PostgreSQL server:

```text
PostgreSQL
├── identity_db
├── catalog_db
└── orders_db
```

Each service must use credentials that provide access only to its own database.

A single physical PostgreSQL instance is acceptable.

Shared logical ownership is not.

---

## 6.3 Historical facts belong to the domain that needs them

Orders must not depend on mutable Catalog data to reconstruct historical orders.

Example:

Catalog currently contains:

```text
Product ID: 287
Title: Nike Air Max
Price: $149.99
```

When an order is placed, Orders persists:

```text
product_id:    287
product_title: Nike Air Max
unit_price:    14999
quantity:      2
```

If Catalog later changes the product to `$169.99`, the historical order remains `$149.99`.

This preserves the correct decision already present in the original `OrderItem` model.

---

## 6.4 Prefer synchronous communication before asynchronous communication

The first distributed workflows should use HTTP.

Example:

```text
Orders
   |
   | POST /internal/products/resolve
   v
Catalog
```

This intentionally exposes the project to important distributed-system concerns:

* dependency timeouts;
* request cancellation;
* retry decisions;
* latency;
* partial failure;
* contract compatibility;
* correlation IDs;
* dependency health.

Messaging should be introduced only after a clear asynchronous use case exists.

---

## 6.5 Avoid shared domain libraries

The monorepo may contain shared infrastructure utilities such as:

```text
platform/
├── observability/
├── requestid/
└── httputil/
```

It should not contain a common domain model such as:

```text
shared/models/Product
shared/models/Order
shared/models/User
```

Each service owns its own representation.

For example:

```text
Catalog.ProductResponse
```

and:

```text
Orders.ProductSnapshot
```

are intentionally different concepts.

### The permission vocabulary is a contract, not a shared model

This rule appears to collide with §9: Identity *issues* permissions that Catalog
and Orders *enforce*, so all three services must agree on the strings
`view_orders`, `edit_products`, and the rest. Duplicating them by hand drifts
silently; a shared domain package would breach this section.

The resolution is that the vocabulary is part of Identity's **published
contract**, not its domain model:

1. Identity owns the canonical permission list and publishes it in its OpenAPI
   specification.
2. Each consuming service **generates** typed constants from that specification
   at build time. No service hand-writes a permission string.
3. CI fails when a service's generated constants drift from Identity's
   published contract.

A service therefore depends on Identity's *contract*, which is allowed, rather
than on Identity's *code*, which is not. Adding a permission is a contract
change that CI surfaces to every consumer — which is the behaviour this
architecture exists to practise.

`platform/` may hold the codegen tooling. It must not hold the list itself.

---

# 7. Target System Architecture

```text
                        ┌──────────────────────┐
                        │    React Admin UI    │
                        │   Vite + React/TS    │
                        └──────────┬───────────┘
                                   │
                                   ▼
                        ┌──────────────────────┐
                        │ Reverse Proxy / API  │
                        │       Gateway        │
                        └─────┬─────┬─────┬────┘
                              │     │     │
                    ┌─────────┘     │     └─────────┐
                    │               │               │
             ┌──────▼───────┐ ┌────▼────────┐ ┌────▼─────────┐
             │ Identity     │ │ Catalog     │ │ Orders       │
             │ Service      │ │ Service     │ │ Service      │
             │              │ │             │ │              │
             │ users        │ │ products    │ │ customers    │
             │ roles        │ │ images      │ │ orders       │
             │ permissions  │ │ status      │ │ order items  │
             │ sessions     │ │ inventory*  │ │ lifecycle    │
             │ reset tokens │ │             │ │ payments*    │
             └──────┬───────┘ └────┬────────┘ │ audit trail  │
                    │               │          └────┬─────────┘
                    ▼               ▼               ▼
                identity_db     catalog_db      orders_db

                       one PostgreSQL instance initially
```

`*` indicates a capability that may be introduced later but remains inside the owning service initially.

---

# 8. Service Responsibilities

## 8.1 API Gateway

The gateway is a **thin Go service**, not proxy configuration.

It is written rather than configured because it performs real logic that a
config-only proxy cannot do safely:

* originates the request ID and trace context for every inbound request;
* terminates authentication once per request (§9);
* injects a **signed** principal for downstream services;
* routes to Identity, Catalog, and Orders;
* applies login rate limiting at the edge.

It owns no business data and no database.

It must stay thin. Business rules, validation, and authorization *decisions*
belong in the owning service; the gateway establishes **who** the caller is, not
**what** they may do.

---

## 8.2 Identity Service

Owns:

* staff users;
* authentication;
* password hashes;
* sessions;
* password reset;
* roles;
* permissions;
* role assignments;
* account activation/deactivation.

Example API responsibilities:

```text
POST /login
POST /logout

GET  /me

GET  /staff
POST /staff
...

GET  /roles
POST /roles
...

POST /internal/sessions/validate
```

Identity should fail closed.

If authorization cannot be established, write operations must not continue.

---

## 8.3 Catalog Service

Owns:

* products;
* product title/description;
* current price;
* image references;
* product state;
* search;
* catalog pagination;
* inventory when inventory is introduced.

Example APIs:

```text
GET    /products
POST   /products
GET    /products/{id}
PUT    /products/{id}
DELETE /products/{id}

POST /internal/products/resolve
```

The internal resolve operation should support batching.

Example:

```json
{
  "productIds": [287, 412, 921]
}
```

This avoids one service call per line item.

---

## 8.4 Orders Service

Owns:

* customers;
* orders;
* order items;
* order totals;
* order lifecycle;
* payment records;
* refunds;
* order audit history;
* reporting based on orders;
* revenue aggregation.

Orders references Catalog products through their identifiers but owns the historical snapshot used for the order.

The Orders database must never join directly against the Catalog database.

---

# 9. Authentication Model

The initial target uses server-side sessions rather than application JWTs, so
that a compromised session can be revoked immediately.

The browser receives an HTTP-only session cookie.

## 9.1 Authentication is terminated once, at the gateway

Naively, every service validates the session by calling Identity. That makes
Identity a synchronous dependency of **every request in the system** and
multiplies the call for every service a request touches.

Instead the gateway resolves the caller once and passes the result down:

```text
Browser
   |  Cookie: session=abc123
   v
Gateway ──────────────►  Identity   POST /internal/sessions/validate
   |   ◄──────────────   principal
   |
   |  X-Principal: {...}
   |  X-Principal-Signature: ...
   v
Identity | Catalog | Orders        (verify signature; never call Identity)
```

The resolved principal:

```json
{
  "userId": "usr_123",
  "permissions": ["view_orders", "edit_orders"],
  "issuedAt": "2026-09-11T19:04:00Z",
  "expiresAt": "2026-09-11T19:04:30Z"
}
```

## 9.2 Rules

1. **Only the gateway talks to Identity for session validation.** A downstream
   service that calls Identity on the request path is a defect.
2. **The principal is signed** (HMAC with a shared secret, or an asymmetric
   signature). Downstream services verify it and reject anything unsigned.
   Without this, anyone who reaches a service directly can forge a header and
   assume any identity — network isolation alone is not the control.
3. **The principal is short-lived** — seconds, not the session lifetime — so a
   captured header is useless almost immediately.
4. The gateway **caches** validated sessions for a small, explicit TTL. This is
   the system's one deliberate consistency trade: a revoked session stays usable
   until its cache entry expires. The TTL is the revocation-latency budget and
   must be stated in configuration, not buried in code.
5. Services enforce permissions from the principal as **route middleware**,
   never from inside individual handlers.

## 9.3 The failure boundary this creates

Identity being down means **no new requests can be authenticated**, though
requests already holding a cached principal continue briefly. That is an
accepted, explicit single point of failure, and the reason it is worth stating
rather than discovering: it defines what "Identity is down" actually costs, and
it is the first thing M7's observability work should make visible.

Identity must fail closed. If authorization cannot be established, write
operations must not proceed.

---

# 10. Communication Model

## 10.1 Initial model

Use synchronous HTTP for required request-path dependencies.

Primary initial dependency:

```text
Orders -> Catalog
```

Authentication dependency (resolved in §9 — the gateway only):

```text
Gateway -> Identity
```

Downstream services do **not** call Identity on the request path.

Every service-to-service request must eventually support:

* request deadline;
* context propagation;
* request ID;
* trace context;
* bounded connection pool;
* typed error handling.

Retries must not be added generically.

They must be introduced only where an operation is safe to retry.

---

# 11. Asynchronous Architecture

An event broker must **not** be introduced during the initial microservice extraction.

It should be introduced only after the synchronous system is working.

Recommended initial broker:

* NATS JetStream

Reason:

* lightweight;
* simple local operation;
* durable messaging;
* sufficient for demonstrating asynchronous service patterns.

Kafka is intentionally excluded initially because its operational complexity would dominate this project's scope.

Candidate events:

```text
order.created
order.status_changed
product.updated
staff.invited
```

---

# 12. Transactional Outbox Requirement

Events must not be published directly as part of a database write path.

This is prohibited:

```text
INSERT order
publish order.created
```

because a crash can produce:

```text
order committed
event missing
```

or:

```text
event published
order rolled back
```

Instead:

```text
BEGIN

INSERT INTO orders ...
INSERT INTO outbox_events ...

COMMIT
```

A separate publisher reads the outbox:

```text
outbox_events
      |
      v
Outbox Publisher
      |
      v
     NATS
```

Consumers must be idempotent because delivery should be assumed to be at least once.

The event implementation should eventually demonstrate:

* transactional outbox;
* event versioning;
* event IDs;
* idempotent consumers;
* retries;
* poison-message/dead-letter behavior;
* observability across asynchronous flows.

---

# 13. Technology Direction

## Backend

Preferred stack:

```text
Go
net/http
chi
PostgreSQL
sqlc
golang-migrate
log/slog
OpenTelemetry
OpenAPI
```

Each service should use the same basic stack unless there is a strong reason not to.

Consistency is preferred over demonstrating multiple frameworks.

## Frontend

Preferred stack:

```text
Vite
React
TypeScript
React Router
TanStack Query
React Hook Form
Zod
```

UI library choice can remain separate from the architectural decision.

## Local runtime

```text
Docker Compose
```

Local development should require approximately:

```bash
make dev
```

or:

```bash
docker compose up
```

No Kubernetes dependency is required for normal local development.

---

# 14. Repository Direction

Suggested target layout:

```text
go-admin/
├── apps/
│   └── web/
│
├── services/
│   ├── identity/
│   │   ├── cmd/
│   │   ├── internal/
│   │   ├── migrations/
│   │   ├── openapi/
│   │   └── Dockerfile
│   │
│   ├── catalog/
│   │   ├── cmd/
│   │   ├── internal/
│   │   ├── migrations/
│   │   ├── openapi/
│   │   └── Dockerfile
│   │
│   └── orders/
│       ├── cmd/
│       ├── internal/
│       ├── migrations/
│       ├── openapi/
│       └── Dockerfile
│
├── platform/
│   ├── observability/
│   └── requestid/
│
├── deploy/
│   └── compose/
│
├── Makefile
└── README.md
```

This is a monorepo for operational convenience.

A monorepo does not mean the services share domain ownership.

---

# 15. Product Requirements

## 15.1 Authentication and accounts

The system must support:

* login;
* logout;
* session expiration;
* password changes;
* password resets;
* staff invites;
* staff deactivation;
* secure password hashing;
* HTTP-only cookies;
* session revocation.

The system must prevent:

* public administrator creation;
* self-promotion through request payloads;
* authorization bypass;
* credential disclosure.

---

## 15.2 Staff and RBAC

The system must support:

* staff listing;
* staff creation/invite;
* staff editing;
* staff deactivation;
* role CRUD;
* permission assignment;
* permission matrix UI.

Rules:

* at least one owner/admin must remain;
* users must not accidentally remove their own final administrative access;
* services enforce permissions independently at the HTTP boundary.

---

## 15.3 Catalog

The system must support:

* product CRUD;
* title;
* description;
* price;
* images;
* active/draft/archived state;
* search;
* filtering;
* pagination;
* sorting.

Later:

* stock level;
* low-stock status;
* inventory adjustment history.

Money must use integer minor units or exact decimal storage.

Floating-point monetary values are prohibited.

---

## 15.4 Customers

Customers initially belong to the Orders domain.

The system must support:

* customer creation;
* lookup by email;
* customer detail;
* customer order history;
* customer lifetime value.

A separate Customer Service is explicitly unnecessary during the initial architecture.

---

## 15.5 Orders

Orders must support a complete lifecycle.

Initial states:

```text
pending
paid
packed
shipped
delivered
cancelled
refunded
```

Transitions must be explicitly validated.

Example:

```text
pending -> paid
paid -> packed
packed -> shipped
shipped -> delivered
```

Cancellation and refund rules must be separately modeled.

Order line items may only be freely edited while an order is in an editable state.

OrderItem stores both:

```text
product_id
product_title_snapshot
unit_price_snapshot
quantity
```

Historical order values must not change when Catalog changes.

---

## 15.6 Audit history

Important changes must record:

* actor;
* action;
* timestamp;
* relevant before state;
* relevant after state.

At minimum, audit events must cover:

* order status transitions;
* product changes;
* role changes;
* permission changes;
* staff activation/deactivation.

---

## 15.7 Dashboard

The dashboard must eventually show:

* revenue over time;
* order volume;
* order counts by status;
* recent orders;
* low-stock products.

Reporting should initially live inside the Orders service.

No separate reporting service is required.

---

# 16. Delivery Milestones

## M0 — Preserve and Secure the Legacy Application

### Goal

Produce a runnable and safe baseline before architectural extraction.

This milestone intentionally stays close to the original repository assessment.

### Required work

* merge/remove internal employer registry paths;
* remove committed credentials;
* create a clean local bootstrap path;
* add `.env.example`;
* fix password hashing bug;
* fix authorization coverage;
* remove public administrator creation;
* fail startup on missing secrets;
* secure session cookie configuration;
* sanitize file uploads;
* remove obvious mass-assignment vulnerabilities;
* fix pagination;
* fix timestamp handling;
* replace dangerous request maps with typed DTOs;
* establish baseline tests;
* document the existing product behavior.

### Status: COMPLETE

Delivered on branch `m0/clean-clone-boot`, tagged `legacy-v1` (commit `cef8e55`).
All sixteen items above are closed; `git clone && make dev && make test` works
from a clean machine. See `docs/ASSESSMENT.md` for the defect catalogue this
milestone was written against.

### Output

```text
legacy-v1
```

### The legacy application is a reference, and is deleted

`legacy-v1` is **not** a migration source and **not** a system to keep running
alongside the services. There is no production data and no user to keep serving.
It exists to answer one question during the rebuild: *what did the working
application do here?*

What carries forward is the **domain decisions** listed in §2 and the behavioural
contract encoded in the tag's tests. The Go source does not.

Disposal is a named step, not an assumption:

* M2 lands Identity → delete the legacy auth, user, role, and permission code.
* M3 lands Catalog → delete the legacy product and upload code.
* M4 lands Orders → delete the remaining legacy backend.
* M5 lands the new frontend → delete `clients/`.

After M5 the only legacy artefact is the `legacy-v1` tag. Nothing in the working
tree should reference it.

Leaving a "temporary" legacy application in the repository is how it survives for
three years. Each milestone above removes what it replaces, in the same
milestone, as part of its definition of done.

### Definition of done

```text
git clone
make dev
make test
```

works from a clean machine with documented prerequisites.

Critical security defects identified in the repository assessment are closed.

---

# M1 — Microservice Platform Skeleton

### Goal

Create the future system topology before migrating significant business behavior.

### Runtime

The local stack should contain:

```text
web
gateway
identity
catalog
orders
postgres
```

### Each backend service must have

```text
GET /healthz
GET /readyz
```

and baseline support for:

* typed configuration;
* graceful shutdown;
* connection pooling;
* database migrations;
* structured logging;
* request IDs;
* OpenAPI;
* standardized HTTP error responses;
* Docker image;
* unit-test setup;
* CI build/test.

### Database

Create:

```text
identity_db
catalog_db
orders_db
```

with separate credentials.

Verify technically that each service cannot access another service's database.

### Definition of done

```bash
make dev
```

boots the entire empty platform.

```bash
make test
```

tests all services.

Health endpoints work.

CI builds and tests each service.

No domain functionality needs to be complete yet.

---

# M2 — Identity Service

### Goal

Build the first complete independent domain service.

### Capabilities

Implement:

* login;
* logout;
* session creation;
* session expiration;
* current user;
* staff CRUD;
* roles;
* permissions;
* role assignment;
* password hashing;
* staff deactivation.

### Security

Authorization must be implemented as route middleware.

Typed permission constants must replace free-form authorization strings where practical.

### Tests

At minimum:

* valid login succeeds;
* invalid password fails;
* expired session fails;
* revoked session fails;
* permission checks succeed/fail appropriately;
* non-admin cannot mutate role assignments;
* user cannot escalate privilege through request body fields.

### Definition of done

Identity runs independently, owns its schema, and can authenticate a user without Catalog or Orders running.

---

# M3 — Catalog Service

### Goal

Build the independently owned catalog domain.

### Capabilities

Implement:

* product creation;
* product editing;
* product listing;
* product detail;
* product archive;
* search;
* pagination;
* sorting;
* image handling;
* current price.

Add internal API:

```text
POST /internal/products/resolve
```

for resolving multiple product IDs in a single request.

### Tests

Include:

* product validation;
* exact money representation;
* archived product behavior;
* missing product response;
* batch resolution;
* permission enforcement.

### Definition of done

Catalog operates independently and owns all product persistence.

Neither Identity nor Orders can read Catalog's database.

---

# M4 — Orders Service and First Distributed Workflow

### Goal

Implement the order domain and introduce meaningful service-to-service communication.

### Capabilities

Implement:

* Customer;
* Order;
* OrderItem;
* order totals;
* order number;
* order lifecycle;
* status transitions;
* audit history;
* filters;
* order search.

### Required dependency

Order creation calls:

```text
Orders -> Catalog
```

to obtain current product information.

Orders persists product title and unit price snapshots.

### Required failure behaviors

Explicitly test:

```text
Catalog available       -> order creation succeeds

Product missing         -> order rejected

Catalog returns 500     -> order fails safely

Catalog timeout         -> deadline enforced

Catalog unavailable     -> no partial order created

Client cancels request  -> downstream request cancelled
```

### Observability

Request IDs must flow:

```text
Browser/Gateway
      ->
Orders
      ->
Catalog
```

### Definition of done

A real distributed order creation transaction works without cross-service database access.

This is the first milestone where the architecture must demonstrate genuine distributed-system behavior.

---

# M5 — Complete Admin Frontend

> **Detailed specification:** `docs/PRD/FrontendProductandDesignDirection.md`
> defines the design contract, the tooling workflow, and the M5.1–M5.5 breakdown.
> This section defines M5's scope and acceptance; that document defines how the
> work is done. Where they disagree, this section governs scope and that document
> governs craft.

### Goal

Replace the unfinished original frontend with a working administration product,
and delete `clients/` once it is replaced.

### Screens

Implement:

* Login
* Dashboard
* Products
* Product Detail
* Product Form
* Orders
* Order Detail
* Order Timeline
* Customers
* Customer Detail
* Staff
* Roles
* Permissions
* Profile
* Settings
* 403
* 404
* 500

### Requirements

Every data screen must support appropriate:

* loading state;
* empty state;
* error state.

List state such as filters and pagination should be reflected in the URL where appropriate.

Use a single typed API client strategy.

Remove hardcoded API URLs.

### E2E

At minimum:

```text
login
  ->
create product
  ->
create order
  ->
advance order status
  ->
verify history
```

### Definition of done

No placeholder screens remain.

The application functions as a usable shop back-office.

---

# M6 — Event-Driven Workflow

### Goal

Introduce asynchronous communication where it adds value and learn the correctness problems associated with messaging.

### Infrastructure

Add:

```text
NATS JetStream
```

### Initial event

Recommended first event:

```text
order.created
```

Optional second event:

```text
order.status_changed
```

### Requirements

Implement transactional outbox:

```text
orders transaction
├── write business data
└── write outbox event
```

Then:

```text
outbox publisher
      ->
NATS
```

Add at least one real consumer.

Suitable examples:

* notification worker;
* analytics projection;
* audit/report projection.

### Consumer requirements

Consumers must support:

* event ID;
* idempotency;
* retry;
* duplicate delivery;
* unknown event version handling.

### Failure tests

Prove:

* service crash after DB commit does not lose event;
* duplicate event does not duplicate side effect;
* broker outage does not roll back committed order;
* publisher resumes after restart.

### Definition of done

The application contains one justified, reliable asynchronous workflow rather than messaging for its own sake.

---

# M7 — Production Engineering and Observability

### Goal

Make the distributed architecture operationally understandable.

### Add

* OpenTelemetry instrumentation;
* distributed traces;
* service-level metrics;
* HTTP latency metrics;
* dependency metrics;
* structured logs;
* correlation IDs;
* readiness and liveness;
* dashboards;
* alert examples;
* rate limiting;
* load testing;
* fault injection.

A request should be traceable across services.

Example:

```text
POST /orders
  |
  +-- Identity / session validation
  |
  +-- Orders
        |
        +-- Catalog /products/resolve
        |
        +-- Postgres INSERT order
        |
        +-- Postgres INSERT outbox_event
```

### Definition of done

A developer can answer from telemetry:

* Which service failed?
* Which dependency caused latency?
* What request triggered the failure?
* Was the business operation committed?
* Was the event eventually published?

---

# M8 — Deployment Experiment

### Goal

Run the exact same services outside Docker Compose.

Kubernetes becomes appropriate here, not earlier.

Possible targets:

* k3d;
* kind;
* local Kubernetes;
* inexpensive managed Kubernetes;
* another container platform first, followed by Kubernetes.

### Optional Kubernetes work

* Deployments;
* Services;
* ConfigMaps;
* Secrets;
* readiness probes;
* rolling updates;
* migration jobs;
* horizontal autoscaling experiment;
* ingress;
* distributed telemetry.

Kubernetes is an operational exercise, not a prerequisite for proving the microservice design.

---

# 17. Testing Strategy

The project should use several test layers.

## Unit tests

Use for:

* domain rules;
* order transition matrix;
* permission logic;
* calculation logic;
* validation.

## Repository integration tests

Use real PostgreSQL for:

* migrations;
* SQL behavior;
* constraints;
* concurrency rules;
* transaction behavior.

Prefer testcontainers where appropriate.

## Service HTTP tests

Use:

```text
httptest
```

for handler-level contracts.

## Service integration tests

Test real service boundaries.

Example:

```text
Orders -> fake/real Catalog
```

Explicitly test dependency failures.

## Contract tests

OpenAPI definitions are service contracts.

Breaking API changes should be detected in CI.

## End-to-end tests

Keep these few but valuable.

Primary smoke path:

```text
login
create product
create order
change order status
view dashboard/order history
```

---

# 18. Reliability Requirements

Every network dependency must have an explicit timeout.

No outbound call should inherit an unlimited timeout.

Service shutdown must:

* stop accepting new work;
* allow in-flight requests a bounded completion period;
* close database connections cleanly.

Retries must be intentional.

Writes must not be blindly retried unless idempotency is guaranteed.

Error handling must distinguish at minimum:

* invalid request;
* unauthorized;
* forbidden;
* not found;
* conflict;
* dependency unavailable;
* internal failure.

---

# 19. Observability Requirements

Eventually every request must include:

```text
request_id
trace_id
service
route
status_code
duration
```

Important business logs should additionally include identifiers such as:

```text
order_id
order_number
product_id
user_id
```

without logging sensitive authentication credentials.

Services should expose enough metrics to observe:

* request rate;
* errors;
* latency;
* dependency failures;
* database latency;
* outbox backlog;
* event publish failures;
* consumer retries.

---

# 20. Security Requirements

The rebuild must correct all critical findings from the original repository.

At minimum:

* no credentials committed to source;
* passwords hashed correctly;
* no public admin registration;
* session cookie `HttpOnly`;
* production cookie `Secure`;
* appropriate `SameSite`;
* CSRF protection where required;
* typed request DTOs;
* no mass assignment;
* route-level permission enforcement;
* input validation;
* sanitized uploads;
* bounded upload size;
* login rate limiting;
* generic authentication error responses;
* no raw database errors returned to clients;
* secure headers before production exposure.

---

# 21. Architecture Decision Summary

The original assessment recommended:

> Modular monolith, one deployable binary, one database.

That recommendation is acknowledged as the simplest architecture for the product itself.

The project deliberately diverges from it because the engineering objective now includes implementing and operating microservices.

The revised decision is:

> **Build three bounded business services—Identity, Catalog, and Orders—with independent persistence ownership. Use synchronous HTTP first, introduce asynchronous events later through a transactional outbox, keep development on Docker Compose initially, and resist further service extraction unless a real boundary appears.**

This is not intended to simulate a large enterprise.

It is intended to create the **smallest credible distributed system** from the existing product.

---

# 22. Success Criteria

The revival is successful when all of the following are true:

1. The application is a complete usable staff back-office.
2. It starts from a clean clone.
3. Identity, Catalog, and Orders can each run independently.
4. Each service owns and protects its own persistence.
5. No cross-service database queries exist.
6. Orders successfully performs a real synchronous Catalog dependency.
7. Dependency failure behavior is explicitly tested.
8. Historical order information survives Catalog changes.
9. Authentication and authorization are secure.
10. OpenAPI defines service contracts.
11. Distributed requests are observable end to end.
12. At least one asynchronous workflow uses:

    * transactional outbox;
    * durable messaging;
    * idempotent consumption.
13. Docker Compose can run the entire environment locally.
14. CI tests every service.
15. Kubernetes, if introduced, runs the same service architecture rather than requiring architectural redesign.
16. The system remains small enough for a single developer to understand and maintain.

---

# 23. Guidance for Future Claude Sessions

When making implementation recommendations for this repository, treat the following as established project direction:

* The original repository assessment is still valid.
* The original implementation should not be preserved merely for backward compatibility.
* Existing product/domain intent should be preserved.
* There is no production data requiring migration compatibility.
* Microservices are now an explicit learning and architectural requirement.
* Do not recommend collapsing the target system back into a modular monolith unless specifically asked to reevaluate that decision.
* Do not expand beyond Identity, Catalog, and Orders without a concrete justification.
* Do not introduce messaging before the synchronous system works.
* Do not introduce Kubernetes before the services work locally under Docker Compose.
* Do not permit cross-service database access.
* Prefer explicit, understandable implementation over infrastructure sophistication.
* Treat every milestone as independently reviewable and testable.
* Preserve the original `OrderItem` snapshot concept.
* Preserve permission-based authorization while rebuilding its implementation safely.

Settled on 2026-09-11, after M0 shipped:

* The legacy application is disposable. Do not invest in it, do not migrate from
  it, and delete each part as its replacement lands (§16 M0).
* The permission vocabulary is Identity's published contract, consumed by
  codegen — not a shared domain package (§6.5).
* Authentication terminates once, at the gateway, which forwards a signed
  short-lived principal. Downstream services never call Identity on the request
  path (§9).
* The gateway is a thin Go service, not proxy configuration (§8.1).
* Milestone numbering in this document is canonical. Earlier milestone numbering
  in `docs/ASSESSMENT.md` is superseded.

The guiding principle is:

> **Use enough architecture to learn the hard parts of microservices, but not enough architecture to prevent finishing the product.**
