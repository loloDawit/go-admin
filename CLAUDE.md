# go-admin

Staff back-office for a small e-commerce shop, being rebuilt as a small-scale
microservice system: **Identity, Catalog, Orders**, behind a thin Go gateway,
with a React admin frontend.

**Authoritative documents**

| Document | Governs |
|---|---|
| `docs/PRD/SmallScaleMicroservicesArchitecture.md` | Architecture, milestones (M0–M8), product requirements. Milestone numbering here is canonical. |
| `docs/PRD/FrontendProductandDesignDirection.md` | Frontend craft, design process, M5 breakdown. |
| `docs/PRD/GoApplicationArchitectureandCodingConventions.md` | How Go code is structured **inside** a service: handlers, services, repositories, DTOs, packages, errors. |
| `docs/PRD/ArchitectureDecisionClarifications.md` | Precedence between the above, and the resolved decisions. **Read this when two documents disagree.** |
| `docs/ASSESSMENT.md` | Why the rebuild; the original defect catalogue. Its milestone plan is superseded by the PRD. |

**Precedence.** System, product, and milestone decisions → the microservices PRD.
Go structure inside a service → the Go conventions. Frontend craft → the frontend
direction. A lower-level document never redefines a higher-level boundary.

The `legacy-v1` tag is a **reference implementation, not a migration source**.
Each milestone deletes the legacy code it replaces. Do not invest in it.

---

## Engineering conventions

These are enforced by tests and by review. They are not style preferences.

**Errors split at the HTTP boundary.** Domain and service layers return sentinel
errors and know nothing about HTTP:

```go
var ErrProductNotFound = errors.New("product not found")
```

The HTTP boundary owns the client-facing mapping — sentinel to
`{code, message, status}` — in one place per service. `errors.New` is therefore
fine in domain and service code; what is forbidden is an **http/** package
inventing a client-facing error outside that central mapping. The AST guard
enforces the narrow rule, not the broad one.

A 5xx message never contains SQL, driver text, or a filesystem path. Wrap the
cause so it reaches the log, never the response body.

(The M0 convention banned `errors.New` everywhere. That was wrong: baking an HTTP
status into a domain error makes the domain know about transport. See
`docs/PRD/ArchitectureDecisionClarifications.md` §6.)

**Comments state constraints, not changes.** A comment earns its place only when
it names something a reader would break by tidying the code — "must be a map:
the struct form skips zero values". Do not restate the code. Do not narrate what
previous code did; git and the assessment hold that. Most functions need none.
This applies to tests and to YAML.

**Tests must discriminate.** A test written for a defect must fail against the
unfixed code, verified, before the fix lands. A test asserting only a status code
where the mechanism matters is not a regression test. When a test cannot fail,
say so rather than counting it as coverage.

**Configuration fails fast.** Anything an operator can tune lives in typed config
with validation. Missing or invalid configuration is fatal at startup, never a
surprise at request time. Validate against the real parser where one exists
rather than reimplementing its rules.

**Organize by business capability, not technical layer.** A capability owns its
package: `staff.go`, `handler.go`, `service.go`, `repository.go`, `postgres.go`,
`dto.go` together. No global `handlers/`, `services/`, `repositories/`, `models/`,
or `transformers/` directories. Extraction later becomes a directory move.

**Authorization is middleware, never a handler call.** A permission check a
handler must remember to make is a check that will be forgotten. Route-level
enforcement plus a coverage test that walks the live route table.

---

## Frontend design contract

### Product character

A professional back-office used repeatedly throughout the workday — not a SaaS
landing page. The character is **quiet, precise, efficient, information-dense
where useful, visually restrained**.

### Rules

1. Inspect existing components before creating new ones.
2. Reuse primitives. A screen that introduces its own button is a defect.
3. Use design tokens, never arbitrary values.
4. Keep one spacing scale and one radius scale.
5. Use semantic colours.
6. Maintain a coherent typography hierarchy.
7. Provide visible keyboard focus states.
8. Design the complete set of states: loading, empty, error, disabled, success.
9. Do not introduce screen-specific patterns that conflict with existing
   application conventions.

### Avoid

Excessive rounded cards, pills, and badges; gratuitous gradients;
glassmorphism; decorative hero sections, blobs, and background effects;
excessive shadows; unnecessary animation; icons where text is clearer; wrapping
every section in a card; inconsistent spacing or radius values; one-off
components where a primitive exists.

The goal is not minimalism. It is a coherent, professional interface.

### Tooling

Use the `frontend-design` skill when designing or substantially reshaping
screens. It informs visual direction, typography, composition, hierarchy, and
interaction; it supplements this contract and does not override it.

21st may be used as a reference or component source, adapted to this project's
tokens and primitives. The application must not become a collection of
unrelated copied components.

### A frontend task is not done when it compiles

Substantial frontend changes must be **rendered and visually inspected** before
completion, using Playwright:

1. implement, then run the application
2. open the flow and exercise the real interaction
3. inspect the desktop state and a narrower viewport
4. critique hierarchy, spacing, alignment, typography, density, overflow,
   responsive behaviour, interaction states, and consistency with adjacent
   screens
5. fix and rerun

Ask explicitly: *does this look like one intentional product, or a collection of
generated components?* Iterate if the latter.

Compilation is not visual validation. Passing tests is not visual validation.
The rendered application is the source of truth for frontend quality.
