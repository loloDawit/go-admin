package main

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/loloDawit/go-admin/platform/observability"
	"github.com/loloDawit/go-admin/platform/requestid"
)

// RequestLogger must be registered before Recoverer (so Recoverer sits closer
// to the handler): RequestLogger logs only after next.ServeHTTP returns, and
// an unrecovered panic would unwind past that statement, never running it.
func newRouter(logger *slog.Logger, upstreams http.Handler) *chi.Mux {
	r := chi.NewRouter()
	r.Use(requestid.Middleware)
	r.Use(observability.RequestLogger(logger))
	r.Use(chimiddleware.Recoverer)

	r.Mount("/", upstreams)

	return r
}
