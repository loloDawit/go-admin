# go-admin

Staff back-office for a small e-commerce shop, being rebuilt as a small-scale
microservice system: **Identity, Catalog, Orders**, behind a thin Go gateway,
with a React admin frontend.

**Authoritative documents.** `docs/` is deliberately untracked — these are
internal planning documents, kept local rather than published with the code. A
fresh clone will not have them; ask for them before assuming a decision is
unrecorded.

| Document | Governs |
|---|---|
| `docs/PRD/SmallScaleMicroservicesArchitecture.md` | Architecture, milestones (M0–M8), product requirements. Milestone numbering here is canonical. |
| `docs/PRD/FrontendProductandDesignDirection.md` | Frontend craft, design process, M5 breakdown. |
| `docs/PRD/GoApplicationArchitectureandCodingConventions.md` | How Go code is structured **inside** a service: handlers, services, repositories, DTOs, packages, errors. |
| `docs/PRD/ArchitectureDecisionClarifications.md` | Precedence between the above, and the resolved decisions. **Read this when two documents disagree.** |
| `docs/ASSESSMENT.md` | Historical: why the rebuild. The application it catalogues no longer exists. |
| `docs/CHECKPOINT.md` | **Run before every milestone PR.** Manual verification plus the repository hygiene check. |

**Precedence.** System, product, and milestone decisions → the microservices PRD.
Go structure inside a service → the Go conventions. Frontend craft → the frontend
direction. A lower-level document never redefines a higher-level boundary.


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

**Every service owns one error registry.** `internal/errs` holds the service's
sentinels *and* its operation strings. No `errors.New` or `fmt.Errorf` with a literal
message anywhere else — a message that appears inline is a message nobody can grep,
audit, or keep consistent. Capability packages reference `errs.ErrX` and wrap with
`errs.Wrap(errs.OpX, err)`. `internal/httperr` maps the registry's sentinels to
`{code, message, status}` and remains the only place a status code is decided.

**SQL lives in one file per capability, never inline at the call site.** A capability's
queries are named constants in its `queries.go`. Inline SQL scattered through method
bodies cannot be reviewed as a set, duplicates silently, and turns a schema change into
a search-and-hope. If two queries differ only in a WHERE clause, they share a base
constant.

**Anything an operator could reasonably tune is configuration, not a constant.** Body
size caps, timeouts, costs, TTLs, limits. A `const maxBodyBytes = 1 << 20` is a value
someone will need to change in production and cannot.

**Code self-explains; comments are the exception.** Add one only where the code
genuinely cannot say it — a constraint a reader would break by tidying, an external
requirement, a counter-intuitive ordering. One line, occasionally two. Three or more
consecutive comment lines is a defect to justify or delete. Most functions need none.

Never write: why a decision was made in conversation; what the code used to do;
narration of the next line; forward references to future tasks; or runtime behaviour
you have not observed. Reviewers flag comment bloat as Important, not Minor.

```go
// hmac.Equal, not ==: a timing-variable compare leaks the signature.   <- earns its place
```

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

**A milestone is not done when the tests pass.** It is done when a human has
booted it from an empty volume, driven the product path by hand, broken it on
purpose, and confirmed nothing unfit for a public repository is tracked. See
`docs/CHECKPOINT.md`.

**Authorization is middleware, never a handler call.** A permission check a
handler must remember to make is a check that will be forgotten. Route-level
enforcement plus a coverage test that walks the live route table.

---

## Frontend design contract

### Foundation

**shadcn/ui on Tailwind v4**, following shadcn's `dashboard-01` block as the
reference implementation. Components live in `apps/web/src/ui/shadcn/` as owned
source, not as a dependency. Radix supplies the behaviour that is always subtly
wrong hand-rolled: focus traps, listbox semantics, roving tabindex.

**The style is `new-york-v4`, and it matters.** `components.json` was once set
to `radix-nova`, whose `CardFooter` bakes in `border-t bg-muted/50` — every card
in the application carried a rule and a grey band across its middle, and no
amount of adjusting the card would have fixed it. When something looks wrong
across many components at once, check the style before the component. Fetch the
block's real source from
`https://ui.shadcn.com/r/styles/new-york-v4/dashboard-01.json` rather than
inferring it from a scaffold.

| Thing | Where |
|---|---|
| Design tokens | `src/styles/theme.css` — the only place a colour, size or radius is defined |
| Library components | `src/ui/shadcn/` (26) — generated, then adapted |
| Project components | `src/ui/` — DataTable, BulkBar, ColumnVisibility, EmptyState, StatusBadge, Money, RowActions, FilterSelect, Field, Dialog, Alert, Pagination |
| Page patterns | `src/patterns/` — ListPage, FormPage, PageBlock, PageHeader, SectionCard, StatCard |
| Status → tone | `src/app/statusTones.ts` — one typed map per domain |

There are no CSS Modules. A screen reaching for a stylesheet is doing something
the system should absorb.

Tailwind v4 is CSS-first: no `tailwind.config.js`. The theme has two layers
because shadcn requires both — `:root` (and `.dark`) hold the bare names its
components read inside arbitrary values, and `@theme inline` republishes them as
Tailwind's `--color-*` scale. `@custom-variant dark` binds `dark:` utilities to
the class rather than the media query, or an explicit choice is ignored.

### Visual identity

