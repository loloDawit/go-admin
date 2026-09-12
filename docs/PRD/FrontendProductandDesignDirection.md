## Frontend Product and Design Direction

The frontend should feel like a mature operational application used repeatedly throughout the workday, not a generic SaaS landing page or AI-generated dashboard.

The frontend implementation must prioritize:

* clear information hierarchy
* restrained visual design
* efficient use of screen space
* consistent typography and spacing
* predictable navigation
* reusable application primitives
* accessible interaction states
* strong loading, empty, error, and success states
* responsive behavior without sacrificing desktop information density

The project should deliberately avoid common AI-generated UI patterns unless they provide a clear functional benefit.

Avoid:

* excessive rounded cards
* excessive pills and badges
* gratuitous gradients
* glassmorphism
* large decorative hero sections
* decorative blobs and background effects
* excessive shadows
* unnecessary animations
* icons where clear text is more appropriate
* wrapping every section in a card
* inconsistent spacing or radius values
* one-off components when an existing primitive can be reused

The goal is not minimalism for its own sake. The goal is a coherent, professional back-office interface.

---

## Frontend Design Tooling

Frontend work should use an agent-assisted visual development workflow rather than relying solely on text prompts and generated code.

### Anthropic Frontend Design Skill

Claude Code should use Anthropic's `frontend-design` skill/plugin when designing or substantially modifying frontend screens.

Its role is to help Claude reason about:

* visual direction
* typography
* composition
* hierarchy
* spacing
* interaction design
* component relationships
* avoiding generic generated aesthetics

The skill supplements the project's design rules; it does not override them.

Project-specific conventions defined in `CLAUDE.md` remain authoritative.

---

### Project Design Contract

`CLAUDE.md` must contain a dedicated frontend design contract.

At minimum it should define:

#### Product character

The application is a professional back-office management system for a small business.

The desired visual character is:

* quiet
* precise
* efficient
* polished
* information-dense where useful
* visually restrained

#### Component rules

Claude must:

1. inspect existing components before creating new ones
2. reuse primitives whenever possible
3. use design tokens rather than arbitrary values
4. maintain a consistent spacing scale
5. maintain a consistent radius scale
6. use semantic colors
7. maintain a coherent typography hierarchy
8. provide visible keyboard focus states
9. design complete loading, empty, error, disabled, and success states
10. avoid creating screen-specific UI patterns that conflict with existing application conventions

#### Design review rule

A frontend task is not complete when the code compiles.

Significant frontend changes must be rendered and visually inspected before completion.

---

### 21st Component Reference

21st may be used as a design and implementation reference when Claude needs a high-quality component or interaction pattern.

Appropriate examples include:

* application navigation
* data tables
* filters
* search interfaces
* dialogs
* forms
* settings screens
* command palettes
* empty states
* detail panels
* pagination
* dashboard layouts

Claude should use 21st as inspiration or as a component source where appropriate, but should adapt components to the project's design system.

The application must not become a collection of unrelated copied components.

Existing application primitives and design tokens take precedence.

---

### Playwright Visual Feedback Loop

Playwright should be part of the normal frontend implementation workflow.

For substantial frontend changes, Claude should:

1. implement the requested screen or interaction
2. run the application
3. open the relevant flow using Playwright
4. exercise the actual user interaction
5. inspect the rendered desktop state
6. inspect an appropriate narrower/mobile viewport
7. capture screenshots where useful
8. critique the result
9. fix visual or interaction problems
10. rerun the flow

The visual review should specifically inspect:

* hierarchy
* spacing
* alignment
* typography
* information density
* overflow
* responsive behavior
* interaction states
* consistency with adjacent screens
* accessibility problems visible through interaction
* generic or unnecessary AI-generated visual patterns

Claude should not consider a screen complete solely because tests pass.

---

# M5 Breakdown — Frontend Replacement and Design System

> **Scope and acceptance for M5 are defined in**
> `docs/PRD/SmallScaleMicroservicesArchitecture.md` §16 M5. This section defines
> *how* the work is done — the phases, the design process, and the quality gate.
> It does not restate the screen list or the acceptance criteria.

## Objective

Replace the legacy frontend with a modern React application while establishing a
coherent frontend architecture and a repeatable design process.

The implementation should demonstrate that the application can evolve without
each new feature inventing its own interface conventions.

`clients/` is deleted as part of this milestone.

## Technology

* Vite
* React
* TypeScript
* React Router
* TanStack Query
* React Hook Form + Zod
* a typed API client generated from each service's OpenAPI specification
* Playwright
* the `frontend-design` skill
* 21st component references where appropriate

Prefer a small set of well-understood dependencies over a large UI framework
adopted without a clear need.

---

## M5.1 — Frontend Foundation

Establish the application shell, navigation, and routing, then the primitives
every later screen depends on:

* typography, spacing, radius, and semantic colour tokens
* form primitives and button variants
* table primitives
* feedback and alert primitives
* dialog conventions
* page-header conventions
* loading, empty, and error states

Build shared primitives before any screen-specific implementation. A screen that
introduces its own button is a defect in this milestone, not a detail.

---

## M5.2 — Typed API Integration

Generate the API client from each service's OpenAPI specification rather than
hand-writing request types. The frontend consumes three services through the
gateway; hand-maintained types across three contracts will drift.

Separate cleanly:

* HTTP/API concerns
* application state
* page composition
* reusable UI primitives
* feature-specific components

API failures map to intentional user-facing error states. The services return a
stable `{code, message}` error shape — surface `message`, switch on `code`, and
never show a raw transport error.

---

## M5.3 — Core Application Screens

Implement complete workflows for Identity, Catalog, and Orders.

Each screen supports the full operational state its workflow requires, not a
visual prototype. Lists support URL-synchronised filters and search, pagination,
and loading/empty/error states. Navigation state is recoverable from the URL
wherever practical.

---

## M5.4 — Visual Quality Gate

Before a primary page or workflow is accepted:

1. apply the project's design contract
2. use `frontend-design` while designing substantial UI
3. consult 21st when a stronger established component pattern would help
4. implement the screen
5. render it with Playwright
6. visually inspect it
7. correct visual inconsistencies
8. validate responsive behaviour
9. validate keyboard interaction where applicable

The review must explicitly ask:

> Does this look like an intentional product designed as part of the same
> application, or like a collection of generated components?

If the latter, iterate before considering the task complete.

---

## M5.5 — End-to-End Workflow

Playwright coverage for the primary happy path:

```text
login
  -> Catalog: create or update a product
  -> Orders: inspect an order
  -> update order status
  -> confirm the updated state
```

The test drives the browser, not the APIs. Add tests for the failure and
validation states discovered while implementing the workflow.

---

# Definition of Done for Frontend Tasks

For frontend work, the general repository Definition of Done is extended with the following requirements.

A frontend task is done only when:

1. functionality works
2. automated tests pass
3. existing primitives were considered before introducing new ones
4. design tokens are used instead of arbitrary styling where appropriate
5. loading/error/empty states are handled
6. the feature has been rendered in the browser
7. the result has been visually reviewed
8. obvious responsive issues have been corrected
9. the implementation is consistent with adjacent application screens
10. no unnecessary AI-generated decorative patterns were introduced

Compilation is not visual validation.

Passing tests is not visual validation.

The rendered application is the final source of truth for frontend quality.
