Architecture Decision Clarifications

Project: go-admin
Date: 2026-09-11
Purpose: Resolve ambiguities between the canonical microservices PRD, the Go application architecture conventions, and the frontend design direction.

1. Document Authority

The project uses three primary architecture/design documents with explicit precedence.

1.1 Canonical system architecture

docs/PRD/SmallScaleMicroservicesArchitecture.md

This is the highest authority for:

product scope

service boundaries

gateway responsibilities

authentication architecture

database ownership

cross-service communication

milestone scope and ordering

asynchronous messaging

observability

deployment direction

If another document conflicts with this PRD on system architecture, service ownership, or milestone scope, this PRD wins.

The target architecture is intentionally a small microservice architecture consisting of:

Gateway
├── Identity
├── Catalog
└── Orders

This is not a modular monolith awaiting later extraction.

1.2 Go implementation conventions

docs/PRD/GoApplicationArchitectureandCodingConventions.md

This document defines how Go code is structured inside each established service.

It governs:

handlers

services/use cases

domain types

repositories

DTOs

mappings

package organization

interfaces

dependency injection

errors

transaction boundaries

logging

abstraction discipline

It does not determine whether Identity, Catalog, or Orders should be separate services. Those boundaries are already settled by the canonical microservices PRD.

1.3 Frontend implementation and design

docs/PRD/FrontendProductandDesignDirection.md

This document defines:

frontend architecture

reusable UI primitives

design tokens

visual character

accessibility

responsive behavior

loading/empty/error/success states

Playwright visual review

frontend-design workflow

component reference usage

The canonical microservices PRD determines M5 scope and acceptance.

The frontend design document determines how that work is implemented and visually evaluated.

2. Resolution of Go Architecture §19

The existing wording of §19 was written for the earlier modular-monolith direction and is stale.

The intent of avoiding unnecessary distributed-system complexity remains valid, but it now applies within each established service, not across Identity, Catalog, and Orders.

Replace the old §19 with the following direction.

Intra-Service Modularity and Service-Sprawl Control

Identity, Catalog, and Orders are already established independent services by the canonical microservices PRD.

Within each service, capabilities should remain normal in-process Go packages unless the canonical PRD explicitly establishes another service boundary.

Examples:

Customer remains inside Orders.

Payments remain inside Orders.

Inventory remains inside Catalog initially.

Roles, permissions, staff, and sessions remain inside Identity.

These packages communicate through ordinary Go calls and small interfaces where useful.

Do not convert them into separate network services merely because they are separate business concepts.

Do not introduce additional:

service discovery

databases

brokers

RPC boundaries

API gateways

distributed transactions

inside one of the established services unless there is a concrete architectural requirement.

Cross-service communication between Gateway, Identity, Catalog, and Orders follows the canonical microservices PRD and therefore legitimately uses:

network boundaries

independently owned databases

HTTP service-to-service calls where defined

the gateway

later asynchronous messaging according to milestone scope

The rule is:

between established services
    -> follow SmallScaleMicroservicesArchitecture.md

inside one service
    -> simplest explicit Go structure
    -> normal Go calls/interfaces
    -> no accidental additional microservices

Messaging remains governed by the canonical milestone sequence.

Do not introduce NATS or other event infrastructure before M6.

3. Update to the Default Claude Decision Rules

The previous rule:

Preserve module boundaries that could support future microservice extraction without prematurely introducing distributed-system complexity.

is stale.

Replace it with:

Preserve the established Identity, Catalog, and Orders service boundaries. Within each service, preserve business-capability boundaries without introducing additional distributed-system boundaries unless explicitly justified.

4. Go Module Decision

Use a single root go.mod initially.

The target repository shape is:

go-admin/
├── go.mod
├── services/
│   ├── gateway/
│   │   ├── cmd/gateway/
│   │   └── internal/
│   ├── identity/
│   │   ├── cmd/identity/
│   │   └── internal/
│   ├── catalog/
│   │   ├── cmd/catalog/
│   │   └── internal/
│   └── orders/
│       ├── cmd/orders/
│       └── internal/
└── platform/

A single Go module is a source/dependency-management choice for the monorepo.

It does not mean the services are one application.

Each service remains:

a separate binary

a separate Docker image

independently runnable

independently migrated

owner of its own database

reachable through network APIs where the PRD requires it

Go's internal rules provide useful compile-time enforcement.

For example:

services/identity/internal/...

must not become a shared implementation library for Catalog or Orders.

A service must not import another service's implementation packages.

For example, Orders must not do this:

import "github.com/.../go-admin/services/catalog/..."

Orders consumes Catalog through Catalog's published HTTP/OpenAPI contract.

