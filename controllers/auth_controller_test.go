package controllers_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/loloDawit/go-admin/internal/errs"
	"github.com/loloDawit/go-admin/internal/testutil"
	"github.com/loloDawit/go-admin/models"
)

// The regression test for the key-casing defect. Register read
// data["firstName"] while UpdateUserInfo read data["firstname"], and the
// frontend sends camelCase — so renaming yourself returned 200 and silently
// did nothing. Verified against the running app before the fix.
func TestUpdateUserInfoAppliesCamelCaseNames(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	user := testutil.SeedUser(t, db, "ada@example.com", "s3cret-password", "owner")
	cookie := testutil.Login(t, app, "ada@example.com", "s3cret-password")

	req := testutil.NewRequest(http.MethodPut, "/api/v1/user/info",
		testutil.JSON(`{"firstName":"Grace","lastName":"Hopper","email":"grace@example.com"}`), cookie)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}

	var reloaded models.User
	if err := db.First(&reloaded, user.Id).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.FirstName != "Grace" {
		t.Errorf("FirstName: want \"Grace\", got %q — the camelCase key was ignored", reloaded.FirstName)
	}
	if reloaded.LastName != "Hopper" {
		t.Errorf("LastName: want \"Hopper\", got %q", reloaded.LastName)
	}
	if reloaded.Email != "grace@example.com" {
		t.Errorf("Email: want grace@example.com, got %q", reloaded.Email)
	}
}

// Go's encoding/json falls back to case-insensitive field matching when no
// exact tag match exists, so "firstname" still binds to FirstName. That is
// standard library behaviour and is left alone — enforcing strict case would
// need a custom decoder for no real gain.
//
// What actually mattered is fixed and is asserted here: there is now ONE
// struct defining the contract, so two handlers can no longer disagree about
// the same field. Both spellings reach the same place instead of one silently
// no-opping.
func TestUpdateUserInfoAcceptsEitherCaseConsistently(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	user := testutil.SeedUser(t, db, "ada@example.com", "s3cret-password", "owner")
	cookie := testutil.Login(t, app, "ada@example.com", "s3cret-password")

	for _, body := range []string{
		`{"firstName":"Grace","lastName":"Hopper","email":"ada@example.com"}`,
		`{"firstname":"Grace","lastname":"Hopper","email":"ada@example.com"}`,
	} {
		db.Model(&models.User{Id: user.Id}).
			Updates(map[string]any{"first_name": "Ada", "last_name": "Lovelace"})

		req := testutil.NewRequest(http.MethodPut, "/api/v1/user/info", testutil.JSON(body), cookie)
		if _, err := app.Test(req, -1); err != nil {
			t.Fatalf("request: %v", err)
		}

		var reloaded models.User
		db.First(&reloaded, user.Id)
		if reloaded.FirstName != "Grace" {
			t.Errorf("body %s: want Grace, got %q", body, reloaded.FirstName)
		}
	}
}

// The rename must be a real write, not a no-op that reports success. This is
// the behaviour that was broken: GORM's struct-form Updates skips zero values,
// so the empty strings the old handler produced were silently dropped.
func TestUpdateUserInfoCanClearNothingSilently(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	user := testutil.SeedUser(t, db, "ada@example.com", "s3cret-password", "owner")
	cookie := testutil.Login(t, app, "ada@example.com", "s3cret-password")

	// An empty required field must be rejected outright, not silently skipped.
	req := testutil.NewRequest(http.MethodPut, "/api/v1/user/info",
		testutil.JSON(`{"firstName":"","lastName":"","email":"ada@example.com"}`), cookie)
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("an empty name must be a 400, got %d — the old code accepted it and silently no-opped",
			resp.StatusCode)
	}

	var reloaded models.User
	db.First(&reloaded, user.Id)
	if reloaded.FirstName != "Test" {
		t.Errorf("a rejected request must not have written: got %q", reloaded.FirstName)
	}
}

// UpdateUserInfo shares CreateUser's uniqueness constraint on email: a
// self-service duplicate must map to errs.EmailTaken (409), not a bare
// database error.
func TestUpdateUserInfoRejectsDuplicateEmail(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	testutil.SeedUser(t, db, "taken@example.com", "s3cret-password", "owner")
	testutil.SeedUser(t, db, "ada@example.com", "s3cret-password", "owner")
	cookie := testutil.Login(t, app, "ada@example.com", "s3cret-password")

	req := testutil.NewRequest(http.MethodPut, "/api/v1/user/info",
		testutil.JSON(`{"firstName":"Ada","lastName":"Lovelace","email":"taken@example.com"}`), cookie)

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
	if body["code"] != errs.EmailTaken.Code {
		t.Errorf("code: want %q, got %v", errs.EmailTaken.Code, body["code"])
	}
}

// UpdateUserInfo must apply the same email format check as CreateUser.
func TestUpdateUserInfoRejectsInvalidEmailFormat(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	testutil.SeedUser(t, db, "ada@example.com", "s3cret-password", "owner")
	cookie := testutil.Login(t, app, "ada@example.com", "s3cret-password")

	req := testutil.NewRequest(http.MethodPut, "/api/v1/user/info",
		testutil.JSON(`{"firstName":"Ada","lastName":"Lovelace","email":"not-an-email"}`), cookie)

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
	if body["code"] != errs.EmailInvalid.Code {
		t.Errorf("code: want %q, got %v", errs.EmailInvalid.Code, body["code"])
	}
}

