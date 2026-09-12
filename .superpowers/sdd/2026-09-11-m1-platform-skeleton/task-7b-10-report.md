# Task 7b/10 report: gateway timeout gap + architecture guards

## Status
Complete.

## Part A — gateway timeout

### A1: test that a slow upstream trips the deadline
`TestSlowUpstreamTripsTheDeadlineAndReturnsGatewayTimeout` in
`services/gateway/internal/routing/routing_test.go`: an `httptest.Server` blocks
on `<-r.Context().Done()` (with a 2s fallback that fails the test if it never
fires), `routing.New` is given a 50ms timeout, and the test asserts: elapsed
time < 500ms (not hanging), the upstream handler actually observed
cancellation (not leaking a goroutine/connection), the response is
`504 gateway_timeout`, and the body contains none of the upstream's URL, host,
`"deadline"`, or `"dial tcp"`.

**Confirmed the test fails when it should:** raising the timeout constant to
3s (above the 2s sleep) produces:
```
routing_test.go:100: upstream handler was not canceled when the gateway's deadline fired
routing_test.go:118: request took 2.0019015s; the gateway must not wait past its own timeout
--- FAIL: TestSlowUpstreamTripsTheDeadlineAndReturnsGatewayTimeout (2.00s)
```
Reverted immediately after.

### A2: does a timeout deserve its own status?
Yes — implemented `504 gateway_timeout`, distinct from `502
upstream_unavailable`. `ReverseProxy.ErrorHandler` now checks
`errors.Is(err, context.DeadlineExceeded)` before falling through to the
existing 502 path. This was clean to do: `Transport.RoundTrip` returns
`context.Cause(ctx)` directly (no `*url.Error` wrapping to unwrap) when the
per-request `context.WithTimeout` in `routing.New` expires, so `errors.Is`
reliably discriminates it from a refused/reset connection. `context.Canceled`
(client disconnected) deliberately still falls through to 502, since that is
not the upstream being slow.

Caveat worth stating honestly: 504 here means "no response arrived within the
deadline," which also covers a upstream that never responds at all (e.g.
blackholed) — so the distinction is "slow-or-unresponsive vs. refused/reset,"
not a perfect "slow vs. down." That's still a materially different operator
signal (retry/backoff vs. investigate why the process is down or the port is
closed), so it's worth keeping. No host, port, or dial text appears in either
body; both are covered by the existing and new leak-assertions in
`routing_test.go`.

### A verification
`gofmt -l services platform test`, `go vet ./services/... ./platform/...
./test/arch/...`, `go build ./...`, and `go test ./services/... ./platform/...
./test/arch/ -count=1` are all clean, including the pre-existing
`TestUnreachableUpstreamReturns502WithoutLeakingTheAddress`, which still
passes unchanged (a refused dial is not `context.DeadlineExceeded`).

## Part B — architecture guards

`test/arch/imports_test.go`, following the brief with the corrections below.

### B1: complete legacy list + prefix-matching correctness
Added `clients` to `legacyRoots` (present in the brief's own description of
the repo root but missing from its list). Replaced the brief's
`modulePath+"/internal/"` (trailing slash, inconsistent with the other
entries) with a shared `hasPathPrefix(imported, prefix)` helper that matches
`imported == prefix || strings.HasPrefix(imported, prefix+"/")` for every
entry uniformly. Verified by mutation test 2 below that this cannot be fooled
by `platform/httpx` or `services/gateway/internal/httpx` sharing the trailing
segment name with the root's `internal/httpx` — only the true legacy path
matches.

### B2: domain-purity guard stays filename-only
Confirmed current `platform/` filenames (`captured.go`, `logger.go`,
`envelope.go`, `pool.go`, `requestid.go`, and their `_test.go` counterparts)
don't collide with the banned words, so the guard passes cleanly today. Left
the implementation as filename-only per the brief's own reasoning — a content
or identifier scan would false-positive on `platform/observability`'s
`statusRecorder` type and `records` field (both contain "order") — and did
not "improve" it. This does mean the guard would miss domain logic embedded
inside a technically-named file's body; that's a real gap, but the
false-positive cost of closing it (breaking legitimate infrastructure code
whenever it uses ordinary English words) is worse than the miss.

