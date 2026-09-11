# ADR 0001 — Remove public registration; seed the owner, invite staff

**Date:** 2026-09-11
**Status:** Accepted
**Milestone:** M0

## Context

`POST /api/v1/register` is a public route (`routes/routes.go:11`) and hardcodes `RoleId: 1` (`controllers/auth_controller.go:32`). Git commit `a97be99 add admin role for register call` confirms role 1 is the admin role.

The consequence: **anyone who can reach the API can mint themselves an administrator account.** Combined with the missing authorization on 22 of 27 routes (ASSESSMENT §4b), an anonymous attacker gains full control of the catalog, every order, and the role/permission system.

There is a second, independent problem. Nothing ever creates role 1. On a clean database, registration fails on a foreign-key violation and returns a raw GORM error as JSON. **The application cannot bootstrap from a fresh clone** (ASSESSMENT §6.2).

These two problems share a root cause: the application has no concept of *how the first privileged account comes to exist*, so registration was made to do double duty as both bootstrap and signup.

## Decision

**Remove public registration entirely.**

1. `POST /api/v1/register` is deleted. The route, the `Register` controller, and the frontend `pages/Register.tsx` are removed.
2. The first account is created by an operator via `make seed`, which reads `OWNER_EMAIL` and `OWNER_PASSWORD` from the environment and creates the roles, the permission set, and one owner user. It is idempotent.
3. All subsequent staff are created by an authenticated admin through `POST /api/v1/users`, which requires the `edit_users` permission. The admin supplies an initial password, which the new user changes via the existing `PUT /api/v1/user/password`.

## Why not the alternatives

**Keep public signup but assign a minimal role.** This was the original plan in ASSESSMENT §10. It closes the privilege-escalation hole but leaves an unauthenticated write endpoint on an internal back-office tool — anyone who finds the URL can still create accounts and consume database rows. For a tool whose entire user population is "staff of one shop", public signup has no legitimate use.

**First-registrant-becomes-owner, then self-close.** Elegant bootstrap, but it introduces a race (two simultaneous first registrations) that needs careful locking to get right, and it leaves a public endpoint whose behaviour depends on hidden global state — hard to reason about and hard to test. The seed command achieves the same bootstrap with none of that.

## Consequences

**Accepted costs**
- Email-based invites are out of scope for M0 (email infrastructure arrives in M4). Until then an admin communicates the initial password out of band. This is a known, documented gap, not an oversight.
- `pages/Register.tsx` and `pages/Register.css` are deleted. `pages/Login.tsx` gains a line directing users to contact their administrator.
- Anyone currently relying on `/register` — in practice, nobody; the endpoint has never worked on a clean database — loses it.

**Benefits**
- The privilege-escalation hole is closed completely rather than narrowed.
- Bootstrap becomes explicit, idempotent, and testable.
- The API surface shrinks by one unauthenticated write endpoint.

## Behaviour change notice

This changes product behaviour: a flow that previously existed (self-service signup) is being removed, not refactored. Per the revival task's working principles, this is documented here **before** the code changes, and it is a deliberate product decision confirmed with the repository owner on 2026-09-11 — not a change made to simplify implementation.

## Follow-up

- M4 replaces the admin-supplied initial password with an emailed, single-use, expiring invite token.