Cross-service generated clients and request/response types should live with the consumer, not in a shared domain package.

platform/ is reserved for deliberately shared technical infrastructure such as:

observability

tracing

request IDs

low-level HTTP helpers

It must not contain business/domain models.

If the project eventually requires genuinely independent release/versioning lifecycles, multi-module plus go.work can be reevaluated later.

5. M1 Walking Skeleton Decision

M1 keeps the scope defined in the canonical PRD:

No domain functionality needs to be complete yet.

However, M1 should include a minimal technical walking skeleton proving that the service topology actually works.

Do not implement Product, Login, Order, or other later domain functionality early.

Instead establish one technical path:

request
  -> gateway
  -> one backend service
  -> that service's own PostgreSQL database
  -> response

This path exists to verify:

gateway routing

container networking

request ID propagation

configuration

database credentials

service-owned database access

migrations

standardized HTTP errors

end-to-end networking through the actual running stack

M1 should finish with:

make dev
make test

plus an automated integration/smoke test proving that the technical path works.

Milestone responsibility remains:

M1 -> technical platform and walking skeleton
M2 -> first complete business service: Identity
M3 -> Catalog
M4 -> Orders + first meaningful distributed business workflow

The M4 Orders -> Catalog interaction remains the first meaningful distributed business workflow.

6. Error Handling Decision

The corrected error model is:

domain/service
    -> domain/application errors
    -> no HTTP knowledge

HTTP boundary
    -> maps known errors
       to {code, message, HTTP status}

Domain and application layers may define sentinel errors:

var ErrProductNotFound = errors.New("product not found")

The HTTP boundary owns the client-facing mapping:

ErrProductNotFound
    ->
404
PRODUCT_NOT_FOUND
"Product not found"

This preserves both requirements:

domain code remains transport-independent

API clients receive stable machine-readable error codes and safe messages

errors.New is therefore not globally prohibited.

Any existing CLAUDE.md rule or AST guard that globally forbids errors.New should be narrowed.

HTTP packages should not invent arbitrary client-facing errors outside the centralized HTTP mapping.

7. Preferred Per-Service Package Organization

The Go convention of organizing by business capability remains correct.

Apply it inside each established service.

Example:

services/
├── identity/
│   └── internal/
│       ├── staff/
│       │   ├── staff.go
│       │   ├── handler.go
│       │   ├── service.go
│       │   ├── repository.go
│       │   ├── postgres.go
│       │   └── dto.go
│       ├── role/
│       ├── permission/
│       └── session/
│
├── catalog/
│   └── internal/
│       ├── product/
│       └── image/
│
└── orders/
    └── internal/
        ├── order/
        └── customer/

Within a capability, the normal request flow remains:

HTTP Request
    ↓
Handler
    ↓
Service / Use Case
    ↓
Domain Logic
    ↓
Repository
    ↓
Database

Avoid global technical-layer directories such as:

handlers/
services/
repositories/
models/
transformers/

across an entire service when capability-oriented packages provide clearer ownership.

Do not introduce additional layers such as:

handler
-> controller
-> action
-> manager
-> service
-> repository
-> DAO

unless a concrete problem demonstrates that the normal structure is insufficient.

8. Current Project State

M0 is complete.

The legacy application is tagged:

legacy-v1

It is a behavioral/historical reference, not the target implementation.

Do not redo M0 or continue modernizing legacy code that is scheduled for replacement.

As replacement milestones land:

M2 -> remove legacy Identity/auth/RBAC implementation
M3 -> remove legacy Catalog/product/upload implementation
M4 -> remove remaining legacy backend
M5 -> remove legacy clients/

After M5, the legacy-v1 tag should be the only remaining legacy artifact.

9. Claude Conflict-Resolution Rule

When documents appear to disagree, use this precedence:

System / product / milestone decision
    -> SmallScaleMicroservicesArchitecture.md

Go implementation decision inside a service
    -> GoApplicationArchitectureandCodingConventions.md

Frontend implementation / design decision
    -> FrontendProductandDesignDirection.md

A lower-level implementation document must not redefine a higher-level system boundary.

Examples:

Go conventions cannot collapse Identity, Catalog, and Orders into one process.

Go conventions cannot add new microservices merely because packages are distinct.

Frontend conventions cannot change M5's required product scope.

The canonical PRD should not dictate arbitrary component styling already delegated to the frontend design contract.

10. Guiding Principles

System architecture:

Use enough architecture to learn the hard parts of microservices, but not enough architecture to prevent finishing the product.

Go implementation:

Use the simplest explicit Go structure that preserves clear business boundaries.

Frontend:

Build a coherent professional operational application, not a collection of generated screens.