### B3: gateway added to the service list
`services = []string{"gateway", "identity", "catalog", "orders"}`.

### B4: mutation tests — each guard observed to fail

**Guard 1 (cross-service import).** Added
`import _ "github.com/loloDawit/go-admin/services/identity/internal/platformcheck"`
to `services/catalog/internal/config/config.go`, ran
`go test ./test/arch/ -run TestNoServiceImportsAnotherService -v`:
```
=== RUN   TestNoServiceImportsAnotherService/catalog
    imports_test.go:53: services/catalog/internal/config/config.go imports github.com/loloDawit/go-admin/services/identity/internal/platformcheck.
        	Services communicate over HTTP through published contracts, never by importing each other. Remove this import.
--- FAIL: TestNoServiceImportsAnotherService (0.01s)
```
Reverted.

**Guard 2 (legacy import).** Added
`_ "github.com/loloDawit/go-admin/internal/httpx"` to
`services/gateway/internal/httperr/httperr.go`. First confirmed `go build
./services/gateway/...` **succeeds** — the root's `internal/httpx` is
importable module-wide and a wrong import path here would compile silently,
exactly the failure mode B1 warns about. Then ran `go test ./test/arch/ -run
TestNoNewCodeImportsLegacyPackages -v`:
```
=== RUN   TestNoNewCodeImportsLegacyPackages
    imports_test.go:68: services/gateway/internal/httperr/httperr.go imports legacy package github.com/loloDawit/go-admin/internal/httpx.
        	New code must not depend on the legacy application at the repo root; use the corresponding platform/ or services/*/internal package instead.
--- FAIL: TestNoNewCodeImportsLegacyPackages (0.01s)
```
Reverted.

**Guard 3 (domain concept in platform/).** Created an empty
`platform/httpx/order_helpers.go`, ran `go test ./test/arch/ -run
TestPlatformHoldsNoDomainConcepts -v`:
```
=== RUN   TestPlatformHoldsNoDomainConcepts
    imports_test.go:99: platform/httpx/order_helpers.go looks like a domain concept.
        	platform/ is technical infrastructure only; move this file under services/*/internal instead.
--- FAIL: TestPlatformHoldsNoDomainConcepts (0.00s)
```
Deleted the file; `git status` clean afterward.

### Deviation from the brief: never-fails smell fixed
The brief's `forEachImport` silently returns (no failure) when its target
directory doesn't exist, which means renaming or moving `services/` or
`platform/` would make every guard pass vacuously — exactly the class of test
this task's own brief warns against ("a guard nobody has watched fail is not
yet a guard"). Changed `forEachImport` to `t.Fatalf` when the directory is
missing, and to also `t.Fatalf` if it scans zero `.go` files, so a structural
rename fails loudly instead of passing by accident.

### Comment accuracy
Did not carry forward the brief's comment "Legacy packages at the repo root
are deleted in M2-M4" verbatim: the milestone briefs in this same
`.superpowers/sdd/2026-09-11-m1-platform-skeleton/` directory disagree with
each other (`task-8-brief.md` says M2-M4, `task-12-brief.md` says M2-M5), so
stating a specific range would be an unverified — and possibly false — claim.
The comment in `imports_test.go` states the constraint itself ("nothing new
may depend on them") without committing to a schedule neither I nor the
existing docs agree on.

## Test summary
`gofmt -l services platform test` (empty), `go vet ./services/...
./platform/... ./test/arch/...` (clean), `go build ./...` (clean), `go test
./services/... ./platform/... ./test/arch/ -count=1` — all packages pass,
including the new `TestSlowUpstreamTripsTheDeadlineAndReturnsGatewayTimeout`
and the three `test/arch` guards.

## Files
- `services/gateway/internal/routing/routing.go` — timeout→504 branch
- `services/gateway/internal/routing/routing_test.go` — A1 test
- `services/gateway/internal/httperr/httperr.go` — `WriteGatewayTimeout`
- `test/arch/imports_test.go` — new

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>
