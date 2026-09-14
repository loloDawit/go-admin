// Package permission is Identity's canonical permission vocabulary, published
// in openapi/identity.yaml. In M3 and M4, Catalog and Orders generate typed
// constants from that specification instead of hand-writing permission
// strings.
package permission

import (
	"net/http"

	"github.com/loloDawit/go-admin/platform/httpx"
)

// Permission is one entry in the vocabulary. Every value here is also seeded
// as a row in services/identity/migrations/000002_identity.up.sql and listed
// in the Permission schema's enum in openapi/identity.yaml; the three must
// name the same set.
type Permission string

const (
	ViewStaff    Permission = "view_staff"
	EditStaff    Permission = "edit_staff"
	ViewRoles    Permission = "view_roles"
	EditRoles    Permission = "edit_roles"
	ViewProducts Permission = "view_products"
	EditProducts Permission = "edit_products"
	ViewOrders   Permission = "view_orders"
	EditOrders   Permission = "edit_orders"
)

// All returns the vocabulary as strings, in declaration order. It is compared
// against the published OpenAPI enum in permission_test.go, so the list
// exists exactly once in Go.
func All() []string {
	return []string{
		string(ViewStaff),
		string(EditStaff),
		string(ViewRoles),
		string(EditRoles),
		string(ViewProducts),
		string(EditProducts),
		string(ViewOrders),
		string(EditOrders),
	}
}

// listResponse is GET /api/v1/permissions's body.
type listResponse struct {
	Permissions []string `json:"permissions"`
}

// Handler serves this package's one route. Spec §8 routes it behind
// view_roles; wiring that authorization is Task 10's, not this package's.
type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

// List answers with the full vocabulary, so a permission-matrix UI reads it
// rather than hardcoding the eight strings.
func (h *Handler) List(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, listResponse{Permissions: All()})
}
