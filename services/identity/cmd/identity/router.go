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
	"github.com/loloDawit/go-admin/services/identity/internal/authz"
	"github.com/loloDawit/go-admin/services/identity/internal/permission"
	"github.com/loloDawit/go-admin/services/identity/internal/role"
	"github.com/loloDawit/go-admin/services/identity/internal/session"
	"github.com/loloDawit/go-admin/services/identity/internal/staff"
)

// RequestLogger must be registered before Recoverer: it logs only after next.ServeHTTP returns, which a panic would unwind past.
// Routes live here, not in main, so router_test.go can pin the spec §6 startup contract without a database.
func newRouter(
	logger *slog.Logger,
	ready *readiness.Handler,
	sessionHandler *session.Handler,
	staffHandler *staff.Handler,
	roleHandler *role.Handler,
	permissionHandler *permission.Handler,
	principalKey []byte,
	writeErr func(context.Context, http.ResponseWriter, error),
) *chi.Mux {
	r := chi.NewRouter()
	r.Use(requestid.Middleware)
	r.Use(observability.RequestLogger(logger))
	r.Use(chimiddleware.Recoverer)

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	r.Get("/readyz", ready.Ready)

	r.Post("/api/v1/login", sessionHandler.Login)
	r.Post("/internal/sessions/validate", sessionHandler.Validate)

	// RequirePasswordChanged sits ahead of every route below, authz-guarded ones included.
	r.Group(func(r chi.Router) {
		r.Use(principal.Middleware(principalKey, func(w http.ResponseWriter, r *http.Request, err error) {
			writeErr(r.Context(), w, err)
		}))
		r.Use(staff.RequirePasswordChanged(writeErr))

		r.Post("/api/v1/logout", sessionHandler.Logout)
		r.Get("/api/v1/me", sessionHandler.Me)
		r.Post("/api/v1/me/password", staffHandler.ChangePassword)

		r.With(authz.Require(permission.ViewStaff, writeErr)).Get("/api/v1/staff", staffHandler.List)
		r.With(authz.Require(permission.EditStaff, writeErr)).Post("/api/v1/staff", staffHandler.Create)
		r.With(authz.Require(permission.ViewStaff, writeErr)).Get("/api/v1/staff/{id}", staffHandler.Get)
		r.With(authz.Require(permission.EditStaff, writeErr)).Patch("/api/v1/staff/{id}", staffHandler.Update)
		r.With(authz.Require(permission.EditStaff, writeErr)).Post("/api/v1/staff/{id}/deactivate", staffHandler.Deactivate)

		r.With(authz.Require(permission.ViewRoles, writeErr)).Get("/api/v1/roles", roleHandler.List)
		r.With(authz.Require(permission.EditRoles, writeErr)).Post("/api/v1/roles", roleHandler.Create)
		r.With(authz.Require(permission.ViewRoles, writeErr)).Get("/api/v1/roles/{id}", roleHandler.Get)
		r.With(authz.Require(permission.EditRoles, writeErr)).Patch("/api/v1/roles/{id}", roleHandler.Update)
		r.With(authz.Require(permission.EditRoles, writeErr)).Delete("/api/v1/roles/{id}", roleHandler.Delete)
		r.With(authz.Require(permission.ViewRoles, writeErr)).Get("/api/v1/permissions", permissionHandler.List)
	})

	return r
}
