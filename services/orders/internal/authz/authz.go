// Package authz enforces route-level permission checks; onErr, not this package, decides the response.
package authz

import (
	"context"
	"net/http"
	"slices"

	"github.com/loloDawit/go-admin/platform/principal"
	"github.com/loloDawit/go-admin/services/orders/internal/errs"
	"github.com/loloDawit/go-admin/services/orders/internal/permission"
)

// Require fails closed: no principal is errs.ErrUnauthenticated, one missing p is errs.ErrForbidden.
func Require(p permission.Permission, onErr func(context.Context, http.ResponseWriter, error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			caller, ok := principal.FromContext(r.Context())
			if !ok {
				onErr(r.Context(), w, errs.ErrUnauthenticated)
				return
			}
			if !slices.Contains(caller.Permissions, string(p)) {
				onErr(r.Context(), w, errs.ErrForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
