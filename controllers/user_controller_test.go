package controllers_test

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/loloDawit/go-admin/internal/testutil"
	"github.com/loloDawit/go-admin/models"
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

func TestCreateUserRejectsRoleExceedingCallersOwn(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	testutil.SeedUser(t, db, "admin6@example.com", "s3cret-password", "admin")
	testutil.GrantPermission(t, db, "admin", "edit_users")
	adminCookie := testutil.Login(t, app, "admin6@example.com", "s3cret-password")

	ownerRole := testutil.SeedRole(t, db, "owner")
	testutil.GrantPermission(t, db, "owner", "edit_roles")

	req := testutil.NewRequest(http.MethodPost, "/api/v1/users",
		testutil.JSON(`{"firstName":"Eve","lastName":"Escalate","email":"eve@example.com","password":"whatever-pw","roleId":`+
			strconv.Itoa(int(ownerRole.Id))+`}`), adminCookie)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("want 403 when assigning a role that exceeds the caller's own permissions, got %d", resp.StatusCode)
	}
}

func TestUpdateUserRejectsRoleExceedingCallersOwn(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	admin := testutil.SeedUser(t, db, "admin7@example.com", "s3cret-password", "admin")
	testutil.GrantPermission(t, db, "admin", "edit_users")
	adminCookie := testutil.Login(t, app, "admin7@example.com", "s3cret-password")

	ownerRole := testutil.SeedRole(t, db, "owner")
	testutil.GrantPermission(t, db, "owner", "edit_roles")

	req := testutil.NewRequest(http.MethodPut, "/api/v1/user/"+strconv.Itoa(admin.Id),
		testutil.JSON(`{"roleId":`+strconv.Itoa(int(ownerRole.Id))+`}`), adminCookie)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("want 403 when self-assigning a role that exceeds the caller's own permissions, got %d", resp.StatusCode)
	}
}

