package controllers_test

import (
	"net/http"
	"testing"

	"github.com/loloDawit/go-admin/internal/testutil"
	"github.com/loloDawit/go-admin/models"
)

func TestCreateRoleRejectsUnknownPermissionId(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	testutil.SeedUser(t, db, "admin9@example.com", "s3cret-password", "admin")
	testutil.GrantPermission(t, db, "admin", "edit_roles")
	cookie := testutil.Login(t, app, "admin9@example.com", "s3cret-password")

	var before int64
	db.Model(&models.Permission{}).Count(&before)

	req := testutil.NewRequest(http.MethodPost, "/api/v1/roles", testutil.JSON(`{"name":"ghost","permissions":[999999]}`), cookie)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("want 404 for unknown permission id, got %d", resp.StatusCode)
	}

	var after int64
	db.Model(&models.Permission{}).Count(&after)
	if after != before {
		t.Errorf("permissions table changed: before=%d after=%d (phantom row inserted)", before, after)
	}
}