// The frontend's Permission[] contract needs the role's permissions
// populated, not null — that drives what the UI shows or hides (M3).
func TestUserEndpointReturnsRolePermissions(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	testutil.SeedUser(t, db, "ada@example.com", "s3cret-password", "owner")
	testutil.GrantPermission(t, db, "owner", "edit_users")
	cookie := testutil.Login(t, app, "ada@example.com", "s3cret-password")

	req := testutil.NewRequest(http.MethodGet, "/api/v1/user", nil, cookie)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}

	var body struct {
		Role struct {
			Permissions []struct {
				Name string `json:"name"`
			} `json:"permissions"`
		} `json:"role"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if len(body.Role.Permissions) == 0 {
		t.Fatal("role.permissions was empty/null — expected the seeded \"edit_users\" permission")
	}
	found := false
	for _, p := range body.Role.Permissions {
		if p.Name == "edit_users" {
			found = true
		}
	}
	if !found {
		t.Errorf("want role.permissions to include \"edit_users\", got %+v", body.Role.Permissions)
	}
}

func TestLoginRejectsWrongPassword(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)
	testutil.SeedUser(t, db, "owner@example.com", "s3cret-password", "owner")

	req := testutil.NewRequest(http.MethodPost, "/api/v1/login",
		testutil.JSON(`{"email":"owner@example.com","password":"test"}`), nil)

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if resp.StatusCode == http.StatusOK {
		t.Fatal("login with the literal \"test\" must fail")
	}
}

// Unknown email and wrong password must be indistinguishable, or the endpoint
// is a user-enumeration oracle (ASSESSMENT 4i).
func TestLoginDoesNotLeakAccountExistence(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)
	testutil.SeedUser(t, db, "owner@example.com", "s3cret-password", "owner")

	unknown := testutil.NewRequest(http.MethodPost, "/api/v1/login",
		testutil.JSON(`{"email":"nobody@example.com","password":"whatever"}`), nil)
	wrongPass := testutil.NewRequest(http.MethodPost, "/api/v1/login",
		testutil.JSON(`{"email":"owner@example.com","password":"whatever"}`), nil)

	r1, _ := app.Test(unknown, -1)
	r2, _ := app.Test(wrongPass, -1)

	if r1.StatusCode != r2.StatusCode {
		t.Fatalf("status must match: unknown=%d wrong-password=%d", r1.StatusCode, r2.StatusCode)
	}

	var b1, b2 map[string]any
	json.NewDecoder(r1.Body).Decode(&b1)
	json.NewDecoder(r2.Body).Decode(&b2)

	if b1["message"] != b2["message"] || b1["code"] != b2["code"] {
		t.Fatalf("bodies must match: %v vs %v", b1, b2)
	}
	if b1["code"] != errs.InvalidCredentials.Code {
		t.Errorf("want code %q, got %v", errs.InvalidCredentials.Code, b1["code"])
	}
}

// The JWT must never appear in the body — that defeats HTTPOnly.
func TestLoginDoesNotReturnTokenInBody(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)
	testutil.SeedUser(t, db, "owner@example.com", "s3cret-password", "owner")

	req := testutil.NewRequest(http.MethodPost, "/api/v1/login",
		testutil.JSON(`{"email":"owner@example.com","password":"s3cret-password"}`), nil)
	resp, _ := app.Test(req, -1)

	var body map[string]any
	json.NewDecoder(resp.Body).Decode(&body)
	if _, present := body["token"]; present {
		t.Fatal("the body must not carry the token; the cookie is the only channel")
	}
}

func TestLoginSetsHardenedCookie(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)
	testutil.SeedUser(t, db, "owner@example.com", "s3cret-password", "owner")

	req := testutil.NewRequest(http.MethodPost, "/api/v1/login",
		testutil.JSON(`{"email":"owner@example.com","password":"s3cret-password"}`), nil)
	resp, _ := app.Test(req, -1)

	for _, c := range resp.Cookies() {
		if c.Name != "jwt" {
			continue
		}
		if !c.HttpOnly {
			t.Error("the session cookie must be HttpOnly")
		}
		if c.SameSite != http.SameSiteLaxMode {
			t.Errorf("the session cookie must be SameSite=Lax, got %v", c.SameSite)
		}
		return
	}
	t.Fatal("no jwt cookie was set")
}

// A malformed body must be a clean 400 from the registry, never a panic or a
// driver error echoed back.
func TestMalformedBodyReturnsRegistryError(t *testing.T) {
	testutil.NewDB(t)
	app := testutil.NewApp(t)

	req := testutil.NewRequest(http.MethodPost, "/api/v1/login",
		testutil.JSON(`{"email": `), nil)
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", resp.StatusCode)
	}

	var body map[string]any
	json.NewDecoder(resp.Body).Decode(&body)
	if body["code"] != errs.InvalidBody.Code {
		t.Errorf("want code %q, got %v", errs.InvalidBody.Code, body["code"])
	}
}
