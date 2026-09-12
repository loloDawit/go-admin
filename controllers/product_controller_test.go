package controllers_test

import (
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/loloDawit/go-admin/internal/testutil"
	"github.com/loloDawit/go-admin/models"
)

func TestUpdateProductIgnoresIdInBody(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	testutil.SeedUser(t, db, "editor@example.com", "s3cret-password", "editor")
	testutil.GrantPermission(t, db, "editor", "edit_products")
	cookie := testutil.Login(t, app, "editor@example.com", "s3cret-password")

	target := models.Product{Title: "Target", Price: 10}
	victim := models.Product{Title: "Victim", Price: 20}
	db.Create(&target)
	db.Create(&victim)

	body := `{"id":` + strconv.Itoa(int(victim.Id)) + `,"title":"Hijacked","price":99}`
	req := testutil.NewRequest(http.MethodPut, "/api/v1/product/"+strconv.Itoa(int(target.Id)),
		strings.NewReader(body), cookie)

	if _, err := app.Test(req, -1); err != nil {
		t.Fatalf("request: %v", err)
	}

	var reloadedVictim models.Product
	db.First(&reloadedVictim, victim.Id)
	if reloadedVictim.Title != "Victim" {
		t.Errorf("the body's id must not redirect the update; victim became %q", reloadedVictim.Title)
	}

	var reloadedTarget models.Product
	db.First(&reloadedTarget, target.Id)
	if reloadedTarget.Title != "Hijacked" {
		t.Errorf("the path id must decide the target; got %q", reloadedTarget.Title)
	}
}

func TestGetMissingProductReturns404(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	testutil.SeedUser(t, db, "viewer@example.com", "s3cret-password", "viewer")
	testutil.GrantPermission(t, db, "viewer", "view_products")
	cookie := testutil.Login(t, app, "viewer@example.com", "s3cret-password")

	req := testutil.NewRequest(http.MethodGet, "/api/v1/product/999999", nil, cookie)
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("want 404 for a missing product, got %d (Find returns an empty struct with 200)",
			resp.StatusCode)
	}
}

func TestNonNumericIdReturns400(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	testutil.SeedUser(t, db, "viewer@example.com", "s3cret-password", "viewer")
	testutil.GrantPermission(t, db, "viewer", "view_products")
	cookie := testutil.Login(t, app, "viewer@example.com", "s3cret-password")

	req := testutil.NewRequest(http.MethodGet, "/api/v1/product/abc", nil, cookie)
	resp, _ := app.Test(req, -1)

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("want 400 for a non-numeric id, got %d (Atoi's error was discarded, giving id 0)",
			resp.StatusCode)
	}
}
