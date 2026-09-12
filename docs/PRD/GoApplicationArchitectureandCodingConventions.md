## Go Application Architecture and Coding Conventions

### Objective

The backend should follow idiomatic Go conventions and maintain clear architectural boundaries without introducing unnecessary framework-style abstractions.

Go does not require MVC, Clean Architecture, or traditional enterprise layering. The goal of this project is therefore not to reproduce Java, C#, Rails, or NestJS patterns in Go.

Prefer:

* explicit dependencies
* small interfaces
* composition
* package-level cohesion
* clear request boundaries
* simple control flow
* minimal abstraction

Do not introduce additional architectural layers unless they solve a concrete problem in the application.

---

## 1. Standard Request Flow

The default request lifecycle should be:

```text
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
```

Responses travel back through the same path:

```text
Database
    ↓
Repository
    ↓
Service
    ↓
Handler
    ↓
HTTP Response
```

This should remain the default architecture unless a feature has a specific reason to deviate from it.

---

## 2. Handlers

Handlers own the HTTP boundary.

Responsibilities include:

* route/path parameters
* query parameters
* request headers
* JSON request decoding
* basic request-shape validation
* calling the appropriate service/use case
* translating application errors into HTTP status codes
* converting domain/application results into API responses
* serializing the HTTP response

Handlers should not contain significant business logic.

Example:

```go
type ProductHandler struct {
    service *ProductService
}

func (h *ProductHandler) Create(w http.ResponseWriter, r *http.Request) {
    var req CreateProductRequest

    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, http.StatusBadRequest, err)
        return
    }

    product, err := h.service.Create(
        r.Context(),
        req.Name,
        req.Price,
    )
    if err != nil {
        handleServiceError(w, err)
        return
    }

    writeJSON(w, http.StatusCreated, toProductResponse(product))
}
```

### Handler vs Controller

Use the term **handler** for HTTP entry points.

Do not create a separate `controller` layer on top of handlers.

The following is discouraged:

```text
router
  → handler
      → controller
          → service
```

unless there is a clearly documented architectural reason.

For this project:

```text
Handler == HTTP controller responsibility
```

---

## 3. Services / Use Cases

Services contain application-level orchestration.

They implement operations such as:

```text
CreateProduct
UpdateProduct
DeleteProduct
CreateOrder
AdjustInventory
RegisterCustomer
```

A service may:

* enforce application rules
* coordinate repositories
* invoke domain behavior
* manage transactions
* call external systems
* combine multiple domain operations
* enforce authorization decisions that belong to the application layer

Example:

```go
type ProductService struct {
    repo ProductRepository
}

func (s *ProductService) Create(
    ctx context.Context,
    name string,
    price int64,
) (*Product, error) {
    product, err := NewProduct(name, price)
    if err != nil {
        return nil, err
    }

    if err := s.repo.Create(ctx, product); err != nil {
        return nil, err
    }

    return product, nil
}
```

Services should not know about HTTP concepts such as:

```text
http.Request
http.ResponseWriter
HTTP status codes
JSON
cookies
```

---

## 4. Actions

Do not create an `actions` package by default.

An "action" is conceptually equivalent to an application use case.

For example:

```text
CreateProduct
UpdateInventory
CancelOrder
```

These operations should normally live as methods on a domain/application service.

Preferred:

```go
productService.Create(...)
productService.Update(...)
```

rather than:

```text
actions/
    create_product.go
    update_product.go
    delete_product.go
```

A dedicated command/use-case type may be introduced when a workflow becomes sufficiently complex to justify it.

Example:

```go
type Checkout struct {
    inventory InventoryRepository
    orders    OrderRepository
    payments  PaymentClient
}
```

The abstraction must be motivated by complexity, not by architectural fashion.

---

## 5. Domain Types

Core business concepts should have explicit Go types.

Examples:

```text
Product
Customer
Order
OrderItem
Inventory
Payment
```

