package main

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/loloDawit/go-admin/platform/observability"
	"github.com/loloDawit/go-admin/platform/requestid"
	"github.com/loloDawit/go-admin/services/gateway/internal/auth"
)

// RequestLogger must be registered before Recoverer: it logs only after next.ServeHTTP returns, which a panic would unwind past.
func newRouter(logger *slog.Logger, upstreams http.Handler, validator *auth.Validator) *chi.Mux {
	r := chi.NewRouter()
	r.Use(requestid.Middleware)
	r.Use(observability.HTTPMiddleware(serviceName))
	r.Use(observability.RouteTagger())
	r.Use(observability.RequestLogger(logger))
	r.Use(chimiddleware.Recoverer)
	r.Use(auth.Middleware(validator))

	r.Mount("/", upstreams)

	return r
}
