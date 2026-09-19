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
	"github.com/loloDawit/go-admin/services/orders/internal/authz"
	"github.com/loloDawit/go-admin/services/orders/internal/config"
	"github.com/loloDawit/go-admin/services/orders/internal/customer"
	"github.com/loloDawit/go-admin/services/orders/internal/order"
	"github.com/loloDawit/go-admin/services/orders/internal/permission"
	"github.com/loloDawit/go-admin/services/orders/internal/reporting"
)

// RequestLogger must be registered before Recoverer: it logs only after next.ServeHTTP returns, which a panic would unwind past.
// Routes live here, not in main, so router_test.go can pin the spec §6 startup contract without a database.
func newRouter(
	logger *slog.Logger,
	ready *readiness.Handler,
	customerHandler *customer.Handler,
	orderHandler *order.Handler,
	reportingHandler *reporting.Handler,
	principalKey []byte,
	writeErr func(context.Context, http.ResponseWriter, error),
) *chi.Mux {
	r := chi.NewRouter()
	r.Use(requestid.Middleware)
	r.Use(observability.HTTPMiddleware(config.ServiceName))
	r.Use(observability.RouteTagger())
	r.Use(observability.RequestLogger(logger))
	r.Use(chimiddleware.Recoverer)

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	r.Get("/readyz", ready.Ready)

	r.Group(func(r chi.Router) {
		r.Use(principal.Middleware(principalKey, func(w http.ResponseWriter, r *http.Request, err error) {
			writeErr(r.Context(), w, err)
		}))

		r.With(authz.Require(permission.EditOrders, writeErr)).Post("/api/v1/customers", customerHandler.Create)
		// A single route serves both List and ByEmail: the email query
		// param picks the lookup, matching customer/handler.go's ByEmail doc.
		r.With(authz.Require(permission.ViewOrders, writeErr)).Get("/api/v1/customers", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("email") != "" {
				customerHandler.ByEmail(w, r)
				return
			}
			customerHandler.List(w, r)
		})
		r.With(authz.Require(permission.ViewOrders, writeErr)).Get("/api/v1/customers/{id}", customerHandler.Get)
		r.With(authz.Require(permission.ViewOrders, writeErr)).Get("/api/v1/customers/{id}/lifetime-value", customerHandler.LifetimeValue)
		r.With(authz.Require(permission.ViewOrders, writeErr)).Get("/api/v1/customers/{id}/orders", customerHandler.OrderHistory)

		r.With(authz.Require(permission.EditOrders, writeErr)).Post("/api/v1/orders", orderHandler.Create)
		r.With(authz.Require(permission.ViewOrders, writeErr)).Get("/api/v1/orders", orderHandler.List)
		r.With(authz.Require(permission.ViewOrders, writeErr)).Get("/api/v1/orders/{id}", orderHandler.Get)
		r.With(authz.Require(permission.ViewOrders, writeErr)).Get("/api/v1/orders/{id}/events", orderHandler.Events)
		r.With(authz.Require(permission.EditOrders, writeErr)).Post("/api/v1/orders/{id}/status", orderHandler.SetStatus)
		r.With(authz.Require(permission.EditOrders, writeErr)).Post("/api/v1/orders/{id}/cancel", orderHandler.Cancel)
		r.With(authz.Require(permission.EditOrders, writeErr)).Post("/api/v1/orders/{id}/refund", orderHandler.Refund)

		r.With(authz.Require(permission.ViewOrders, writeErr)).Get("/api/v1/reports/dashboard", reportingHandler.Dashboard)
	})

	return r
}