Domain types should represent business concepts rather than database tables or HTTP payloads.

Example:

```go
type Product struct {
    ID    ProductID
    Name  string
    Price Money
}
```

Business rules that naturally belong to a domain object may live with that object.

Example:

```go
func (p *Product) ChangePrice(price Money) error {
    if price.IsNegative() {
        return ErrInvalidPrice
    }

    p.Price = price
    return nil
}
```

Avoid anemic domain structures when behavior clearly belongs to the domain type, but also avoid forcing every operation into a domain object.

---

## 6. Avoid Generic `models` Packages

Do not use `models` as a catch-all directory.

The word "model" is ambiguous and commonly mixes several unrelated concepts:

```text
database rows
domain entities
HTTP requests
HTTP responses
external API payloads
```

Instead, name types by their role.

For example:

```text
Product
CreateProductRequest
ProductResponse
ProductRow
```

These types should live close to the feature that owns them.

Preferred:

```text
internal/
    product/
        product.go
        handler.go
        service.go
        repository.go
        postgres.go
        dto.go
```

Avoid:

```text
models/
    product.go
    customer.go
    order.go

controllers/
services/
repositories/
```

The preferred organization is primarily **by business capability**, not by technical layer.

---

## 7. DTOs and API Contracts

HTTP request and response structures should be distinct from domain objects when their contracts differ.

Example:

```go
type CreateProductRequest struct {
    Name  string `json:"name"`
    Price int64  `json:"price"`
}

type ProductResponse struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Price int64  `json:"price"`
}
```

Conversion should happen at the HTTP boundary.

Example:

```go
func toProductResponse(p *Product) ProductResponse {
    return ProductResponse{
        ID:    p.ID.String(),
        Name:  p.Name,
        Price: p.Price.Cents(),
    }
}
```

Simple conversion functions are preferred over introducing a generic transformation framework.

---

## 8. Transformers

Do not create a generic `transformers` package by default.

Simple mappings should use explicit functions such as:

```go
toProductResponse(...)
fromCreateProductRequest(...)
toOrderResponse(...)
toProductRow(...)
fromProductRow(...)
```

These functions should live near the types they transform.

Preferred:

```text
product/
    dto.go
```

with:

```go
func toProductResponse(...)
```

rather than:

```text
transformers/
    product_transformer.go
```

A dedicated mapper abstraction should only be introduced when transformation logic becomes large, reusable, or independently testable enough to justify it.

---

## 9. Repositories

Repositories define the persistence operations needed by application logic.

Example:

```go
type ProductRepository interface {
    Create(ctx context.Context, product *Product) error
    Get(ctx context.Context, id ProductID) (*Product, error)
    Update(ctx context.Context, product *Product) error
    Delete(ctx context.Context, id ProductID) error
}
```

Repository interfaces should be small and driven by the consumer's needs.

Do not automatically create generic CRUD repositories such as:

```go
type Repository[T any] interface {
    Create(T)
    Update(T)
    Delete(T)
    FindByID(string)
    FindAll()
}
```

unless the abstraction demonstrably improves the system.

Prefer explicit domain-specific interfaces.

---

## 10. Persistence Implementations

Database-specific implementations should remain behind repository boundaries.

Example:

```text
product/
    repository.go
    postgres.go
```

or, if persistence becomes substantial:

```text
product/
    repository.go

store/
    postgres/
        product.go
```

Database representations may differ from domain representations when useful.

For example:

```go
type productRow struct {
    ID        string
    Name      string
    Price     int64
    CreatedAt time.Time
}
```

The database schema should not automatically become the application's domain model.

---

## 11. Package Organization

Prefer organizing the application around business capabilities.

Example:

