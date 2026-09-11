I have an old side project in this repository that I built roughly four years ago. It was fairly amateur at the time and was never developed into a complete product.

The repository is a monorepo containing both the frontend/client and backend/API.

I want you to take a deep look at the entire repository and determine whether this project is worth reviving. Do not just get it compiling, update dependencies, or clean up a few files. Treat this as if you inherited an abandoned product and were responsible for turning it into a well-designed, production-quality application.

Start by understanding the existing system before making major changes.

Inspect the entire repository, including:

* frontend architecture
* backend architecture
* monorepo/package structure
* database/data model
* APIs and contracts between frontend and backend
* authentication/authorization, if present
* configuration and environment management
* third-party integrations
* build tooling
* deployment configuration
* tests
* CI/CD
* documentation
* dependency health
* security issues
* dead or unfinished code
* partially implemented features
* TODOs and abandoned ideas
* duplicated or unnecessary abstractions
* code that no longer makes sense with the current ecosystem

I also want you to infer what the original product was intended to do from the code, models, routes, UI, README, commits, and naming.

Do not assume the existing implementation is the right architecture just because it already exists.

### Phase 1 — Understand the product

First, explain what you believe this application is supposed to be.

Describe:

1. The core product idea.
2. The intended users.
3. The main user journeys.
4. The major entities/domain concepts.
5. What functionality currently exists.
6. What functionality appears unfinished.
7. What is missing for this to feel like a complete product rather than a prototype.

If the original product direction is unclear, reconstruct the most coherent product vision supported by the repository.

### Phase 2 — Technical assessment

Perform a detailed technical review of both frontend and backend.

For each major area, classify it as:

* Keep
* Refactor
* Replace
* Remove
* Missing

Explain why.

Evaluate things such as:

* framework/version age
* architectural boundaries
* state management
* API design
* domain modeling
* database schema
* migrations
* error handling
* validation
* authentication
* authorization
* security
* logging
* observability
* testing strategy
* performance
* accessibility
* responsive design
* frontend component quality
* backend service structure
* configuration
* secrets handling
* developer experience
* local environment
* production readiness

Pay special attention to code that reflects beginner-level design decisions that should not be carried forward.

Do not over-engineer the replacement either. Prefer a simple modern architecture appropriate for the size of the product.

### Phase 3 — Determine whether to revive or rebuild

Give me a clear recommendation:

**A. Incrementally modernize the existing application**

or

**B. Preserve the useful domain logic/data model but rebuild major parts**

or

**C. Start over**

Do not recommend a rewrite merely because the code is old.

Base the decision on:

* how much useful code exists
* quality of the domain model
* framework compatibility
* complexity of migration
* test coverage
* security risk
* maintainability
* cost of modernization versus replacement

Explain exactly what should survive.

### Phase 4 — Define the actual product

This is important:

I do NOT want this to remain a toy CRUD application.

Design a complete version of the product.

Based on the repository and the apparent original idea, propose the feature set the application should have today.

Break features into:

**Core / MVP**
Features required for the product to actually work end to end.

**Product-quality**
Features needed for it to feel polished and usable by real users.

**Later / Optional**
Features worth adding after the core product is solid.

Think through the complete lifecycle of each primary user journey.

For example, don't stop at:

"User can create X."

Think through:

create → validate → persist → display → edit → state transitions → permissions → errors → notifications → history → deletion/cancellation → edge cases.

Identify missing workflows that the old implementation never considered.

### Phase 5 — Modern architecture proposal

Propose the architecture you would use if we revive it today.

Keep the monorepo unless there is a strong reason not to.

Recommend:

* monorepo tooling
* frontend framework
* backend framework/runtime
* database
* ORM/query layer
* API style
* auth solution
* validation strategy
* shared types/contracts
* testing stack
* linting/formatting
* local development setup
* CI/CD approach
* deployment model
* environment/secrets management
* logging/observability

For every significant technology replacement, explain why it is worth changing rather than simply following current trends.

Avoid unnecessary microservices. This should probably remain a modular monolith unless the domain clearly requires otherwise.

### Phase 6 — Data model and API review

Inspect the existing schema carefully.

Identify:

* entities that are modeled incorrectly
* missing entities
* bad relationships
* duplicated data
* weak identifiers
* missing timestamps/audit fields
* inappropriate nullable fields
* enum/state-machine candidates
* missing indexes
* lifecycle/state issues

Propose an improved domain model.

Also review the API surface.

Point out:

* inconsistent endpoints
* weak request/response contracts
* business logic leaking into controllers/routes
* validation gaps
* security problems
* APIs that should be consolidated or redesigned

If useful, propose the new endpoint structure or contract.

### Phase 7 — UI/UX review

Go through the frontend as a product, not just as code.

Identify:

* unfinished screens
* confusing flows
* inconsistent navigation
* missing empty/loading/error states
* poor forms
* accessibility problems
* weak mobile/responsive behavior
* duplicated components
* missing design system primitives

Then describe the screens and flows a complete version of the application should contain.

Do not spend time making the current UI prettier if the underlying flow is wrong.

### Phase 8 — Build a concrete revival roadmap

Create an implementation roadmap using milestones.

For example:

M0 — repository stabilization
M1 — architecture foundation
M2 — core domain workflow
M3 — complete frontend workflow
M4 — authentication/security
M5 — product-quality features
M6 — production readiness

But derive the actual milestones from this repository.

For every milestone include:

* goal
* concrete changes
* frontend work
* backend work
* database changes
* tests
* definition of done

Make the milestones independently reviewable and preferably releasable.

Avoid giant "rewrite everything" milestones.

### Phase 9 — Begin the revival

After completing the assessment, do not stop at recommendations.

Start implementing the revival.

Choose the highest-leverage foundational milestone and work through it.

Before making destructive architectural changes, document the reasoning.

Preserve useful behavior and data where practical.

As you work:

* keep frontend and backend contracts aligned
* add tests around important existing behavior before changing it
* remove dead code when confidently identified
* avoid compatibility layers unless they serve a real migration purpose
* update documentation alongside architecture changes
* keep commits/changes logically scoped

Do not silently change product behavior just to make implementation easier.

### Important working principles

1. **Understand before rewriting.**
2. **Treat the frontend and backend as one product.**
3. **Focus on complete end-to-end workflows, not isolated components.**
4. **Prefer domain clarity over clever abstractions.**
5. **Do not preserve bad architecture because it already exists.**
6. **Do not replace working architecture just because newer tools exist.**
7. **Do not reduce this exercise to dependency upgrades.**
8. **Do not build a generic CRUD app.**
9. **Think about real-world error cases and lifecycle states.**
10. **Aim for something I could realistically continue developing and eventually deploy.**

### First deliverable

Before changing significant code, give me a repository assessment containing:

1. Your understanding of the product.
2. A map of the current architecture.
3. What is good and worth keeping.
4. What is weak or amateur and should change.
5. What is unfinished.
6. Major technical risks.
7. Major product gaps.
8. Recommended target architecture.
9. Recommended complete feature set.
10. A milestone-based revival plan.
11. Your recommendation: modernize, partial rebuild, or full rebuild.
12. The first milestone you recommend implementing and why.

Use concrete examples from the repository. Reference actual files, modules, routes, models, and components rather than giving generic software-engineering advice.

Once the assessment is complete, proceed with the first milestone unless you discover a decision that genuinely requires product input from me.
