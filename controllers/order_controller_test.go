package controllers_test

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/loloDawit/go-admin/internal/testutil"
	"github.com/loloDawit/go-admin/models"
)

func TestOrderTimestampsArePopulated(t *testing.T) {
	db := testutil.NewDB(t)

	order := models.Order{FirstName: "Ada", LastName: "Lovelace", Email: "ada@example.com"}
	if err := db.Create(&order).Error; err != nil {
		t.Fatalf("create order: %v", err)
	}

	var reloaded models.Order
	if err := db.First(&reloaded, order.Id).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}

	if reloaded.CreatedAt.IsZero() {
		t.Fatal("CreatedAt must be populated")
	}
	if time.Since(reloaded.CreatedAt) > time.Minute {
		t.Fatalf("CreatedAt looks wrong: %v", reloaded.CreatedAt)
	}
}

func TestChartGroupsRevenueByRealDate(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	testutil.SeedUser(t, db, "viewer@example.com", "s3cret-password", "viewer")
	testutil.GrantPermission(t, db, "viewer", "view_orders")
	cookie := testutil.Login(t, app, "viewer@example.com", "s3cret-password")

	day1 := models.Order{FirstName: "Ada", LastName: "Lovelace", Email: "ada@example.com", CreatedAt: time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)}
	db.Create(&day1)
	db.Create(&models.OrderItem{OrderId: day1.Id, ProductTitle: "Widget", Price: 10.50, Quantity: 2})
	db.Create(&models.OrderItem{OrderId: day1.Id, ProductTitle: "Gizmo", Price: 5.25, Quantity: 4})

	day2 := models.Order{FirstName: "Grace", LastName: "Hopper", Email: "grace@example.com", CreatedAt: time.Date(2026, 1, 16, 8, 0, 0, 0, time.UTC)}
	db.Create(&day2)
	db.Create(&models.OrderItem{OrderId: day2.Id, ProductTitle: "Gadget", Price: 2, Quantity: 3})

	req := testutil.NewRequest(http.MethodGet, "/api/v1/chart", nil, cookie)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}

	var sales []struct {
		Date string  `json:"date"`
		Sum  float64 `json:"sum"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&sales); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if len(sales) != 2 {
		t.Fatalf("want two days of sales, got %d: %+v", len(sales), sales)
	}
	if sales[0].Date != "2026-01-15" {
		t.Errorf("date[0]: want 2026-01-15, got %q", sales[0].Date)
	}
	if want := 10.50*2 + 5.25*4; sales[0].Sum != want {
		t.Errorf("sum[0]: want %.2f, got %.2f", want, sales[0].Sum)
	}
	if sales[1].Date != "2026-01-16" {
		t.Errorf("date[1]: want 2026-01-16, got %q", sales[1].Date)
	}
	if want := 2.0 * 3; sales[1].Sum != want {
		t.Errorf("sum[1]: want %.2f, got %.2f", want, sales[1].Sum)
	}
}

func TestExportWritesExactPricesNotTruncatedIntegers(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	testutil.SeedUser(t, db, "exporter@example.com", "s3cret-password", "exporter")
	testutil.GrantPermission(t, db, "exporter", "view_orders")
	cookie := testutil.Login(t, app, "exporter@example.com", "s3cret-password")

	order := models.Order{FirstName: "Ada", LastName: "Lovelace", Email: "ada@example.com"}
	db.Create(&order)
	db.Create(&models.OrderItem{OrderId: order.Id, ProductTitle: "Widget", Price: 10.50, Quantity: 2})

	tempPattern := filepath.Join(os.TempDir(), "orders-*.csv")
	before, _ := filepath.Glob(tempPattern)

	req := testutil.NewRequest(http.MethodGet, "/api/v1/export", nil, cookie)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(resp.Header.Get("Content-Disposition"), "orders.csv") {
		t.Errorf("Content-Disposition: got %q", resp.Header.Get("Content-Disposition"))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	csv := string(body)
	if !strings.Contains(csv, "ID,Name,Email,Product Title,Price,Quantity") {
		t.Errorf("missing header row: %q", csv)
	}
	if !strings.Contains(csv, ",,,Widget,10.50,2") {
		t.Errorf("price must be formatted as 10.50, not truncated; got %q", csv)
	}

	after, _ := filepath.Glob(tempPattern)
	if len(after) > len(before) {
		t.Errorf("export left a temp file behind: before=%v after=%v", before, after)
	}
}

func TestCreateOrderAcceptsCustomerNameAndComputesTotal(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	testutil.SeedUser(t, db, "creator@example.com", "s3cret-password", "creator")
	testutil.GrantPermission(t, db, "creator", "edit_orders")
	cookie := testutil.Login(t, app, "creator@example.com", "s3cret-password")

	body := `{"firstName":"Ada","lastName":"Lovelace","email":"ada@example.com",
		"orderItems":[{"productTitle":"Widget","price":3,"quantity":2}]}`
	req := testutil.NewRequest(http.MethodPost, "/api/v1/orders", testutil.JSON(body), cookie)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("want 201, got %d", resp.StatusCode)
	}

	var got struct {
		Name  string  `json:"name"`
		Total float64 `json:"total"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Name != "Ada Lovelace" {
		t.Errorf("name: want %q, got %q", "Ada Lovelace", got.Name)
	}
	if got.Total != 6 {
		t.Errorf("total: want 6, got %v", got.Total)
	}

	var reloaded models.Order
	if err := db.First(&reloaded, "email = ?", "ada@example.com").Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.FirstName != "Ada" || reloaded.LastName != "Lovelace" {
		t.Errorf("name must be persisted; got %q %q", reloaded.FirstName, reloaded.LastName)
	}
}

