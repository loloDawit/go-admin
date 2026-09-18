package main

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/loloDawit/go-admin/platform/observability"
	"github.com/loloDawit/go-admin/platform/requestid"
	"github.com/loloDawit/go-admin/services/gateway/internal/auth"
	"github.com/loloDawit/go-admin/services/gateway/internal/faults"
	"github.com/loloDawit/go-admin/services/gateway/internal/ratelimit"
)

const loginPath = "/api/v1/login"

// RequestLogger must be registered before Recoverer: it logs only after next.ServeHTTP returns, which a panic would unwind past.
func newRouter(logger *slog.Logger, upstreams http.Handler, validator *auth.Validator, limits ratelimit.Config, faultCfg faults.Config) *chi.Mux {
	r := chi.NewRouter()
	r.Use(requestid.Middleware)
	r.Use(observability.HTTPMiddleware(serviceName))
	r.Use(observability.RouteTagger())
	r.Use(observability.RequestLogger(logger))
	r.Use(chimiddleware.Recoverer)
	r.Use(auth.Middleware(validator))
	// Before the limiter, so a load test sees the injected fault rather than a
	// refusal caused by requests piling up behind it.
	r.Use(faults.Middleware(faultCfg))
	r.Use(ratelimit.Middleware(limits))

	// Login is limited per IP and separately: it is the one unauthenticated
	// path, so there is no principal to key on, and it is the one worth
	// brute-forcing.
	r.With(ratelimit.LoginMiddleware(limits)).Handle(loginPath, upstreams)
	r.Mount("/", upstreams)

	return r
}
