# go-admin — Repository Assessment & Revival Plan

**Date:** 2026-09-11
**Repo:** `/Users/dawitnoah/Desktop/project/go-admin`
**Scope reviewed:** all 1,300 lines of first-party code (7 Go controllers, 7 models, 2 middlewares, 1 route file, 9 React/TS source files), git history (88 commits, all 2022-04-01 → 2022-04-03), CI config, build tooling.

---

## 1. Understanding of the product

### 1.1 Core product idea

This is a **staff back-office (admin panel) for a small e-commerce shop**.

The evidence is unambiguous and comes from the domain model, not from the README (which is one line: `## GO Admin`):

- `models/product.go` — catalog items with `Title`, `Description`, `Image`, `Price`
- `models/order.go` — `Order` + `OrderItem`, where `OrderItem` snapshots `ProductTitle` and `Price` at purchase time (the classic order-line pattern)
- `models/role.go` / `models/permission.go` — a role→permission many-to-many for *staff*, not customers
- `controllers/order_controller.go:Chart` — a daily revenue time series (`SUM(oi.price * oi.quantity) GROUP BY date`)
- `controllers/order_controller.go:Export` — CSV export of orders for finance/ops
- `controllers/image_controller.go` — product image upload
- `clients/public/index.html` — `<title>Go React Admin</title>`, and a `<link rel="canonical" href="https://getbootstrap.com/docs/5.1/examples/dashboard/">` left in, confirming the UI was scaffolded from the Bootstrap 5 dashboard example

