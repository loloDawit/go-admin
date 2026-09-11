package routes_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/loloDawit/go-admin/internal/testutil"
)

// publicRoutes lists every endpoint that intentionally requires no resource
// permission. Adding an entry here must be a deliberate edit, since per-route
// middleware attachment (routes.go) makes an omitted middleware argument a
// silent, compiling, test-passing way to ship an unguarded endpoint.
var publicRoutes = map[string]bool{
	"POST /api/v1/login":        true,
	"GET /api/v1/user":          true,
	"HEAD /api/v1/user":         true, // Fiber registers HEAD alongside every GET
	"POST /api/v1/logout":       true,
	"PUT /api/v1/user/info":     true,
	"PUT /api/v1/user/password": true,
}

// TestEveryResourceRouteIsPermissionGated walks the live route table and
// asserts that any route not named in publicRoutes rejects a signed-in user
// holding zero permissions. A route added without a RequirePermission
// argument fails this test instead of shipping silently unguarded.
func TestEveryResourceRouteIsPermissionGated(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	testutil.SeedUser(t, db, "noperm@example.com", "s3cret-password", "noperm")
	cookie := testutil.Login(t, app, "noperm@example.com", "s3cret-password")

	tested := map[string]bool{}

	for _, group := range app.Stack() {
		for _, route := range group {
			// The Group("", IsAuthenticated) registration surfaces as a
			// Use pseudo-route at the bare group prefix on every method.
			if route.Path == "/api/v1" {
				continue
			}
			// Static file serving; checked separately below.
			if strings.HasPrefix(route.Path, "/api/v1/uploads") {
				continue
			}

			key := route.Method + " " + route.Path
			if tested[key] || publicRoutes[key] {
				continue
			}
			tested[key] = true

			target := route.Path
			for _, param := range route.Params {
				target = strings.Replace(target, ":"+param, "1", 1)
			}

			req := testutil.NewRequest(route.Method, target, nil, cookie)
			resp, err := app.Test(req, -1)
			if err != nil {
				t.Fatalf("%s %s: %v", route.Method, target, err)
			}
			if resp.StatusCode != http.StatusForbidden {
				t.Errorf("%s %s: want 403 for a user holding no permissions, got %d",
					route.Method, target, resp.StatusCode)
			}
		}
	}
}

func TestUploadsRequireAuthentication(t *testing.T) {
	testutil.NewDB(t)
	app := testutil.NewApp(t)

	req := testutil.NewRequest(http.MethodGet, "/api/v1/uploads/x.png", nil, nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("want 401 for an unauthenticated upload fetch, got %d", resp.StatusCode)
	}
}
