// Package permission is Identity's canonical permission vocabulary, published in openapi/identity.yaml.
package permission

import (
	"net/http"

	"github.com/loloDawit/go-admin/platform/httpx"
)

// Permission: every value here must also be seeded in 000002_identity.up.sql and listed in identity.yaml's Permission enum.
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

// All is compared against the published OpenAPI enum in permission_test.go, so the list exists exactly once in Go.
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

// Handler serves this package's one route; spec §8 routes it behind view_roles.
type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

// List answers with the full vocabulary, so a permission-matrix UI reads it
// rather than hardcoding the eight strings.
func (h *Handler) List(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, listResponse{Permissions: All()})
}