```text
cmd/
    api/
        main.go

internal/
    product/
        product.go
        handler.go
        service.go
        repository.go
        postgres.go
        dto.go

    customer/
        customer.go
        handler.go
        service.go
        repository.go
        postgres.go
        dto.go

    order/
        order.go
        handler.go
        service.go
        repository.go
        postgres.go
        dto.go

    inventory/
        inventory.go
        service.go
        repository.go

    platform/
        database/
        logging/
        config/
```

This is preferred over global technical-layer packages:

```text
controllers/
services/
repositories/
models/
transformers/
```

The reason is that feature-oriented packages preserve ownership boundaries and make future extraction easier.

---

## 12. Dependency Direction

Dependencies should generally point inward:

```text
handler
   ↓
service
   ↓
domain / repository interface
   ↓
infrastructure implementation
```

Business logic should not depend directly on:

```text
HTTP routers
database drivers
JSON libraries
framework-specific request objects
```

when those dependencies can reasonably remain at the boundary.

---

## 13. Interfaces

Use interfaces when there is an actual boundary.

Good examples:

```go
type ProductRepository interface {
    Get(ctx context.Context, id ProductID) (*Product, error)
}

type PaymentGateway interface {
    Charge(ctx context.Context, payment Payment) error
}
```

Avoid creating an interface for every struct purely for abstraction.

Do not automatically create:

```text
ProductServiceInterface
ProductHandlerInterface
ProductTransformerInterface
```

Go interfaces should generally be:

* small
* behavior-oriented
* defined close to the consumer
* introduced when substitutability is useful

---

## 14. Constructors and Dependency Injection

Dependencies should be explicit.

Example:

```go
repo := NewPostgresProductRepository(db)
service := NewProductService(repo)
handler := NewProductHandler(service)
```

Prefer normal Go constructor functions over dependency-injection frameworks.

The application wiring should remain visible and understandable.

---

## 15. Error Handling

> The full resolved model — including the client-facing `{code, message, status}`
> mapping owned by the HTTP boundary — is in
> `docs/PRD/ArchitectureDecisionClarifications.md` §6.

Application/domain errors should be distinguishable from infrastructure failures.

Example:

```go
var (
    ErrProductNotFound = errors.New("product not found")
    ErrInvalidPrice    = errors.New("invalid price")
)
```

The service returns application/domain errors:

```go
return nil, ErrProductNotFound
```

The handler translates those errors:

```go
switch {
case errors.Is(err, ErrProductNotFound):
    writeError(w, http.StatusNotFound, err)
case errors.Is(err, ErrInvalidPrice):
    writeError(w, http.StatusBadRequest, err)
default:
    writeError(w, http.StatusInternalServerError, err)
}
```

The service should not return HTTP status codes.

---

## 16. Transactions

Transaction boundaries should generally be owned by the application/service layer when a use case modifies multiple pieces of state.

Example:

```text
CreateOrder
    ├── reserve inventory
    ├── persist order
    └── record payment state
```

Do not expose transaction handling to HTTP handlers.

Keep transaction mechanics behind persistence abstractions where practical.

---

## 17. Logging

Logging should occur where useful context exists.

Handlers may log:

```text
request metadata
HTTP failures
request IDs
```

Services may log:

```text
important application decisions
external dependency failures
workflow state
```

Repositories may log or wrap:

```text
database failures
```

Avoid logging the same error independently at every layer.

Errors should be enriched or logged at the layer that has useful operational context.

---

## 18. Do Not Over-Abstract

The project should actively avoid unnecessary layers such as:

```text
Handler
→ Controller
→ Action
→ Manager
→ Service
→ Repository
→ DAO
→ Model
```

A normal operation should ideally remain understandable by following only a few files:

```text
handler.go
→ service.go
→ repository implementation
```

Additional abstractions require a concrete justification.

---

## 19. Intra-Service Modularity and Service-Sprawl Control

> Replaces the former "Microservice Compatibility" section, which was written for
> the earlier modular-monolith direction. See
> `docs/PRD/ArchitectureDecisionClarifications.md` §2.

