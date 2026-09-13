package main

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/loloDawit/go-admin/platform/observability"
	"github.com/loloDawit/go-admin/platform/readiness"
	"github.com/loloDawit/go-admin/platform/requestid"
	"github.com/loloDawit/go-admin/services/catalog/internal/platformcheck"
)

// RequestLogger must be registered before Recoverer (so Recoverer sits closer
// to the handler): RequestLogger logs only after next.ServeHTTP returns, and
// an unrecovered panic would unwind past that statement, never running it.
//
// Routes are registered here, not in main, so the spec §6 startup contract
// (liveness never touches the database; readiness reports failure safely) has
// a router_test.go pin that runs without a database.
func newRouter(logger *slog.Logger, handler *platformcheck.Handler, ready *readiness.Handler) *chi.Mux {
	r := chi.NewRouter()
	r.Use(requestid.Middleware)
	r.Use(observability.RequestLogger(logger))
	r.Use(chimiddleware.Recoverer)

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	r.Get("/readyz", ready.Ready)
	r.Get("/_platform", handler.Platform)

	return r
}