Taken wholesale from `dashboard-01`: **Geist Variable**, the neutral oklch ramp
with a near-black primary, `--radius: 0.625rem` with a derived scale, Tailwind's
own type sizes under this project's semantic names, KPI cards carrying
`bg-gradient-to-t from-primary/5 to-card` with `shadow-xs`, and the shell as
`<AppSidebar variant="inset" />` inside a `SidebarProvider` that sets
`--sidebar-width` and `--header-height`.

**Status badges outline the chip and colour the icon**, never the chip: a column
of filled colour competes with the data beside it.

Two deliberate divergences, both asked for: the sidebar collapses to an **icon
rail** where the block uses `offcanvas` and hides outright, and status keeps
semantic hues the neutral palette does not have.

`* { @apply border-border }` in the base layer is load-bearing. Tailwind's bare
`border` sets a width and leaves the colour at `currentColor`; without that rule
every border in the application paints near-black.

**Status keeps its hues.** The neutral palette has none, and a colour telling a
reader an order was refunded is information, not decoration.

### Hierarchy

1. The most important value on a screen is the largest text on it.
2. A screen has exactly one filled button. A destructive action that opens a
   confirmation uses `danger`, which is quiet; `dangerSolid` is the filled
   button *inside* that confirmation.
3. Hierarchy must survive in greyscale, and in both themes.
4. Section boundaries are a change of surface, not only a hairline rule.

### Layout

5. **The frame fills the viewport and content fills the frame.** Never cap the
   shell. Capping it made the application look shrunk on a workstation monitor,
   twice. Two tests in `shell.spec.ts` guard this.
6. A form bounds its own measure; a table does not.
7. **Every table declares its column widths and which single column grows.**
   Where the growing column holds right-aligned content the rest stay clustered.
   Where none declares `grow`, the first column without a width takes the role.
8. A container query cannot query the element it is declared on. `@container/x`
   goes on a parent; the `@lg/x:` classes go on the child.

### The table contract

One definition owns interaction; a screen declares only what is true of its
data. Every list: the whole row opens its detail; row actions sit behind one
trigger in a trailing column; every column the server allowlists is sortable and
no other is; pagination, column visibility and the filter bar are identical
everywhere; the header sticks. Selection chrome ships only with a bulk action
behind it, and a bulk run reports what happened — "6 archived, 1 failed" names
the failure rather than claiming an unverified success.

### Composition

9. Screens compose a pattern; they do not lay themselves out.
10. Reuse primitives. A screen that introduces its own button is a defect.
11. Use tokens and Tailwind utilities, never arbitrary colour or size values.
12. Visible keyboard focus states.
13. Design the complete set of states: loading, empty, error, disabled, success.
    A list filtered to nothing is a *different screen* from an empty one. Every
    mutation confirms itself with a toast naming what happened in the same words
    the button used. Design the sparse case: one data point is not a trend, and
    plotting it alone reads as a broken chart.
14. `/kit` renders every primitive in every state.

### Testing the frontend

The suite runs against the dev server for speed, but **the authoritative run is
the gateway-served build**:

```bash
docker compose -p <project> -f deploy/compose/docker-compose.yml up -d --no-deps --build gateway
BASE_URL=http://localhost:8080 npx playwright test
```

Failures seen only against the dev server are artifacts: StrictMode double-fires
effects, breaking request-count budgets, and a slower first paint widens mount
races. After installing a dependency, delete `node_modules/.vite` or the dev
server serves a stale pre-bundle.

`playwright.config.ts` matches specs by glob; it once matched a filename list,
so a new spec silently never ran.

**`getByRole` matches an accessible name by case-insensitive substring.** This
has broken tests five times here: a toast saying "Account deactivated" answers
to "Deactivated", and an empty list's "No orders yet" heading answers to
"Orders". Assert `level` on headings and `exact` where names can collide.

Radix is not the DOM the old suite assumed: `selectOption` and `toHaveValue` do
not apply to a listbox, and `dialog[open]` finds nothing. `e2e/select.ts` holds
the replacements, including `settled()` — a bare `aria-busy` count also passes
before React mounts, which is a gate that cannot fail.

Never assert a literal palette value; assert the behaviour the feature claims.

`e2e/sweep.spec.ts` walks every route across both themes and four widths,
checking horizontal overflow, borders painting `currentColor`, and console
errors. It is paced: a hundred page loads back to back trip the gateway's own
rate limiter at fifty requests a second.

A lazy route or component mounts after `settled()` returns — wait for the thing
itself. Hover a real data point rather than the computed centre of a plot; the
centre need not sit in an active band, and it breaks the moment layout changes.

### A frontend task is not done when it compiles

Substantial frontend changes must be **rendered and visually inspected** before
completion, using Playwright:

1. implement, then run the application
2. open the flow and exercise the real interaction
3. inspect the desktop state and a narrower viewport
4. critique hierarchy and emphasis, measure and column widths, surface
   layering, contrast of the active navigation state, spacing, alignment,
   typography, density, dead space, overflow and what sits inside a scroll
   region, responsive behaviour, interaction states, and consistency with
   adjacent screens
5. fix and rerun

Ask both questions explicitly:

> *Does this look like one intentional product, or a collection of generated
> components?*

> *With two seconds and no colour, what does the eye land on first — and is
> that the most important thing on the screen?*

The first detects incoherence. The second detects flatness, which a uniformly
bland screen passes the first question with ease.

Compilation is not visual validation. Passing tests is not visual validation.
The rendered application is the source of truth for frontend quality.
