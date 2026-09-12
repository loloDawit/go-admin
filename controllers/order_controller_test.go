package controllers_test

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/loloDawit/go-admin/internal/testutil"
	"github.com/loloDawit/go-admin/models"
)

func TestUpdateOrderIgnoresOrderItemsAndKeepsEmail(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	testutil.SeedUser(t, db, "editor@example.com", "s3cret-password", "editor")
	testutil.GrantPermission(t, db, "editor", "edit_orders")
	cookie := testutil.Login(t, app, "editor@example.com", "s3cret-password")

	order := models.Order{Email: "buyer@example.com", OrderItems: []models.OrderItem{
		{ProductTitle: "Widget", Price: 5, Quantity: 2},
	}}
	if err := db.Create(&order).Error; err != nil {
		t.Fatalf("seed order: %v", err)
	}

	body := `{"orderItems":[{"productTitle":"Smuggled","price":1,"quantity":1}]}`
	req := testutil.NewRequest(http.MethodPut, "/api/v1/order/"+strconv.Itoa(int(order.Id)),
		testutil.JSON(body), cookie)

	if _, err := app.Test(req, -1); err != nil {
		t.Fatalf("request: %v", err)
	}

	var reloaded models.Order
	if err := db.Preload("OrderItems").First(&reloaded, order.Id).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.Email != "buyer@example.com" {
		t.Errorf("email must not be blanked by a request that only sends orderItems; got %q", reloaded.Email)
	}
	if len(reloaded.OrderItems) != 1 || reloaded.OrderItems[0].ProductTitle != "Widget" {
		t.Errorf("order items must be unaffected by UpdateOrder; got %+v", reloaded.OrderItems)
	}
}

func TestDeleteOrderWithItemsReturns409(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	testutil.SeedUser(t, db, "editor2@example.com", "s3cret-password", "editor")
	testutil.GrantPermission(t, db, "editor", "edit_orders")
	cookie := testutil.Login(t, app, "editor2@example.com", "s3cret-password")

	order := models.Order{Email: "buyer2@example.com", OrderItems: []models.OrderItem{
		{ProductTitle: "Widget", Price: 5, Quantity: 2},
	}}
	if err := db.Create(&order).Error; err != nil {
		t.Fatalf("seed order: %v", err)
	}

	req := testutil.NewRequest(http.MethodDelete, "/api/v1/order/"+strconv.Itoa(int(order.Id)), nil, cookie)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("want 409 for an order with line items, got %d", resp.StatusCode)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["code"] != "resource_in_use" {
		t.Errorf("code: want resource_in_use, got %v", body["code"])
	}
}