Identity, Catalog, and Orders are **already established independent services** by
the canonical microservices PRD. This section does not reopen that.

Within each service, capabilities remain ordinary in-process Go packages unless
the canonical PRD establishes another service boundary:

* Customer remains inside Orders.
* Payments remain inside Orders.
* Inventory remains inside Catalog initially.
* Roles, permissions, staff, and sessions remain inside Identity.

These packages communicate through ordinary Go calls and small interfaces. Do not
convert them into separate network services merely because they are separate
business concepts.

Inside one of the established services, do not introduce additional service
discovery, databases, brokers, RPC boundaries, API gateways, or distributed
transactions without a concrete architectural requirement.

Cross-service communication between Gateway, Identity, Catalog, and Orders
follows the canonical PRD and therefore legitimately uses network boundaries,
independently owned databases, HTTP service-to-service calls, the gateway, and —
from M6 — asynchronous messaging.

```text
between established services
    -> follow SmallScaleMicroservicesArchitecture.md

inside one service
    -> simplest explicit Go structure
    -> normal Go calls and interfaces
    -> no accidental additional microservices
```

Do not introduce NATS or other event infrastructure before M6.

---

## 20. Migration Guidance for Existing Code

> **Largely moot.** The legacy application is tagged `legacy-v1` and is deleted
> milestone by milestone rather than migrated — see
> `ArchitectureDecisionClarifications.md` §8. Do not continue modernizing code
> scheduled for replacement. The responsibility-mapping questions below remain
> useful when reading `legacy-v1` to recover intended behaviour.

When modernizing existing code, do not perform mechanical renaming.

For example, do not simply transform:

```text
controllers/
models/
```

into:

```text
handlers/
services/
repositories/
domain/
dto/
transformers/
```

Instead, inspect the responsibility of each existing function.

For each function, determine:

```text
Is this HTTP handling?
→ handler

Is this business/application orchestration?
→ service

Is this domain behavior?
→ domain type or domain function

Is this database access?
→ repository implementation

Is this request/response mapping?
→ boundary conversion function

Is this generic infrastructure?
→ platform/infrastructure package
```

Refactor according to responsibility rather than directory naming.

---

## 21. Default Decision Rules for Claude

When implementing or refactoring backend functionality, follow these defaults:

1. Prefer a handler + service + repository flow.
2. Organize code around business capabilities.
3. Do not introduce both handlers and controllers.
4. Do not create an `actions` layer unless a use case warrants its own abstraction.
5. Do not create generic transformer classes or packages for simple mappings.
6. Do not use `models` as a catch-all type directory.
7. Keep HTTP DTOs separate from domain types when their contracts differ.
8. Keep database implementation details behind repositories.
9. Prefer explicit constructor-based dependency injection.
10. Prefer small consumer-owned interfaces.
11. Avoid generic repositories and speculative abstractions.
12. Keep business logic independent of HTTP and database frameworks.
13. Add architectural layers only when there is a concrete requirement.
14. Optimize for readability and maintainability over theoretical architectural purity.
15. Preserve the established Identity, Catalog, and Orders service boundaries. Within each service, preserve business-capability boundaries without introducing additional distributed-system boundaries unless explicitly justified.

---

## 22. Architecture Review Requirement

Before introducing a new architectural concept such as:

```text
actions
controllers
commands
mediators
transformers
managers
generic repositories
event buses
CQRS
domain events
```

Claude should first determine whether the existing handler/service/repository structure is insufficient.

If a new abstraction is introduced, the implementation should explain:

* what problem it solves
* why the existing structure is insufficient
* where the new boundary belongs
* whether the abstraction is needed now or merely anticipates a hypothetical future requirement

Do not introduce architectural complexity solely because it is common in another framework or architecture pattern.

The default bias for this codebase is:

> Use the simplest explicit Go structure that preserves clear business boundaries and can evolve when actual complexity appears.