**Provenance — verified, not inferred.** This repository is a fork-by-retyping of [`antoniopapa/go-admin`](https://github.com/antoniopapa/go-admin), the companion backend for the Udemy course [React and Golang: A Practical Guide](https://www.udemy.com/course/react-go-admin/) (upstream frontend: [`antoniopapa/react-admin`](https://github.com/antoniopapa/react-admin)). Confirmed by direct source comparison:

- `models/entity.go` is byte-identical, down to the `Take(db, limit, offset) interface{}` signature
- `middlewares/permission_middleware.go` reproduces upstream's `"view_"+page || "edit_"+page` / `"edit_"+page` scheme exactly
- `models/order.go` matches structurally, including the `json:"-"` customer names, the `gorm:"-"` computed `Name`/`Total`, and `CreatedAt`/`UpdatedAt` typed as `string`
- the file set matches one-for-one under a rename (`authController.go`→`auth_controller.go`, `util/`→`utils/`, `connect.go`→`db.go`, `paginate.go`→`pagination.go`)
- `.gitignore` excludes `uploads/*`, `csv/*.csv`, and `.realize.*` — upstream contains precisely `uploads/si.jpeg`, `csv/orders.csv`, and `.realize.yaml`. The ignore rules were inherited for files this repo never had.

That matters for the assessment because **it explains the gaps**: the repo implements the tutorial's backend surface and then stops partway through the frontend. It is not an abandoned product with a lost vision — it is a half-finished walkthrough.

**Which defects are inherited and which are local** — this changes how to read §4:

| Defect | Origin |
|---|---|
| §4a `SetPassword` hashes `"test"` | **Local.** Upstream correctly hashes `password`. A debugging placeholder never reverted. |
| §4b authorization applied to only one controller | **Local.** Upstream calls `IsAuthorized` from its product, order, and role controllers too. |
| §4l pagination `Ceil` after integer division | Inherited (upstream uses `limit := 15`; this repo changed it to 5). |
| §4j/k `CreatedAt string`, `json:"-"` customer names | Inherited. |
| §4y the `Entity` abstraction | Inherited verbatim. |

Deliberate local divergences: `/api/v1/` prefix with singular resource paths (upstream: `/api/` plural), camelCase JSON keys (upstream: snake_case), added `checkmail` validation and a `Validate()` method upstream lacks, and a hand-written React client without upstream's Redux.

The two most severe findings being *local regressions* rather than tutorial defects does not change the §11 recommendation — that case rests on the security model and the absent test seam — but it does mean the inherited domain model is slightly sounder than the one currently in the working tree.

### 1.2 Intended users

**Internal staff of a small shop**, differentiated by role:

- **Admin** — manages staff accounts, roles, and permissions
- **Staff/Editor** — manages catalog and fulfils orders
- **Viewer** — read-only access to orders and the sales dashboard

There is no customer-facing user in this codebase. `models/user.go` is a *staff* user (it has a `RoleId` and nothing resembling a shipping address or cart). Customers exist only as three denormalized columns on `Order` (`FirstName`, `LastName`, `Email`).

### 1.3 Main user journeys

Reconstructed from the route map in `routes/routes.go`:

1. **Staff onboarding** — register → login → land on dashboard
2. **Catalog management** — list products (paginated) → create → upload image → edit → delete
3. **Order review** — list orders with computed totals → open one → see line items → update → export CSV
4. **Sales reporting** — view the daily revenue chart
5. **Access control administration** — create roles → attach permissions → assign a role to a staff user
6. **Self-service account** — update own name/email, change own password

Journeys 1 and 6 exist end-to-end on the backend. Journeys 2–5 exist **only** on the backend; the frontend has no screens for them at all.

### 1.4 Major entities / domain concepts

| Entity | Defined in | Notes |
|---|---|---|
| `User` (staff) | `models/user.go` | belongs-to `Role` |
| `Role` | `models/role.go` | many-to-many `Permission` via `role_permissions` |
| `Permission` | `models/permission.go` | free-text `Name`, conventionally `view_<page>` / `edit_<page>` |
| `Product` | `models/product.go` | catalog item |
| `Order` | `models/order.go` | has-many `OrderItem`; customer identity inlined |
| `OrderItem` | `models/order.go` | price/title snapshot, **no `ProductId`** |
| `Entity` | `models/entity.go` | a `Count`/`Take` interface used only by pagination |

**Missing entities that the domain clearly implies:** `Customer`, `Payment`, order `Status`, `Address`, `Category`, `InventoryLevel`, `AuditLog`.

### 1.5 What functionality currently exists

**Backend — works, or nearly:**
- Register / login / logout with bcrypt + HS256 JWT in an httpOnly cookie (`controllers/auth_controller.go`, `utils/jwt.go`)
- Authentication middleware gating all non-auth routes (`middlewares/auth_middleware.go`)
- Full CRUD for users, products, orders, roles; create/list for permissions
- Offset pagination shared across users/products/orders (`models/pagination.go`)
- Image upload with static file serving (`controllers/image_controller.go`, `routes/routes.go:57`)
- CSV export and a revenue chart endpoint (`controllers/order_controller.go`)
- GORM `AutoMigrate` for all six tables (`database/db.go:22`)

**Frontend — two working screens:**
- `pages/Login.tsx` and `pages/Register.tsx` — real forms, real API calls
- `Components/Layout.tsx` — an auth guard that pings `/api/v1/user` and redirects to `/login` on failure
- `Components/Nav.tsx` — shows the signed-in user's first name, logout action

**Verified build state:** `go build ./...` and `go vet ./...` are clean. The Go side compiles today.

### 1.6 What appears unfinished

- **`pages/Users.tsx`** — a Bootstrap example table with hardcoded `Header`/`random data placeholder text`. No fetch, no state. Pure scaffold.
- **`pages/Dashboards.tsx`** — renders the literal string `Dashboard`. The `/api/v1/chart` endpoint it exists to display is never called.
- **No screens at all** for products, orders, roles, permissions, or profile — despite 20 backend endpoints serving them.
- **`Components/Nav.tsx:51`** links to `/profile`, which has no route in `App.tsx`. Dead link.
- **`Components/Menu.tsx`** offers only Dashboard and Users — the sidebar was never extended past the tutorial's first screen.
- **`models/entity.go`** — the `Entity` abstraction was extracted (commits `040ae51 implement interface functions`, `dc7738d implement interface`) but only ever serves `Paginate`.
- **`Order.Total` / `Order.Name`** are `gorm:"-"` computed fields populated only in `Take` — so a single-order fetch (`GetOrder`) returns `total: 0` and `name: ""`. The computation was never finished for the detail path.
- **CI is a stub** — `.github/workflows/codeql-analysis.yml` is the unmodified GitHub default. There is no build, test, or lint workflow.

### 1.7 What is missing to be a complete product

This is the honest headline: **the application cannot be started from a clean clone, and has never been run by anyone but its author.**

- **No bootstrap path.** `database/db.go:12` hardcodes `dns := "test-user:password@/go_admin"` — it ignores the `.env` that `main.go` insists on loading. There is no schema seed, no `docker-compose`, no `Dockerfile`, no migrations directory, no `.env.example`. A new developer cannot get a running database.
- **No first admin.** `Register` hardcodes `RoleId: 1` (`auth_controller.go:32`), but nothing ever creates role 1. On a fresh database registration fails on a foreign-key error and returns a raw GORM error object as JSON. The chicken-and-egg is unresolved.
- **Zero tests.** No `*_test.go`, no `*.test.tsx`. `setupTests.ts` exists but tests no code.
- **No deployment config** of any kind.
- **No order lifecycle.** An order has no status. It cannot be placed, paid, packed, shipped, delivered, cancelled, or refunded — the single most important workflow in the domain is absent.
- **No way for orders to be created** in the real world. Nothing writes orders except a raw `POST /api/v1/orders` — and that endpoint can't even set the customer name (§4, mass-assignment/`json:"-"` note).
- **No observability** — no structured logging, no request IDs, no metrics, no error tracking. The only logging in the codebase is a stray `fmt.Println` inside the authorization hot path (`middlewares/permission_middleware.go:38`).

---

## 2. Map of the current architecture

```
┌─────────────────────────────────────────────────────────────┐
│  clients/  —  CRA 5.0.0 + React 18 + TS 4.6 + Bootstrap CDN │
│                                                             │
│  index.tsx ──> App.tsx (BrowserRouter, 4 routes)            │
│                  ├── /        Dashboards.tsx  [STUB]        │
│                  ├── /users   Users.tsx       [STUB]        │
│                  ├── /login   Login.tsx       [works]       │
│                  └── /register Register.tsx   [works]       │
│                                                             │
│  Components/Layout.tsx  — auth guard, fetches /user         │
│  Components/Nav.tsx     — ALSO fetches /user (duplicate)    │
│  Components/Menu.tsx    — 2-item sidebar                    │
│  interfaces/user.ts     — hand-written, already misaligned  │
│                                                             │
│  No state management. No API client layer. No error         │
│  boundaries. axios called inline with //@ts-ignore.         │
└────────────────────────┬────────────────────────────────────┘
                         │ hardcoded http://localhost:8080
                         │ withCredentials: true (cookie)
┌────────────────────────▼────────────────────────────────────┐
│  Go 1.17 + Fiber v2.31 + GORM 1.23 + MySQL                  │
│                                                             │
│  main.go ── godotenv ── CORS(AllowCredentials, wildcard)    │
│     └── routes/routes.go  (flat, 27 routes, all /api/v1)    │
│           │                                                 │
│           ├── PUBLIC:  register, login                      │
│           └── app.Use(middlewares.IsAuthenticated)          │
│                 └── everything else                         │
│                                                             │
│  controllers/*.go  — HTTP parsing + business logic + SQL     │
│                      all in one function. No service layer. │
│  models/*.go       — GORM structs + Count/Take + validation │
│  middlewares/      — IsAuthenticated (route-level)          │
│                      IsAuthorized (called MANUALLY, and     │
│                      ONLY from user_controller.go)          │
│  utils/jwt.go      — global mutable SecretKey               │
│  database/db.go    — global DB, hardcoded DSN, AutoMigrate  │
└─────────────────────────────────────────────────────────────┘
```

**Monorepo structure:** there isn't one. The Go module sits at the repository root and the React app is a subdirectory with its own independent `package.json`. There is no workspace tool, no shared package, no shared types, and no root-level task runner beyond a 2-target `Makefile` (`gofmtcheck`, and `startdev` which only starts the frontend).

**Layering:** three layers exist by folder name (`controllers` / `models` / `database`) but not by responsibility. Controllers call `database.DB` directly. Models contain query logic. There is no repository, service, or domain layer, and no dependency injection — `database.DB` and `utils.SecretKey` are package-level mutable globals.

---

## 3. What is good and worth keeping

Being fair to a four-year-old side project, these were the right instincts:

1. **The domain decomposition is correct.** Product / Order / OrderItem / User / Role / Permission is the right entity set for a shop back-office. The *modeling* of those entities is weak (§6), but the *choice* of entities survives a rebuild intact.

2. **`OrderItem` snapshots `ProductTitle` and `Price`.** This is a genuinely good decision that many senior developers get wrong — an order line must record what was actually sold at the price actually charged, not join to a mutable product row. Keep this.

3. **Permission-based authorization rather than role-string checks.** `middlewares/permission_middleware.go` checks for a named permission (`edit_users`) rather than `if role == "admin"`. The *implementation* is broken (§4), but checking capabilities instead of roles is the architecturally correct model and should carry forward.

4. **httpOnly cookie + bcrypt.** Choosing an httpOnly cookie over `localStorage` for the session, and bcrypt with cost 14 for passwords, are both correct. The execution has holes (§4) but the intent is right.

5. **GET/write permission distinction.** `permission_middleware.go:34-48` treats `GET` as satisfied by either `view_x` or `edit_x`, and writes as requiring `edit_x`. That read/write split is a sound model.

6. **TypeScript on the frontend from day one**, and a `.prettierrc`. Good discipline for 2022.

7. **The route map is a usable specification.** `routes/routes.go` is a clear, complete statement of the intended API surface. It should be treated as the requirements document for the rebuild even though the routes themselves get redesigned (§6).

8. **The unmerged `fix/remove-internal-registry-paths` branch** (commit `67d4cfe`) already renames the module from `gitlab.nordstrom.com/go-admin` to `github.com/loloDawit/go-admin` across 24 imports and repoints `package-lock.json` from an internal Artifactory mirror to public npm. This is genuinely useful work that should be merged first, not redone.

---

## 4. What is weak or amateur and should change

I'll separate these by severity, because the distinction matters for the revive/rebuild decision.

### 4.1 Critical — security

**(a) Every account has the same password. `models/user.go:20-23`**

```go
func (user *User) SetPassword(password string) {
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("test"), 14)
	user.Password = string(hashedPassword)
}
```

The `password` parameter is ignored; the literal string `"test"` is hashed. Every user ever registered through `Register` (`auth_controller.go:42`) or changed through `UpdatePassword` (`auth_controller.go:172`) has the password `test`. Login then succeeds for anyone who types `test`. **This is a total authentication bypass** and it is the single worst defect in the repository. The `bcrypt` error is also discarded with `_`.

**(b) Almost nothing is authorized.** `middlewares.IsAuthorized` is invoked in exactly five places, all in `controllers/user_controller.go` (lines 13, 21, 36, 55, 72). It is never called from the product, order, role, permission, or upload controllers. The practical consequence: **any authenticated user — including one who just self-registered — can create, edit, and delete the entire catalog, every order, and every role and permission**, and can grant themselves any permission they like via `POST /api/v1/roles`. Authorization is opt-in per handler and was opted into once.

**(c) Public registration creates administrators.** `auth_controller.go:32` hardcodes `RoleId: 1`, and git commit `a97be99 add admin role for register call` confirms role 1 is the admin role. The registration endpoint is public (`routes/routes.go:11`). Anyone who can reach the API can mint themselves an admin account.

**(d) Login succeeds with no signing key. `utils/jwt.go:15-22`**

```go
if len(SecretKey) > 0 {
	return clamis.SignedString([]byte(SecretKey))
}
return "", nil
```

If `SECRET` is unset or empty, `GenerateJWT` returns an empty string **and a nil error**. `Login` treats that as success and sets an empty `jwt` cookie. A misconfigured deploy produces an app that appears to log users in while issuing no credential at all. The correct behaviour is to refuse to start.

**(e) CORS is wildcard-with-credentials. `main.go:31-33`** — `cors.New(cors.Config{AllowCredentials: true})` with `AllowOrigins` left at its default `"*"`. Combined with cookie auth and no CSRF token, and a session cookie set with neither `SameSite` nor `Secure` (`auth_controller.go:83-88`), this is a cross-site request forgery hole. `Login` additionally returns the JWT in the response body (`auth_controller.go:92-94`) — two parallel auth channels, one of which defeats the point of `HTTPOnly`.

**(f) Arbitrary file write via upload. `controllers/image_controller.go:25-29`** — `file.Filename` comes straight from the client and is concatenated into `"./uploads/"+filename`. There is no path sanitization (a `../` traversal writes outside `uploads/`), no MIME allowlist, no extension check, no size limit, and no collision handling (same name silently overwrites). The route is also unauthorized (see 4b). The returned URL is hardcoded to `http://localhost:3000` (line 33) while the backend serves on 8080 — so the feature returns broken URLs even when it works.

**(g) Mass assignment.** Handlers construct a struct with the path `Id` set, then `BodyParser` into the *same* struct — e.g. `controllers/product_controller.go:31-40`, `user_controller.go:41-49`. A client-supplied `"id"` in the body overwrites the path parameter, so `PUT /api/v1/user/5` with `{"id": 1, ...}` updates user 1. In `UpdateUser` the body can also set `roleId`, which combined with 4b means any authenticated user can escalate to admin. Password fields are similarly assignable on `User` even though `json:"-"` protects the read direction only.

**(h) Credentials in source. `database/db.go:12`** — `dns := "test-user:password@/go_admin"` is committed. It also means the `.env` loading in `main.go` is decorative for the database.

**(i) User enumeration.** `Login` returns 404 "user not found" for an unknown email (`auth_controller.go:64-69`) and 400 for a bad password (`:71-76`). The response distinguishes registered from unregistered addresses.

### 4.2 Serious — correctness

**(j) The chart has always returned nothing useful.** `models/order.go:12-13` declares `UpdatedAt` and `CreatedAt` as `string`. GORM only auto-populates these when they are `time.Time` (or an int Unix field), so every row's `created_at` is the empty string. `Chart`'s `DATE_FORMAT(o.created_at, '%Y-%m-%d')` therefore groups every order into one meaningless bucket. The flagship reporting feature does not work and never did.

**(k) Orders cannot be created through the API.** `Order.FirstName` and `Order.LastName` are tagged `json:"-"` (`models/order.go:8-9`), and `CreateOrder` (`order_controller.go:66-74`) populates the order solely via `BodyParser`. Those fields are unreachable from JSON, so every created order has an empty customer name. `Order.Name` and `Order.Total` are `gorm:"-"` and computed only inside `Take` — so `GetOrder` returns `total: 0`, `name: ""` for every single order.

**(l) Pagination arithmetic. `models/pagination.go:17`** — `math.Ceil(float64(int(total) / limit))` performs **integer** division first and then ceils the already-truncated result, so `Ceil` is a no-op. With 12 records and `limit` 5, `last_page` is 2 instead of 3 — the final page is unreachable in any UI that trusts it. `limit` is also hardcoded to 5 (line 12) with no client control.

**(m) Roles never load on the user list. `models/user.go:40`** — `db.Preload("role")` uses the lowercase column name; GORM preloads by *field* name, which is `Role`. The association silently doesn't load, so `GET /api/v1/users` returns every user with an empty role object. (`user_controller.go:28` gets the casing right, so the bug is inconsistent across endpoints.)

**(n) Unchecked type assertions panic the request. `controllers/role_controller.go:41,46,60,90,95,102`** — `roleDTO["permissions"].([]interface{})` panics if the key is absent; `permissionId.(string)` panics if the client sends permission IDs as JSON numbers, which is the natural encoding. `roleDTO["name"].(string)` panics if `name` is omitted. These are the default failure modes, not edge cases. Line 56 also calls `database.DB.Table(...).Delete(result)` with a nil `interface{}`.

**(o) Database errors are systematically discarded.** `DB.Create`, `DB.Updates`, and `DB.Delete` results are unchecked in every controller except `Register`. A failed write returns HTTP 200 with the object the client just sent. Likewise `Find` on a nonexistent ID returns 200 and a zero-valued struct rather than 404 — `GetProduct` (`product_controller.go:15-24`) will happily return `{"id": 9999, "title": ""}`.

**(p) `Updates(struct)` cannot clear a field.** GORM's struct-based `Updates` skips zero values, so no client can ever blank a product description or set a price to 0 through `UpdateProduct`.

**(q) `strconv.Atoi` errors ignored everywhere** — `id, _ := strconv.Atoi(ctx.Params("id"))` in every controller means `/api/v1/product/abc` silently becomes product 0.

**(r) CSV export is racy and lossy. `order_controller.go:79-84,107`** — all users write to the same `./csv/orders.csv` path (concurrent exports corrupt each other), `os.Create` fails if `csv/` doesn't exist (it's gitignored, so it won't on a fresh clone), and `strconv.Itoa(int(orderItem.Price))` truncates every price to a whole number. `Export` is a `POST` that returns a file download (`routes/routes.go:41`) — wrong verb.

**(s) Inconsistent ID types.** `User.Id` is `int`; `Order.Id`, `Product.Id`, `Role.Id`, `Permission.Id` are `uint`. Sequential integer IDs are also enumerable across the whole API.

**(t) Authorization does N queries per request.** `permission_middleware.go:27-33` loads the user, then separately loads the role with permissions, on *every* authorized request, with no caching and no join. It also leaves a `fmt.Println(permission.Name, page)` in the loop (line 38), logging authorization internals to stdout on every check.

### 4.3 Structural / design

**(u) No layering.** Every controller does HTTP parsing, validation, business logic, and data access in one function. There is no service or repository layer, which is precisely why there are no tests — nothing is testable without a live MySQL.

**(v) Global mutable state.** `database.DB` and `utils.SecretKey` are package-level vars mutated at startup. No injection, no ability to substitute a test double.

**(w) `AutoMigrate` instead of migrations. `database/db.go:22`** — `AutoMigrate` never drops or alters columns destructively, cannot be reviewed in a pull request, cannot be rolled back, and gives no record of schema history. It is unsafe as a production deployment mechanism.

**(x) `map[string]string` as the request type.** `Register`, `Login`, `UpdateUserInfo`, and `UpdatePassword` all parse into `map[string]string`, so there is no schema, no type safety, and no validation of unexpected fields. This directly caused a live bug: `Register` reads `data["firstName"]` (camelCase, `auth_controller.go:28`) but `UpdateUserInfo` reads `data["firstname"]` (lowercase, `:139`) — and the frontend sends camelCase. **Profile updates silently blank the user's name.** A typed struct would have caught this at compile time.

**(y) `models.Entity` is an abstraction with one implementation path.** `Count`/`Take` were pushed onto three models to serve one `Paginate` function. `Take` returns `interface{}`, discarding all type information. A generic function (Go 1.18+) or a simple repository does this better.

**(z) Validation is vestigial.** `user.Validate(action string)` (`models/user.go:46`) takes an `action` parameter it never reads, is called only from `Register`, and is the only validation in the entire backend. `CreateUser`, every product write, and every order write validate nothing at all.

### 4.4 Frontend

- **`react-scripts` 5.0.0 (CRA) is deprecated and unmaintained**, with known advisories in its transitive dependency tree. This is not a "keep up with trends" call — there is no upstream to receive fixes from.
- **React 18 runtime with React 17 types and the React 17 API.** `package.json` pins `react: ^18.0.0` but `@types/react: ^17.0.43`, and `index.tsx:7` uses the removed-in-spirit `ReactDOM.render` instead of `createRoot`. The app runs in React 17 legacy mode; concurrent features are off.
- **The frontend/backend contract is already wrong.** `interfaces/user.ts:11` declares `permissions: string` on `Role`, but the backend sends an array of permission objects (`models/role.go:6`). Line 8 declares `roleId: 1` — a *literal type*, so only the value 1 type-checks. These interfaces are hand-maintained guesses, and they've already drifted.
- **`//@ts-ignore` above every single axios call** (`Layout.tsx:23`, `Nav.tsx:20`, `Nav.tsx:37`, `Login.tsx:31`, `Register.tsx:38`). Type safety was switched off at exactly the boundary where it matters most.
- **No API client layer.** Every component hand-builds an axios config object with a hardcoded `http://localhost:8080` URL. There is no base URL config, no interceptor, no shared error handling.
- **No auth context.** `Layout.tsx` and `Nav.tsx` independently fetch `/api/v1/user` on mount — two requests for one page load, two sources of truth, and `Nav` has no error handling at all (`Nav.tsx:37` will throw an unhandled rejection when logged out).
- **No loading, empty, or error states anywhere.** `Login.tsx:32-35` on a failed login throws an unhandled promise rejection and `setRedirect(true)` is never reached — the user sees a form that silently does nothing. `console.log(we)` is left in at line 33.
- **Logout doesn't clear client state** (`Nav.tsx:11-22`) and doesn't await before navigating.
- **Uncontrolled inputs in `Login.tsx`** (no `value` prop) while `Register.tsx` uses controlled ones — inconsistent within four files.
- **No accessibility work.** No form `<label>` elements (placeholders are used as labels, which screen readers do not announce reliably), `<a href="#">` for the brand link (`Nav.tsx:41`), `<Link>` elements as direct children of `<ul>` without `<li>` (`Nav.tsx:49-55`) — invalid HTML that breaks list semantics.
- **Responsive behaviour is inherited, not designed** — the Bootstrap example's grid classes are copied but the sidebar has no mobile toggle, so on small screens it collapses and cannot be reopened.

---

## 5. What is unfinished

Consolidated (detail in §1.6):

| Area | State |
|---|---|
| `pages/Users.tsx` | Bootstrap placeholder markup, no data |
| `pages/Dashboards.tsx` | Renders the word "Dashboard" |
| Products / Orders / Roles / Permissions UI | Do not exist — 20 backend endpoints with no client |
| `/profile` | Linked from `Nav.tsx:51`, no route, no page |
| `Menu.tsx` | 2 of ~6 needed nav items |
| Chart endpoint | Implemented, broken (§4j), never called by the UI |
| CSV export | Implemented, racy and lossy (§4r), no UI trigger |
| Image upload | Implemented, insecure (§4f), no UI trigger |
| `Order.Total` / `Order.Name` | Computed on list, never on detail |
| `models/entity.go` | Abstraction extracted, used once |
| `user.Validate(action)` | Parameter accepted, never used |
| CI | Default CodeQL stub only; no build/test/lint |
| Tests | None |
| Docs | One line |

---

## 6. Major technical risks

1. **Authentication is bypassed (§4a) and authorization is essentially absent (§4b, §4c).** If this were ever exposed to a network it would be fully compromised by an anonymous user. This alone disqualifies incremental modernization as a strategy — the security model has to be rebuilt, not patched.

2. **The app does not boot from a clean clone.** Hardcoded DSN, no seed, no role 1, no compose file, no `.env.example`. Any revival that doesn't start here produces no reviewable increment.

3. **Zero test coverage means every change is unverifiable.** There is no safety net, and — worse — no *testable seam*: with `database.DB` as a global and all logic inside HTTP handlers, tests cannot be written without first restructuring. Test debt and architecture debt are the same debt here.

4. **`AutoMigrate` on a production database (§4w)** — no history, no rollback, no review.

5. **Dependency rot.** Go 1.17 (unsupported; 1.24/1.25 are current), `golang-jwt/jwt` v3 unversioned-incompatible import (v5 is current; the v3 line carries known algorithm-confusion advisories), CRA 5 unmaintained, React types a major version behind the runtime.

6. **Silent data corruption.** Discarded DB errors (§4o) mean failed writes report success. `UpdateUserInfo`'s key-casing bug (§4x) blanks user names on every profile save. `Itoa(int(price))` (§4r) truncates money in exports.

7. **Panics reachable from ordinary input** (§4n) — a JSON-numeric permission ID takes down the request.

8. **`gitlab.nordstrom.com` in the module path** — an employer-internal host in a public repo. Fixed on an unmerged branch (`67d4cfe`); currently still on `main`.

**Risk that is *not* present, and it matters:** there is no production deployment and no data. No Dockerfile, no compose, no migrations, no SQL dumps, and a DSN of `test-user:password@/go_admin`. **Migration cost is zero.** Nothing has to be preserved, backfilled, or kept backward-compatible. This substantially lowers the cost of the aggressive option.

---

## 7. Major product gaps

1. **No order lifecycle.** The central object of an e-commerce back-office has no status field. Pending → paid → packed → shipped → delivered, plus cancel and refund, is the primary workflow and it is entirely missing.
2. **No customer entity.** Customers exist as three denormalized columns on `Order` (§6 model review below). You cannot look up a customer, see their order history, or correct a misspelled email once without touching every order.
3. **Nothing creates orders.** No storefront, no checkout, no staff order-entry form, no import. The dashboard has no source of data.
4. **No inventory.** Products have no stock level, so nothing prevents overselling.
5. **No search or filtering.** Pagination is offset-only with a hardcoded page size of 5 and no query, sort, or date-range filter. Unusable past a few dozen records.
6. **No audit trail.** In a multi-staff tool with destructive actions and hard deletes, there is no record of who changed or deleted what.
7. **No soft deletes.** `DB.Delete` on structs without `gorm.DeletedAt` is a permanent row removal. Deleting a product orphans its order items' meaning; deleting a user destroys attribution.
8. **No notifications** — no order confirmation, no password reset, no email at all.
9. **No password reset or email verification.** A forgotten password is an unrecoverable account.
10. **No bulk operations** — no multi-select, no bulk status change, no import.
11. **No settings** — no tax rate, no currency (prices are bare `float32`/`float64`; **money should never be a float**), no store profile.

---

## 8. Recommended target architecture

Deliberately modest. This is a back-office for a small shop, not a platform. **Modular monolith, one deployable binary, one database.** No microservices, no event bus, no CQRS.

### Backend

| Concern | Recommendation | Why change (or not) |
|---|---|---|
| Language | **Go 1.25** | Keep. Go is a good fit and it's the repo's identity. Generics (1.18+) delete the `Entity` interface outright. |
| HTTP framework | **`net/http` + `chi` router** (replace Fiber) | The *only* substantive replacement I'm recommending, and not for fashion. Fiber is built on `fasthttp`, which does not implement `net/http` interfaces. That excludes the standard middleware ecosystem, OpenTelemetry's `otelhttp`, and `httptest` — the last one matters most, because it's why handler tests don't exist here. Go 1.22+ gave the standard mux method-and-pattern routing; `chi` adds grouping and is a thin, stable layer over the standard library. If you'd rather not touch this, Fiber is *defensible* — it works and it's maintained — but you'll keep paying for the ecosystem gap in testing and tracing. |
| Layout | `cmd/api/`, `internal/{domain,http,store,auth,config}` | Enforces the boundaries the current three folders only imply. `internal/` makes the boundary compiler-checked. |
| Database | **PostgreSQL 17** (replace MySQL) | Recommended *only because migration cost is zero* (§6). Postgres gives `numeric` for money (vs. today's `float32` — §7.11), native enums or check constraints for order status, partial and expression indexes, `jsonb` for audit payloads, and better `docker-compose` ergonomics. If you have a reason to prefer MySQL, keep it — this is a preference, not a defect, and nothing else in the plan depends on it. |
| Data access | **`sqlc`** (replace GORM) | GORM caused four distinct bugs here (§4m silent preload failure, §4p zero-value skip, §4o swallowed errors, §4w AutoMigrate). `sqlc` compiles hand-written SQL into typed Go — the query is visible and reviewable, the types are generated from the real schema, and errors are explicit. A reasonable alternative is keeping GORM with strict discipline; I'd rather remove the class of bug. |
| Migrations | **`golang-migrate`**, versioned SQL in `migrations/` | Replaces `AutoMigrate`. Reviewable, ordered, reversible. |
| Validation | **`go-playground/validator`** on typed request DTOs | Replaces `map[string]string` (§4x) and the single `Validate` method. |
| Auth | **Session cookie backed by a `sessions` table** (replace JWT) | JWTs cannot be revoked, which is wrong for an admin tool — you must be able to terminate a compromised staff session immediately. A server-side session is simpler *and* more capable here, and it removes the `SecretKey` global and the empty-secret hole (§4d). Keep bcrypt (cost 12); keep the httpOnly cookie; add `SameSite=Lax`, `Secure` in production, and a CSRF token for state-changing requests. |
| Authorization | **Permission middleware applied to route groups, not handlers** | Keeps the good permission-based model (§3.3) and fixes the fatal opt-in flaw (§4b). Permissions become a typed constant set, loaded once per request and cached on the request context. |
| Config | **`env` struct parsed at startup, fail-fast** | Replaces the hardcoded DSN (§4h) and the silent empty-secret path (§4d). Missing `SESSION_SECRET` must panic at boot, never at request time. |
| Logging | **`log/slog`** (stdlib), JSON in prod | Replaces `fmt.Println` (§4t). Request ID middleware, structured fields. |
| Errors | Typed domain errors mapped to HTTP at one boundary | Fixes §4o. Never return a raw driver error to a client (`auth_controller.go:46` currently does). |
| Testing | stdlib `testing` + `testify/require` + **`testcontainers-go`** | Real Postgres in integration tests. `httptest` for handlers (enabled by the `net/http` move). |

### Frontend

| Concern | Recommendation | Why |
|---|---|---|
| Build | **Vite 7** (replace CRA) | Non-optional. `react-scripts` is unmaintained. |
| Framework | **React 19 + TypeScript 5.x** | Keep React; align types with runtime; `createRoot`. |
| Routing | **React Router 7** (data router) | Keep. Add route-level loaders and a protected-route wrapper replacing `Layout`'s ad-hoc guard. |
| Server state | **TanStack Query** | Replaces hand-rolled `useEffect` + axios. Gives caching, loading/error states, and invalidation-on-mutation — the entire class of §4 frontend gaps. |
| Client state | **React Context** for auth only | Fixes the duplicate `/user` fetch (§4 frontend). No Redux; this app doesn't need it. |
| HTTP | Single typed API client, `fetch`, `credentials: 'include'`, base URL from env | Replaces five inline axios configs with hardcoded localhost and `//@ts-ignore`. |
| Contracts | **OpenAPI spec generated from Go, TS client generated from it** | The contract becomes generated, not hand-written — this is the direct fix for the already-drifted `interfaces/user.ts` (§4 frontend). Validated in CI. |
| Forms | **React Hook Form + Zod**, Zod schemas shared with generated types | Real client-side validation; currently there is none beyond `required`. |
| UI | **Tailwind + shadcn/ui** (replace Bootstrap CDN) | Bootstrap arrived via a copied example page, not a decision. shadcn/ui gives accessible primitives (Radix) you own in-repo — directly addresses §4 a11y findings. |
| Testing | **Vitest + Testing Library + Playwright** | Unit/component plus one E2E smoke path. |

### Repo / tooling

- **pnpm workspaces + Turborepo**, `apps/api` (Go), `apps/web` (React), `packages/api-client` (generated TS). Light-touch: keeps the monorepo the task asks for without adopting Nx-scale machinery for two apps.
- **Docker Compose** for Postgres + MailHog; `make dev` brings up everything.
- **golangci-lint** + **Biome** (lint+format, replacing Prettier alone); pre-commit hook.
- **GitHub Actions**: lint → test → build → OpenAPI drift check, on every PR. Keep the existing CodeQL workflow.
- **Deployment**: single container (multi-stage Dockerfile embedding the built SPA via `embed.FS`) + managed Postgres. Fly.io or Render. One artifact, one rollback.
- **Secrets**: `.env` locally (gitignored, with a committed `.env.example`), platform secret store in production. Never in source (§4h).

---

## 9. Recommended complete feature set

### Core / MVP — the product works end to end

**Auth & accounts**
- Login, logout, session expiry and renewal
- Password change with current-password confirmation
- Password reset via emailed token
- First-run bootstrap: seed an owner account from env or a `make seed` CLI (fixes §6.2)

**Staff & RBAC**
- Staff CRUD, invite by email, deactivate (not hard delete)
- Roles CRUD with a typed permission matrix
- Guard: cannot delete or demote the last owner; cannot edit your own role

**Catalog**
- Product CRUD, image upload with validation, draft/active/archived status
- Stock level with decrement on fulfilment
- Search by title, filter by status, sort by price/date

**Customers**
- A real `Customer` entity; auto-created or matched by email at order creation
- Detail view with order history and lifetime value

**Orders — the full lifecycle, not just CRUD**
- Create (staff entry form) → validate stock and price → persist with line snapshots
- Status machine: `pending → paid → packed → shipped → delivered`, plus `cancelled` and `refunded`, with only legal transitions permitted and each transition recorded with actor and timestamp
- Edit line items **only while pending**; immutable afterward
- Cancel with a reason and stock restoration
- Refund (full or partial) recorded as a `Payment` row, not a mutation of the order
- Per-order timeline of every state change and who made it
- Filter by status, date range, customer; search by order number or email

**Dashboard**
- Revenue over time (a *working* chart — §4j), order counts by status, recent orders, low-stock alerts

**Cross-cutting**
- Real loading, empty, and error states on every screen
- Server-side validation surfaced as per-field form errors
- 404 / 403 / 500 pages
- Optimistic concurrency (version column) so two staff editing one order don't silently overwrite each other

### Product-quality — it feels finished

- Sortable, filterable, paginated tables with URL-synced state (shareable links)
- Bulk selection and bulk status transitions
- CSV export of the *current filtered view*, generated to a temp file per request (fixes §4r)
- Global search (⌘K) across orders, products, customers
- Toast notifications on every mutation
- Confirmation dialogs on destructive actions, with undo where feasible
- Audit log: who changed what, when, with before/after values; filterable
- Soft delete plus restore for products, customers, and staff
- Transactional email: order confirmation, shipping notification, staff invite, password reset
- Full keyboard navigation, focus management, WCAG 2.1 AA contrast, screen-reader-tested tables and dialogs
- Genuine responsive design — a usable mobile order list, not a collapsed desktop grid
- Dark mode
- Order number as a human-readable, non-enumerable identifier (`ORD-2026-0001`), not a sequential primary key (§4s)
- Rate limiting on login and password reset
- Empty-state onboarding ("no products yet — add your first")

### Later / optional

- Public storefront or a checkout API, so orders originate outside the admin
- Payment provider integration (Stripe) with webhook-driven status
- Shipping carrier integration and label printing
- Product variants (size/colour) and categories
- Discount codes and promotions
- Multi-currency and configurable tax rules
- Saved views and scheduled email reports
- Two-factor authentication for staff
- Webhooks out to other systems
- Customer-facing order status page
- Multi-warehouse inventory

---

## 10. Milestone-based revival plan

Each milestone is independently reviewable and independently releasable. No milestone is "rewrite everything."

### M0 — Boots safely from a clean clone *(recommended first — see §12)*

**Goal:** `git clone && make dev` yields a running, seeded, non-compromised application. Close every critical security hole in the code that exists today.

**Concrete changes**
- Merge branch `fix/remove-internal-registry-paths` (module path + npm registry are already fixed there)
- `internal/config`: parse env into a typed struct, **fail fast** on missing `DB_DSN` / `SESSION_SECRET`; add `.env.example`
- `docker-compose.yml` for the database; `make dev`, `make test`, `make seed`
- **Fix §4a** — `SetPassword` hashes the actual password and returns its error
- **Fix §4b** — apply `IsAuthorized` as route-group middleware in `routes.go` covering products, orders, roles, permissions, and upload
- **Fix §4c** — registration assigns a non-privileged default role; the owner account is seeded (see *Documented behaviour change* below)
- **Fix §4d** — refuse to start with an empty secret; remove the token from the login response body
- **Fix §4e** — explicit CORS origin from config; `SameSite=Lax` + `Secure` in production
- **Fix §4f** — sanitize upload filenames (`filepath.Base` + random prefix), MIME allowlist, size cap, config-driven public URL
- **Fix §4g** — clear the body-supplied `Id` and `RoleId` before applying updates
- **Fix §4l** — correct pagination arithmetic; make page size a bounded query parameter
- **Fix §4m** — `Preload("Role")`
- **Fix §4n** — replace `fiber.Map` type assertions with a typed `RoleRequest` DTO
- **Fix §4j/k** — `Order.CreatedAt`/`UpdatedAt` → `time.Time`; drop `json:"-"` from customer names
- **Fix §4x** — typed request DTOs for the four `map[string]string` handlers, fixing `firstname`/`firstName`
- **Fix §4t** — remove the `fmt.Println` from the authorization path
- Dependency bumps: Go 1.25, `golang-jwt/jwt` → v5, Fiber, GORM
- Real `README.md`: what it is, how to run it, how to test it

**Frontend work:** minimal and deliberate — align `interfaces/user.ts` with the actual response shape, remove the `//@ts-ignore` directives, add error handling to `Login.tsx` so a failed login is visible. The frontend rebuild is M3; this is only enough to keep it honest.

**Backend work:** as above. **Database changes:** seed migration for roles and permissions; `orders.created_at`/`updated_at` become real timestamps.

**Tests** *(written before the fixes — they become the executable contract M1 rebuilds against)*: register→login→access round trip; login rejects a wrong password (this test fails today, proving §4a); unauthenticated request is rejected; a user lacking `edit_products` cannot `POST /products` (fails today, proving §4b); pagination `last_page` is correct at a partial final page (fails today, proving §4l); upload rejects a traversal filename.

**Documented behaviour change:** public registration currently grants admin. M0 changes it to grant a minimal role. This is a product behaviour change, not a refactor, and the task's rules require it be written down first — it will be recorded in `docs/decisions/0001-registration-default-role.md` before the code changes.

**Definition of done:** fresh clone → `make dev` → working seeded app; `make test` green; CI runs vet + test + web build on PRs; no credentials in source; every test above passes.

---

### M1 — Architecture foundation

**Goal:** a testable backend with reviewable schema. No user-visible change.

**Changes:** restructure to `cmd/api` + `internal/{domain,http,store,auth,config}`; introduce service and repository layers with interfaces; replace `AutoMigrate` with `golang-migrate` SQL migrations reproducing the current schema exactly; replace the `Entity` interface with a generic paginator; `log/slog` + request-ID middleware; typed domain errors with one HTTP mapping boundary; **decide Fiber vs. `net/http`+chi here and execute it in one pass**.

**Database:** migrations become the source of truth; all money columns → `numeric(12,2)`; `created_at`/`updated_at` on every table; soft-delete columns; indexes on `orders.email`, `orders.created_at`, `order_items.order_id`, `users.email`, `role_permissions`.

**Tests:** service-layer unit tests with mocked repositories; repository integration tests against a testcontainer Postgres; M0's HTTP tests still green.

**Done:** `go test ./...` runs with no external setup beyond Docker; every handler is under 30 lines; no package-level mutable globals.

---

### M2 — The order domain done properly

**Goal:** orders become a real lifecycle, not a CRUD table.

**Changes:** introduce `Customer`; add `OrderStatus` enum and a transition table enforcing legal moves; add `Payment`; give `OrderItem` a `ProductId` alongside its existing snapshot; add `orders.order_number` and `orders.version`; totals computed and persisted rather than recomputed on read (§4k); add `AuditLog` and write to it on every state change; rewrite `Chart` against real timestamps (§4j); rewrite `Export` to a per-request temp file with correct decimal formatting (§4r).

**Tests:** the transition matrix exhaustively (legal moves succeed, illegal ones 409); cancel restores stock; refund creates a `Payment` and never mutates history; the chart returns correct daily sums against seeded data.

**Done:** an order can be created, paid, packed, shipped, delivered, cancelled, and refunded through the API, with a complete audit trail and a working revenue chart.

---

### M3 — The frontend, rebuilt

**Goal:** every backend capability has a real screen.

**Changes:** Vite + React 19 + TS 5 + Tailwind/shadcn; move to `apps/web` under pnpm workspaces; generate the TS client from the OpenAPI spec (kills the hand-written `interfaces/`); TanStack Query for all server state; a single `AuthContext` replacing the duplicate `/user` fetches; protected route wrapper replacing `Layout`'s guard; React Hook Form + Zod on every form.

**Screens:** Login, Register, Forgot/Reset Password, Dashboard (working charts + KPIs), Products list/detail/form, Orders list/detail with timeline and status actions, Customers list/detail, Staff list/form, Roles with a permission matrix, Profile, Settings, plus 404/403/500. Every list gets loading, empty, and error states and URL-synced filters.

**Tests:** component tests for forms and tables; a Playwright E2E covering login → create product → create order → advance to shipped.

**Done:** no placeholder screens remain; no hardcoded URLs; `Users.tsx` shows real paginated data; the dashboard renders real revenue.

---

### M4 — Auth & security hardening

**Goal:** production-grade authentication.

**Changes:** JWT → server-side sessions with revocation; CSRF tokens; password reset and email verification (MailHog locally, provider in prod); rate limiting on auth endpoints; staff invite flow; session list with "sign out everywhere"; security headers (HSTS, CSP, `X-Content-Type-Options`); optional TOTP 2FA.

**Tests:** revoked session is rejected immediately; CSRF blocks a cross-origin state change; reset tokens are single-use and expire; rate limiter engages under brute force.

**Done:** a pen-test checklist for OWASP Top 10 passes; no credential is ever readable by JavaScript.

---

### M5 — Product-quality features

Bulk operations, global search, toasts, confirm/undo, audit log viewer, soft delete + restore, transactional email, a11y audit to WCAG 2.1 AA, responsive rework, dark mode, empty-state onboarding.

**Done:** an axe-core audit is clean, the app is usable on a phone, every mutation gives feedback.

---

### M6 — Production readiness

Multi-stage Dockerfile with the SPA embedded; `/healthz` and `/readyz`; OpenTelemetry traces and metrics; Sentry; automated backups with a documented restore drill; staging environment; deploy pipeline with migration gating and rollback; runbook; load test.

**Done:** deployed to a real URL, monitored, alerting, with a restore actually rehearsed.

---

## 11. Recommendation

# **B — Preserve the domain model and product intent; rebuild the implementation.**

**Not A (incrementally modernize).** The blocker is not that the code is old — it's that *there is no working product to incrementally improve*. Authentication is bypassed for every account (§4a), authorization is absent from 22 of 27 routes (§4b), public registration mints admins (§4c), the chart has never produced a correct result (§4j), orders cannot receive a customer name (§4k), profile updates blank user names (§4x), and the whole thing cannot start from a clean clone (§6.2). "Incremental" implies a working baseline to increment from. There isn't one. Patching each defect in place still leaves handlers that mix HTTP, business logic, and SQL with no testable seam — you'd finish the patches and still have to do M1.

**Not C (start over).** Three things genuinely deserve to survive, and discarding them would be waste: the entity decomposition is right, `OrderItem`'s price/title snapshot is a correct decision many experienced developers get wrong, and permission-based (not role-string) authorization is the right access model. `routes/routes.go` is also a clear, complete statement of intent that functions as a requirements document. A blank-page start would rediscover all four.

**What actually survives — be clear that these are *decisions*, not lines of code:**

| Survives | Where it lives now |
|---|---|
| Entity set: User/Role/Permission/Product/Order/OrderItem | `models/*.go` |
| Order-line snapshotting of title and price | `models/order.go:18-24` |
| Permission-based RBAC with a view/edit split | `middlewares/permission_middleware.go:34-48` |
| bcrypt + httpOnly cookie session | `models/user.go`, `controllers/auth_controller.go:78-89` |
| The API surface as a specification | `routes/routes.go` |
| Go + React + TypeScript + monorepo | repo-wide |
| Module-path and npm-registry fixes | branch `fix/remove-internal-registry-paths` |

Nearly every *file* gets rewritten. **The reason is the inverted security model and the absence of a testable seam — not the calendar.** If the code were four years old and merely dated but correct, the answer would be A.

**The zero-data fact is what makes B cheap.** There is no deployment, no migrations, no dumps, and a DSN of `test-user:password` (§6). Nothing must be backfilled or kept backward-compatible, so the DB, ORM, and framework choices can be made on merit alone. B here costs little more than A and delivers a foundation A cannot reach.

---

## 12. First milestone to implement, and why

### **M0 — "Boots safely from a clean clone."**

**Why this one, ahead of the architecture work:**

1. **It removes the emergency.** §4a means every account's password is `test`; §4b means a self-registered user owns the catalog and can grant themselves any permission. These are live, exploitable defects sitting in a public repo. They get fixed before anything is restructured.

2. **It's the precondition for every other milestone.** Right now no one — including a future you, or a contributor, or CI — can run this application. Until `make dev` works, no milestone can be reviewed, demoed, or verified. M1 through M6 all depend on it.

3. **It creates the safety net before the dangerous work.** The task's rules require adding tests around important behaviour *before* changing it. M0's tests (register/login, authorization, pagination) encode what the system is *supposed* to do, and several of them **fail on today's code** — which is precisely how the critical bugs get proven rather than asserted. Those same tests then hold M1's restructuring to the same contract. Skipping to M1 would mean rewriting the architecture with no test to tell you whether behaviour changed.

4. **It's small, self-contained, and genuinely releasable.** No framework swap, no database change, no frontend rewrite. It's bounded, reviewable in one sitting, and it leaves the repo in a strictly better state than it found it.

5. **It's honest about scope.** M0 does not pretend the architecture is fixed. It makes the existing application safe and runnable so the architectural work in M1 can proceed against something verifiable.

**One product decision inside M0 that changes behaviour** — registration currently grants admin (`auth_controller.go:32`); M0 changes it to grant a minimal role, with the owner seeded separately. That's the correct behaviour, but it *is* a product change, so per the task's rules the reasoning is written to `docs/decisions/0001-registration-default-role.md` before the code moves.

**One assumption worth stating, which does not block M0:** I've reconstructed the product as a *staff back-office for a small shop*, with orders entering via staff entry now and a storefront or checkout API later (§1.1). If the real intent was a customer-facing storefront, that changes M2's shape considerably — but M0 and M1 are identical either way, so there is no reason to wait on an answer.
