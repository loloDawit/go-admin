package controllers_test

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/loloDawit/go-admin/internal/testutil"
	"github.com/loloDawit/go-admin/models"
)

// permissions[] arrives as JSON numbers, not strings.
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

	var created models.Role
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(created.Permissions) != 1 || created.Permissions[0].Id != perm.Id {
		t.Errorf("want permission %d attached, got %+v", perm.Id, created.Permissions)
	}
}

func TestCreateRoleAcceptsDuplicatePermissionIds(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	testutil.SeedUser(t, db, "admin3@example.com", "s3cret-password", "admin")
	testutil.GrantPermission(t, db, "admin", "edit_roles")
	cookie := testutil.Login(t, app, "admin3@example.com", "s3cret-password")

	var perm models.Permission
	if err := db.Where(models.Permission{Name: "view_orders"}).
		FirstOrCreate(&perm, models.Permission{Name: "view_orders"}).Error; err != nil {
		t.Fatalf("seed permission: %v", err)
	}

	body := `{"name":"dup-support","permissions":[` + strconv.Itoa(int(perm.Id)) + `,` + strconv.Itoa(int(perm.Id)) + `]}`
	req := testutil.NewRequest(http.MethodPost, "/api/v1/roles", testutil.JSON(body), cookie)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("want 201 for an all-valid id list with a duplicate, got %d", resp.StatusCode)
	}
}

func TestDeleteRoleWithGrantedPermissionReturns409(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	testutil.SeedUser(t, db, "admin4@example.com", "s3cret-password", "admin")
	testutil.GrantPermission(t, db, "admin", "edit_roles")
	cookie := testutil.Login(t, app, "admin4@example.com", "s3cret-password")

	victimRole := testutil.SeedRole(t, db, "victim-role")
	testutil.GrantPermission(t, db, "victim-role", "view_products")

	req := testutil.NewRequest(http.MethodDelete, "/api/v1/role/"+strconv.Itoa(int(victimRole.Id)), nil, cookie)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("want 409 for a role with granted permissions, got %d", resp.StatusCode)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["code"] != "resource_in_use" {
		t.Errorf("code: want resource_in_use, got %v", body["code"])
	}
}

func TestDeleteRoleHeldByUserReturns409(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	testutil.SeedUser(t, db, "admin5@example.com", "s3cret-password", "admin")
	testutil.GrantPermission(t, db, "admin", "edit_roles")
	cookie := testutil.Login(t, app, "admin5@example.com", "s3cret-password")

	// SeedUser attaches its own fresh role, which is what we're deleting.
	holder := testutil.SeedUser(t, db, "holder@example.com", "s3cret-password", "held-role")

	req := testutil.NewRequest(http.MethodDelete, "/api/v1/role/"+strconv.Itoa(int(holder.RoleId)), nil, cookie)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("want 409 for a role still held by a user, got %d", resp.StatusCode)
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