// PUT /user/:id must not accept a nested "role" object: models.User carries
// a bindable Role association, and saving it through GORM inserts
// role_permissions rows directly, bypassing ensureCanAssignRole's roleId check
// entirely.
func TestUpdateUserRejectsNestedRoleAssociationPermissionInjection(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	admin := testutil.SeedUser(t, db, "admin8@example.com", "s3cret-password", "admin8role")
	testutil.GrantPermission(t, db, "admin8role", "edit_users")
	adminCookie := testutil.Login(t, app, "admin8@example.com", "s3cret-password")

	var adminRole models.Role
	if err := db.Where("name = ?", "admin8role").First(&adminRole).Error; err != nil {
		t.Fatalf("find caller role: %v", err)
	}

	var editRoles models.Permission
	if err := db.Where(models.Permission{Name: "edit_roles"}).
		FirstOrCreate(&editRoles, models.Permission{Name: "edit_roles"}).Error; err != nil {
		t.Fatalf("seed edit_roles permission: %v", err)
	}

	body := `{"firstName":"Test","lastName":"User","email":"admin8@example.com","role":{"id":` + strconv.Itoa(int(adminRole.Id)) +
		`,"permissions":[{"id":` + strconv.Itoa(int(editRoles.Id)) + `}]}}`

	req := testutil.NewRequest(http.MethodPut, "/api/v1/user/"+strconv.Itoa(admin.Id),
		testutil.JSON(body), adminCookie)

	if _, err := app.Test(req, -1); err != nil {
		t.Fatalf("request: %v", err)
	}

	var reloaded models.Role
	if err := db.Preload("Permissions").First(&reloaded, adminRole.Id).Error; err != nil {
		t.Fatalf("reload caller role: %v", err)
	}
	for _, p := range reloaded.Permissions {
		if p.Name == "edit_roles" {
			t.Fatal("caller's role gained edit_roles through a nested role association in the request body")
		}
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

func TestUpdateUserRejectsDuplicateEmail(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	testutil.SeedUser(t, db, "existing@example.com", "s3cret-password", "admin")
	testutil.GrantPermission(t, db, "admin", "edit_users")
	target := testutil.SeedUser(t, db, "target@example.com", "s3cret-password", "admin")
	adminCookie := testutil.Login(t, app, "existing@example.com", "s3cret-password")

	req := testutil.NewRequest(http.MethodPut, "/api/v1/user/"+strconv.Itoa(target.Id),
		testutil.JSON(`{"firstName":"Target","lastName":"User","email":"existing@example.com"}`), adminCookie)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("want 409, got %d", resp.StatusCode)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["code"] != "email_taken" {
		t.Errorf("code: want email_taken, got %v", body["code"])
	}
}

// A roleId-only body must not silently blank the other required fields —
// UpdateUser always writes firstName/lastName/email, so an incomplete body
// must be rejected outright rather than clearing the account's email.
func TestUpdateUserRejectsRoleOnlyPayload(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	testutil.SeedUser(t, db, "admin9@example.com", "s3cret-password", "admin")
	testutil.GrantPermission(t, db, "admin", "edit_users")
	adminCookie := testutil.Login(t, app, "admin9@example.com", "s3cret-password")

	target := testutil.SeedUser(t, db, "target9@example.com", "s3cret-password", "staff")
	staffRole := testutil.SeedRole(t, db, "staff")

	req := testutil.NewRequest(http.MethodPut, "/api/v1/user/"+strconv.Itoa(target.Id),
		testutil.JSON(`{"roleId":`+strconv.Itoa(int(staffRole.Id))+`}`), adminCookie)

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
	if body["code"] != "missing_field" {
		t.Errorf("code: want missing_field, got %v", body["code"])
	}

	var reloaded models.User
	if err := db.First(&reloaded, target.Id).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.Email != "target9@example.com" {
		t.Errorf("a rejected request must not have written: email is now %q", reloaded.Email)
	}
}

// UpdateUser must apply the same format check CreateUser does.
func TestUpdateUserRejectsInvalidEmailFormat(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	testutil.SeedUser(t, db, "admin10@example.com", "s3cret-password", "admin")
	testutil.GrantPermission(t, db, "admin", "edit_users")
	adminCookie := testutil.Login(t, app, "admin10@example.com", "s3cret-password")

	target := testutil.SeedUser(t, db, "target10@example.com", "s3cret-password", "staff")

	req := testutil.NewRequest(http.MethodPut, "/api/v1/user/"+strconv.Itoa(target.Id),
		testutil.JSON(`{"firstName":"Target","lastName":"User","email":"not-an-email"}`), adminCookie)

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
	if body["code"] != "email_invalid" {
		t.Errorf("code: want email_invalid, got %v", body["code"])
	}
}

// The subset check in ensureCanAssignRole only guards which role gets
// assigned; without a check on the target user, an admin (who lacks
// edit_roles) could demote the owner — the only holder of edit_roles — to a
// lesser role, since staff's permissions are a subset of admin's own.
func TestUpdateUserRejectsDemotingAMorePrivilegedUser(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	testutil.SeedUser(t, db, "admin11@example.com", "s3cret-password", "admin")
	testutil.GrantPermission(t, db, "admin", "edit_users")
	adminCookie := testutil.Login(t, app, "admin11@example.com", "s3cret-password")

	owner := testutil.SeedUser(t, db, "owner11@example.com", "s3cret-password", "owner")
	testutil.GrantPermission(t, db, "owner", "edit_roles")

	staffRole := testutil.SeedRole(t, db, "staff")

	req := testutil.NewRequest(http.MethodPut, "/api/v1/user/"+strconv.Itoa(owner.Id),
		testutil.JSON(`{"roleId":`+strconv.Itoa(int(staffRole.Id))+`}`), adminCookie)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("want 403 when demoting a user whose role holds permissions the caller lacks, got %d", resp.StatusCode)
	}

	var reloaded models.User
	if err := db.First(&reloaded, owner.Id).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.RoleId != owner.RoleId {
		t.Error("the owner's role must not have changed")
	}
}

// DeleteUser must reject deleting a user whose role holds permissions the
// caller lacks, the same guard UpdateUser applies.
func TestDeleteUserRejectsDeletingAMorePrivilegedUser(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	testutil.SeedUser(t, db, "admin12@example.com", "s3cret-password", "admin")
	testutil.GrantPermission(t, db, "admin", "edit_users")
	adminCookie := testutil.Login(t, app, "admin12@example.com", "s3cret-password")

	owner := testutil.SeedUser(t, db, "owner12@example.com", "s3cret-password", "owner")
	testutil.GrantPermission(t, db, "owner", "edit_roles")

	req := testutil.NewRequest(http.MethodDelete, "/api/v1/user/"+strconv.Itoa(owner.Id), nil, adminCookie)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("want 403 when deleting a user whose role holds permissions the caller lacks, got %d", resp.StatusCode)
	}

	var reloaded models.User
	if err := db.First(&reloaded, owner.Id).Error; err != nil {
		t.Fatalf("the owner must not have been deleted: %v", err)
	}
}

// The target-user check must not block a caller from acting on a user whose
// role holds no more than the caller's own.
func TestUpdateUserAllowsModifyingALesserPrivilegedUser(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	testutil.SeedUser(t, db, "admin13@example.com", "s3cret-password", "admin")
	testutil.GrantPermission(t, db, "admin", "edit_users")
	adminCookie := testutil.Login(t, app, "admin13@example.com", "s3cret-password")

	staffUser := testutil.SeedUser(t, db, "staff13@example.com", "s3cret-password", "staff")

	req := testutil.NewRequest(http.MethodPut, "/api/v1/user/"+strconv.Itoa(staffUser.Id),
		testutil.JSON(`{"firstName":"Renamed","lastName":"Staffer","email":"staff13@example.com"}`), adminCookie)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("an admin must still be able to modify a staff user, got %d", resp.StatusCode)
	}
}

// A caller acting on their own row must not be blocked by the target-user
// check, even though it loads the target the same way as any other id.
func TestUpdateUserAllowsSelfModification(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	admin := testutil.SeedUser(t, db, "admin14@example.com", "s3cret-password", "admin")
	testutil.GrantPermission(t, db, "admin", "edit_users")
	adminCookie := testutil.Login(t, app, "admin14@example.com", "s3cret-password")

	req := testutil.NewRequest(http.MethodPut, "/api/v1/user/"+strconv.Itoa(admin.Id),
		testutil.JSON(`{"firstName":"Renamed","lastName":"Admin","email":"admin14@example.com"}`), adminCookie)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("a caller must be able to act on their own row, got %d", resp.StatusCode)
	}
}
