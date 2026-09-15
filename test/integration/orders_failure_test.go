//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"
)

// orderItemResponse mirrors order/dto.go's OrderItemResponse; this package
// cannot import orders' internal package, so the shape is duplicated here.
type orderItemResponse struct {
	ProductID      string `json:"productId"`
	TitleSnapshot  string `json:"titleSnapshot"`
	UnitPriceMinor int64  `json:"unitPriceMinor"`
	Currency       string `json:"currency"`
	Quantity       int    `json:"quantity"`
	LineTotalMinor int64  `json:"lineTotalMinor"`
}

// orderResponse mirrors order/dto.go's OrderResponse.
type orderResponse struct {
	ID         string              `json:"id"`
	Number     string              `json:"number"`
	CustomerID string              `json:"customerId"`
	Status     string              `json:"status"`
	TotalMinor int64               `json:"totalMinor"`
	Currency   string              `json:"currency"`
	Items      []orderItemResponse `json:"items"`
}

// TestAnOrdersLinesSurviveTheProductChangingUnderneath drives the whole path
// through the gateway: create a product in Catalog, order it in Orders, then
// change the product's title and price and archive it, all through Catalog's
// own API. The order must still show what was bought at the price paid, not
// what the product looks like now. This is the first time the listing SQL,
// the resolve-at-create-time snapshot, and the get-by-id query have run
// against real Postgres through a real route, rather than a fake repository.
func TestAnOrdersLinesSurviveTheProductChangingUnderneath(t *testing.T) {
	c := loggedInClient(t)
	conn := ordersConn(t)

	sku := uniqueSKU(t, "snapshot")
	product := createProduct(t, c, sku, "Original Title", "before the change", 5000)

	// Orders only accepts an active product; Catalog's API has no publish
	// route yet, so the transition is made directly, the same way
	// orders_customer_test.go sets order status directly.
	catConn := catalogConn(t)
	if _, err := catConn.Exec(context.Background(), `UPDATE products SET status = 'active' WHERE id = $1`, mustInt64(t, product.ID)); err != nil {
		t.Fatalf("activate product: %v", err)
	}

	customerID := insertCustomer(t, conn, fmt.Sprintf("snapshot-%d@example.com", time.Now().UnixNano()), "Snapshot Customer")

	var created orderResponse
	status, env := apiCall(t, c, http.MethodPost, "/api/v1/orders", map[string]any{
		"customerId": fmt.Sprintf("%d", customerID),
		"items": []map[string]any{
			{"productId": product.ID, "quantity": 2},
		},
	}, &created)
	if status != http.StatusCreated {
		t.Fatalf("create order: want 201, got %d (%s)", status, env.Code)
	}
	t.Cleanup(func() {
		_, _ = conn.Exec(context.Background(), `DELETE FROM orders WHERE id = $1`, mustInt64(t, created.ID))
	})

	if len(created.Items) != 1 {
		t.Fatalf("want one order line, got %d", len(created.Items))
	}
	if created.Items[0].TitleSnapshot != "Original Title" || created.Items[0].UnitPriceMinor != 5000 {
		t.Fatalf("order line at creation: want (Original Title, 5000), got (%s, %d)",
			created.Items[0].TitleSnapshot, created.Items[0].UnitPriceMinor)
	}
	if created.TotalMinor != 10000 {
		t.Fatalf("order total at creation: want 10000, got %d", created.TotalMinor)
	}

	if status, env := apiCall(t, c, http.MethodPatch, "/api/v1/products/"+product.ID, map[string]any{
		"title":      "Changed Title",
		"priceMinor": 9999,
	}, nil); status != http.StatusOK {
		t.Fatalf("update product: want 200, got %d (%s)", status, env.Code)
	}
	if status, env := apiCall(t, c, http.MethodPost, "/api/v1/products/"+product.ID+"/archive", nil, nil); status != http.StatusOK {
		t.Fatalf("archive product: want 200, got %d (%s)", status, env.Code)
	}

	var got orderResponse
	if status, env := apiCall(t, c, http.MethodGet, "/api/v1/orders/"+created.ID, nil, &got); status != http.StatusOK {
		t.Fatalf("get order: want 200, got %d (%s)", status, env.Code)
	}

	if len(got.Items) != 1 {
		t.Fatalf("want one order line, got %d", len(got.Items))
	}
	if got.Items[0].TitleSnapshot != "Original Title" {
		t.Errorf("order line title: want the title at purchase time (Original Title), got %q", got.Items[0].TitleSnapshot)
	}
	if got.Items[0].UnitPriceMinor != 5000 {
		t.Errorf("order line price: want the price paid (5000), got %d", got.Items[0].UnitPriceMinor)
	}
	if got.TotalMinor != 10000 {
		t.Errorf("order total: want 10000 (unaffected by the product's new price), got %d", got.TotalMinor)
	}
}
