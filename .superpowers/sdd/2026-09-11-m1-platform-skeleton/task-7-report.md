# Task 7 report: Gateway

## Status
Complete.

## Files
- `services/gateway/cmd/gateway/main.go`
- `services/gateway/cmd/gateway/router.go`
- `services/gateway/cmd/gateway/router_test.go`
- `services/gateway/internal/config/config.go`, `config_test.go`
- `services/gateway/internal/routing/routing.go`, `routing_test.go`
- `services/gateway/internal/httperr/httperr.go`
- `services/gateway/Dockerfile`

## Test summary
`go test ./services/... ./platform/... -count=1` — all packages pass, including 5 routing tests, 2 router (full-middleware-stack) tests, and 2 config tests newly added for the gateway; `gofmt -l`, `go vet`, and `go build ./...` all clean.

## Deviations from the brief (both required to satisfy the task's own constraints, not brief bugs I introduced then had to work around)

1. **Path rewrite moved from the route handler into `proxy.Director`.** The brief's handler did `req.URL.Path = "/_platform"` before calling `proxy.ServeHTTP(w, req)`. Since `req.WithContext` is a shallow copy, this mutates the *inbound* request's shared `*url.URL`, so `RequestLogger` (which reads `r.URL.Path` after the handler returns) logs `route=/_platform` for every service instead of `/_platform/{service}`. Fixed by rewriting the path inside `Director`, which `ReverseProxy.ServeHTTP` runs against `req.Clone(ctx)` — a deep copy — leaving the inbound request's URL untouched.

2. **The gateway now forwards its originated request ID to the upstream.** `requestid.Middleware` only sets the response header and the request context, not `r.Header`; nothing in the brief's code copied it onto the outbound request, so the upstream would mint its own unrelated ID. Added a line in the same `Director` wrapper: `out.Header.Set(requestid.Header, requestid.FromContext(out.Context()))`. Without this, "the gateway originates a request ID" doesn't actually correlate gateway and upstream logs.

Both are covered by tests: `TestProxiedResponseHasExactlyOneRequestIDHeader` (routing package) and `TestRouterProxiesThroughTheFullMiddlewareStack` (cmd/gateway, exercises `requestid → RequestLogger → Recoverer → routing` end to end via `observability.NewCaptured`, and asserts the logged `route` attribute and that the upstream saw the same ID as the client). `TestRouterRecoversFromAPanicAndStillLogs` pins the middleware order itself.

I also strengthened the brief's leak test to decode the body and assert `code == "upstream_unavailable"` (deleting the custom `ErrorHandler` still produced a leak-free 502 from `ReverseProxy`'s own default, so the original assertions alone didn't discriminate loss of the envelope).

## Concerns
- Upstream call deadline is enforced (`context.WithTimeout` per request) but a deadline-exceeded call returns 502 via the same `ErrorHandler`/`upstream_unavailable` path, not a distinct 504 — not specified either way by the brief; flagging in case M2 wants to distinguish timeout from connection failure.
- No test pins the `UpstreamTimeout` value actually bounding a slow upstream (a `httptest.Server` that sleeps past the deadline) — behavior relies on `ReverseProxy` honoring the inbound request's context, which it does, but it's unverified by an explicit test.

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>