func TestGetOrderComputesNameAndTotal(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	testutil.SeedUser(t, db, "viewer2@example.com", "s3cret-password", "viewer2")
	testutil.GrantPermission(t, db, "viewer2", "view_orders")
	cookie := testutil.Login(t, app, "viewer2@example.com", "s3cret-password")

	order := models.Order{FirstName: "Grace", LastName: "Hopper", Email: "grace@example.com"}
	db.Create(&order)
	db.Create(&models.OrderItem{OrderId: order.Id, ProductTitle: "Widget", Price: 3, Quantity: 2})

	req := testutil.NewRequest(http.MethodGet, "/api/v1/order/"+strconv.Itoa(int(order.Id)), nil, cookie)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}

	var body struct {
		Name  string  `json:"name"`
		Total float64 `json:"total"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Name != "Grace Hopper" {
		t.Errorf("name: want %q, got %q", "Grace Hopper", body.Name)
	}
	if body.Total != 6 {
		t.Errorf("total: want 6, got %v", body.Total)
	}
}

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

func TestUpdateOrderResponseIncludesComputedNameAndTotal(t *testing.T) {
	db := testutil.NewDB(t)
	app := testutil.NewApp(t)

	testutil.SeedUser(t, db, "editor3@example.com", "s3cret-password", "editor")
	testutil.GrantPermission(t, db, "editor", "edit_orders")
	cookie := testutil.Login(t, app, "editor3@example.com", "s3cret-password")

	order := models.Order{
		FirstName: "Ada", LastName: "Lovelace", Email: "buyer3@example.com",
		OrderItems: []models.OrderItem{{ProductTitle: "Widget", Price: 3, Quantity: 2}},
	}
	if err := db.Create(&order).Error; err != nil {
		t.Fatalf("seed order: %v", err)
	}

	body := `{"email":"new@example.com"}`
	req := testutil.NewRequest(http.MethodPut, "/api/v1/order/"+strconv.Itoa(int(order.Id)),
		testutil.JSON(body), cookie)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}

	var got struct {
		Name  string  `json:"name"`
		Total float64 `json:"total"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Name != "Ada Lovelace" {
		t.Errorf("name: want %q, got %q", "Ada Lovelace", got.Name)
	}
	if got.Total != 6 {
		t.Errorf("total: want 6, got %v", got.Total)
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
