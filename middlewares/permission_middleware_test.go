package middlewares_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/loloDawit/go-admin/internal/testutil"
)

// The headline regression test: before this task, a user with NO permissions
// could create products, because no product route consulted IsAuthorized.
func TestWriteRoutesRejectUserWithoutEditPermission(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	testutil.SeedUser(t, db, "viewer@example.com", "s3cret-password", "viewer")
	testutil.GrantPermission(t, db, "viewer", "view_products")
	cookie := testutil.Login(t, app, "viewer@example.com", "s3cret-password")

	writes := []struct {
		method, path, body string
	}{
		{http.MethodPost, "/api/v1/products", `{"title":"Pwned","price":1}`},
		{http.MethodPut, "/api/v1/product/1", `{"title":"Pwned"}`},
		{http.MethodDelete, "/api/v1/product/1", ``},
		{http.MethodPost, "/api/v1/orders", `{"email":"x@example.com"}`},
		{http.MethodPost, "/api/v1/roles", `{"name":"superuser","permissions":[1]}`},
		{http.MethodPost, "/api/v1/permissions", `{"name":"edit_everything"}`},
	}

	for _, w := range writes {
		req := testutil.NewRequest(w.method, w.path, strings.NewReader(w.body), cookie)
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("%s %s: %v", w.method, w.path, err)
		}
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("%s %s: want 403, got %d (unauthorized write is possible)",
				w.method, w.path, resp.StatusCode)
		}
	}
}

func TestReadRouteAllowsViewPermission(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	testutil.SeedUser(t, db, "viewer@example.com", "s3cret-password", "viewer")
	testutil.GrantPermission(t, db, "viewer", "view_products")
	cookie := testutil.Login(t, app, "viewer@example.com", "s3cret-password")

	req := testutil.NewRequest(http.MethodGet, "/api/v1/products", nil, cookie)
	resp, _ := app.Test(req, -1)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("view_products must permit GET /products, got %d", resp.StatusCode)
	}
}

func TestEditPermissionImpliesRead(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	testutil.SeedUser(t, db, "editor@example.com", "s3cret-password", "editor")
	testutil.GrantPermission(t, db, "editor", "edit_products")
	cookie := testutil.Login(t, app, "editor@example.com", "s3cret-password")

	req := testutil.NewRequest(http.MethodGet, "/api/v1/products", nil, cookie)
	resp, _ := app.Test(req, -1)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("edit_products must imply read access, got %d", resp.StatusCode)
	}
}

func TestUnauthenticatedRequestIsRejected(t *testing.T) {
	testutil.NewDB(t)
	app := testutil.NewApp(t)

	req := testutil.NewRequest(http.MethodGet, "/api/v1/products", nil, nil)
	resp, _ := app.Test(req, -1)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("want 401 without a cookie, got %d", resp.StatusCode)
	}
}
