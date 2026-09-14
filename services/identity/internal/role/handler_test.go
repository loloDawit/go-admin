package role_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/loloDawit/go-admin/services/identity/internal/permission"
	"github.com/loloDawit/go-admin/services/identity/internal/role"
)

func withChiURLParam(req *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func newTestHandler() (*role.Handler, *fakeRepository) {
	svc, repo := newTestService()
	return role.NewHandler(svc, 1<<20, func(context.Context, http.ResponseWriter, error) {}), repo
}

func TestCreateHandlerReturns201WithThePermissionsGiven(t *testing.T) {
	h, _ := newTestHandler()

	body := `{"name":"manager","permissions":["` + string(permission.ViewStaff) + `"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/roles", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status: want 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp role.RoleResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Name != "manager" || len(resp.Permissions) != 1 {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

// TestUpdateHandlerWithOmittedPermissionsFieldLeavesThemUnchanged pins the partial-update DTO contract: an omitted key must not clear the permissions.
func TestUpdateHandlerWithOmittedPermissionsFieldLeavesThemUnchanged(t *testing.T) {
	svc, _ := newTestService()
	h := role.NewHandler(svc, 1<<20, func(context.Context, http.ResponseWriter, error) {})

	created, err := svc.Create(context.Background(), role.CreateRole{Name: "manager", Permissions: []string{string(permission.ViewStaff)}})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	id := strconv.FormatInt(created.ID, 10)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/roles/"+id, strings.NewReader(`{"name":"renamed"}`))
	req = withChiURLParam(req, "id", id)
	rec := httptest.NewRecorder()
	h.Update(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp role.RoleResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Permissions) != 1 || resp.Permissions[0] != string(permission.ViewStaff) {
		t.Fatalf("permissions changed by an omitted field: got %v", resp.Permissions)
	}
}

func TestDeleteHandlerReturns204(t *testing.T) {
	svc, _ := newTestService()
	h := role.NewHandler(svc, 1<<20, func(context.Context, http.ResponseWriter, error) {})

	created, err := svc.Create(context.Background(), role.CreateRole{Name: "manager"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	id := strconv.FormatInt(created.ID, 10)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/roles/"+id, nil)
	req = withChiURLParam(req, "id", id)
	rec := httptest.NewRecorder()
	h.Delete(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status: want 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestListHandlerReturnsEveryRole(t *testing.T) {
	svc, _ := newTestService()
	h := role.NewHandler(svc, 1<<20, func(context.Context, http.ResponseWriter, error) {})
	if _, err := svc.Create(context.Background(), role.CreateRole{Name: "manager"}); err != nil {
		t.Fatalf("create: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/roles", nil)
	rec := httptest.NewRecorder()
	h.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp role.ListRoleResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Roles) != 1 {
		t.Fatalf("roles: want 1, got %d", len(resp.Roles))
	}
}
