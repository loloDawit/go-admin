package controllers_test

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/loloDawit/go-admin/internal/testutil"
	"github.com/loloDawit/go-admin/models"
)

// permissions[] arrives as JSON numbers, not strings. The old handler did
// permissionId.(string), which panics on a numeric id.
func TestCreateRoleAcceptsNumericPermissionIds(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	testutil.SeedUser(t, db, "admin@example.com", "s3cret-password", "admin")
	testutil.GrantPermission(t, db, "admin", "edit_roles")
	cookie := testutil.Login(t, app, "admin@example.com", "s3cret-password")

	var perm models.Permission
	if err := db.Where(models.Permission{Name: "view_orders"}).
		FirstOrCreate(&perm, models.Permission{Name: "view_orders"}).Error; err != nil {
		t.Fatalf("seed permission: %v", err)
	}

	body := `{"name":"support","permissions":[` + strconv.Itoa(int(perm.Id)) + `]}`
	req := testutil.NewRequest(http.MethodPost, "/api/v1/roles", testutil.JSON(body), cookie)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("want 201, got %d", resp.StatusCode)
	}
}

func TestCreateRoleRejectsEmptyName(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	testutil.SeedUser(t, db, "admin2@example.com", "s3cret-password", "admin")
	testutil.GrantPermission(t, db, "admin", "edit_roles")
	cookie := testutil.Login(t, app, "admin2@example.com", "s3cret-password")

	req := testutil.NewRequest(http.MethodPost, "/api/v1/roles", testutil.JSON(`{"name":"","permissions":[]}`), cookie)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", resp.StatusCode)
	}
}
