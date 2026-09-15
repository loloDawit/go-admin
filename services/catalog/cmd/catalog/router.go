package main

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/loloDawit/go-admin/platform/observability"
	"github.com/loloDawit/go-admin/platform/principal"
	"github.com/loloDawit/go-admin/platform/readiness"
	"github.com/loloDawit/go-admin/platform/requestid"
	"github.com/loloDawit/go-admin/services/catalog/internal/authz"
	"github.com/loloDawit/go-admin/services/catalog/internal/image"
	"github.com/loloDawit/go-admin/services/catalog/internal/permission"
	"github.com/loloDawit/go-admin/services/catalog/internal/product"
)

// RequestLogger must be registered before Recoverer: it logs only after next.ServeHTTP returns, which a panic would unwind past.
// Routes live here, not in main, so router_test.go can pin the spec §6 startup contract without a database.
func newRouter(
	logger *slog.Logger,
	ready *readiness.Handler,
	productHandler *product.Handler,
	imageHandler *image.Handler,
	principalKey []byte,
	writeErr func(context.Context, http.ResponseWriter, error),
) *chi.Mux {
	r := chi.NewRouter()
	r.Use(requestid.Middleware)
	r.Use(observability.RequestLogger(logger))
	r.Use(chimiddleware.Recoverer)

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	r.Get("/readyz", ready.Ready)

	// Gateway-network-only, no principal: identity's /internal/sessions/validate
	// is the same shape. The gateway never proxies /internal/*.
	r.Post("/internal/products/resolve", productHandler.Resolve)

	r.Group(func(r chi.Router) {
		r.Use(principal.Middleware(principalKey, func(w http.ResponseWriter, r *http.Request, err error) {
			writeErr(r.Context(), w, err)
		}))

		r.With(authz.Require(permission.ViewProducts, writeErr)).Get("/api/v1/products", productHandler.List)
		r.With(authz.Require(permission.EditProducts, writeErr)).Post("/api/v1/products", productHandler.Create)
		r.With(authz.Require(permission.ViewProducts, writeErr)).Get("/api/v1/products/search", productHandler.Search)
		r.With(authz.Require(permission.ViewProducts, writeErr)).Get("/api/v1/products/{id}", productHandler.Get)
		r.With(authz.Require(permission.EditProducts, writeErr)).Patch("/api/v1/products/{id}", productHandler.Update)
		r.With(authz.Require(permission.EditProducts, writeErr)).Post("/api/v1/products/{id}/activate", productHandler.Activate)
		r.With(authz.Require(permission.EditProducts, writeErr)).Post("/api/v1/products/{id}/archive", productHandler.Archive)

		r.With(authz.Require(permission.ViewProducts, writeErr)).Get("/api/v1/products/{id}/images", imageHandler.List)
		r.With(authz.Require(permission.EditProducts, writeErr)).Post("/api/v1/products/{id}/images", imageHandler.Upload)
		r.With(authz.Require(permission.EditProducts, writeErr)).Delete("/api/v1/products/{id}/images/{imageID}", imageHandler.Delete)
	})

	return r
}
