package controllers_test

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/loloDawit/go-admin/internal/testutil"
)

func TestCreateUserWithAdminSuppliedPasswordCanLogIn(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	testutil.SeedUser(t, db, "admin@example.com", "s3cret-password", "admin")
	testutil.GrantPermission(t, db, "admin", "edit_users")
	adminCookie := testutil.Login(t, app, "admin@example.com", "s3cret-password")

	staffRole := testutil.SeedRole(t, db, "staff")

	req := testutil.NewRequest(http.MethodPost, "/api/v1/users",
		testutil.JSON(`{"firstName":"Grace","lastName":"Hopper","email":"grace@example.com","password":"hopper-init-pw","roleId":`+
			strconv.Itoa(int(staffRole.Id))+`}`), adminCookie)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("want 201, got %d", resp.StatusCode)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if _, present := body["password"]; present {
		t.Error("response body must not carry the password field")
	}

	testutil.Login(t, app, "grace@example.com", "hopper-init-pw")
}

func TestCreateUserRejectsMissingRole(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	testutil.SeedUser(t, db, "admin2@example.com", "s3cret-password", "admin")
	testutil.GrantPermission(t, db, "admin", "edit_users")
	adminCookie := testutil.Login(t, app, "admin2@example.com", "s3cret-password")

	req := testutil.NewRequest(http.MethodPost, "/api/v1/users",
		testutil.JSON(`{"firstName":"Ada","lastName":"Lovelace","email":"ada2@example.com","password":"whatever-pw","roleId":0}`),
		adminCookie)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", resp.StatusCode)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["code"] != "role_required" {
		t.Errorf("code: want role_required, got %v", body["code"])
	}
}

func TestCreateUserRejectsEmptyPassword(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	testutil.SeedUser(t, db, "admin3@example.com", "s3cret-password", "admin")
	testutil.GrantPermission(t, db, "admin", "edit_users")
	adminCookie := testutil.Login(t, app, "admin3@example.com", "s3cret-password")

	staffRole := testutil.SeedRole(t, db, "staff")

	req := testutil.NewRequest(http.MethodPost, "/api/v1/users",
		testutil.JSON(`{"firstName":"Ada","lastName":"Lovelace","email":"ada3@example.com","password":"","roleId":`+
			strconv.Itoa(int(staffRole.Id))+`}`), adminCookie)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", resp.StatusCode)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["code"] != "password_empty" {
		t.Errorf("code: want password_empty, got %v", body["code"])
	}
}

func TestCreateUserRejectsNonexistentRole(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	testutil.SeedUser(t, db, "admin5@example.com", "s3cret-password", "admin")
	testutil.GrantPermission(t, db, "admin", "edit_users")
	adminCookie := testutil.Login(t, app, "admin5@example.com", "s3cret-password")

	req := testutil.NewRequest(http.MethodPost, "/api/v1/users",
		testutil.JSON(`{"firstName":"Ada","lastName":"Lovelace","email":"ada5@example.com","password":"whatever-pw","roleId":999}`),
		adminCookie)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("want 404, got %d", resp.StatusCode)
	}
}

func TestCreateUserRejectsDuplicateEmail(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	testutil.SeedUser(t, db, "admin4@example.com", "s3cret-password", "admin")
	testutil.GrantPermission(t, db, "admin", "edit_users")
	adminCookie := testutil.Login(t, app, "admin4@example.com", "s3cret-password")

	staffRole := testutil.SeedRole(t, db, "staff")

	req := testutil.NewRequest(http.MethodPost, "/api/v1/users",
		testutil.JSON(`{"firstName":"Dup","lastName":"User","email":"admin4@example.com","password":"whatever-pw","roleId":`+
			strconv.Itoa(int(staffRole.Id))+`}`), adminCookie)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("want 409, got %d", resp.StatusCode)
	}
}
