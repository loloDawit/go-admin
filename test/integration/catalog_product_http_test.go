//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"
)

type productResponse struct {
	ID          string `json:"id"`
	SKU         string `json:"sku"`
	Title       string `json:"title"`
	Description string `json:"description"`
	PriceMinor  int64  `json:"priceMinor"`
	Currency    string `json:"currency"`
	Status      string `json:"status"`
}

type productPage struct {
	Items    []productResponse `json:"items"`
	Page     int               `json:"page"`
	PageSize int               `json:"pageSize"`
	Total    int               `json:"total"`
}

// createProduct drives the real route through the gateway, so the repository's
// own SQL runs. Tests that retype a query prove Postgres's semantics, not that
// the code issues that query — a parameter-numbering fault in the list query
// reached a 500 in the running service while every such test stayed green.
func createProduct(t *testing.T, c *http.Client, sku, title, description string, price int64) productResponse {
	t.Helper()
	var created productResponse
	status, env := apiCall(t, c, http.MethodPost, "/api/v1/products", map[string]any{
		"sku": sku, "title": title, "description": description, "priceMinor": price,
	}, &created)
	if status != http.StatusCreated {
		t.Fatalf("create %s: want 201, got %d (%s)", sku, status, env.Code)
	}
	// Delete the row rather than archiving it: archived products are still rows,
	// and a test that counts them would see every other test's leftovers.
	t.Cleanup(func() {
		conn := catalogConn(t)
		_, _ = conn.Exec(context.Background(), `DELETE FROM products WHERE sku = $1`, sku)
	})
	return created
}

func uniqueSKU(t *testing.T, prefix string) string {
	t.Helper()
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

func TestProductListThroughTheGateway(t *testing.T) {
	c := loggedInClient(t)
	created := createProduct(t, c, uniqueSKU(t, "list"), "Walnut desk", "a sturdy oak surface", 14999)

	var page productPage
	status, env := apiCall(t, c, http.MethodGet, "/api/v1/products", nil, &page)
	if status != http.StatusOK {
		t.Fatalf("list: want 200, got %d (%s)", status, env.Code)
	}
	if page.PageSize == 0 {
		t.Error("list did not state the effective page size")
	}

	var found bool
	for _, p := range page.Items {
		if p.ID == created.ID {
			found = true
			if p.PriceMinor != 14999 {
				t.Errorf("price: want 14999, got %d", p.PriceMinor)
			}
		}
	}
	if !found {
		t.Errorf("created product absent from the list: %+v", page.Items)
	}
}

func TestProductSearchThroughTheGatewayRanksTitleFirst(t *testing.T) {
	c := loggedInClient(t)
	stamp := time.Now().UnixNano()
	createProduct(t, c, fmt.Sprintf("desc-%d", stamp), fmt.Sprintf("Walnut desk %d", stamp), fmt.Sprintf("a sturdy zqxoak%d surface", stamp), 1000)
	createProduct(t, c, fmt.Sprintf("title-%d", stamp), fmt.Sprintf("Zqxoak%d stool", stamp), "walnut legs", 1000)

	var page productPage
	status, env := apiCall(t, c, http.MethodGet, fmt.Sprintf("/api/v1/products?q=zqxoak%d", stamp), nil, &page)
	if status != http.StatusOK {
		t.Fatalf("search: want 200, got %d (%s)", status, env.Code)
	}
	if len(page.Items) != 2 {
		t.Fatalf("want both products matched, got %d", len(page.Items))
	}
	if page.Items[0].SKU != fmt.Sprintf("title-%d", stamp) {
		t.Errorf("title match must rank first, got %s", page.Items[0].SKU)
	}
}

func TestProductPagingParametersReachTheQuery(t *testing.T) {
	c := loggedInClient(t)
	createProduct(t, c, uniqueSKU(t, "page"), "Paged", "", 1000)

	var page productPage
	status, _ := apiCall(t, c, http.MethodGet, "/api/v1/products?page=1&pageSize=1", nil, &page)
	if status != http.StatusOK {
		t.Fatalf("paged list: want 200, got %d", status)
	}
	if len(page.Items) > 1 {
		t.Errorf("pageSize=1 returned %d items", len(page.Items))
	}
}

func TestProductSortThroughTheGateway(t *testing.T) {
	c := loggedInClient(t)

	var page productPage
	if status, _ := apiCall(t, c, http.MethodGet, "/api/v1/products?sort=title", nil, &page); status != http.StatusOK {
		t.Fatalf("sort=title: want 200, got %d", status)
	}
	if status, env := apiCall(t, c, http.MethodGet, "/api/v1/products?sort=title;DROP TABLE products", nil, nil); status == http.StatusOK {
		t.Fatalf("an injected sort value was accepted (%s)", env.Code)
	}
}

func TestArchivedProductLeavesTheDefaultListButStaysRetrievable(t *testing.T) {
	c := loggedInClient(t)
	created := createProduct(t, c, uniqueSKU(t, "arch"), "Archivable", "", 1000)

	if status, env := apiCall(t, c, http.MethodPost, "/api/v1/products/"+created.ID+"/archive", nil, nil); status != http.StatusOK {
		t.Fatalf("archive: want 200, got %d (%s)", status, env.Code)
	}

	var page productPage
	apiCall(t, c, http.MethodGet, "/api/v1/products", nil, &page)
	for _, p := range page.Items {
		if p.ID == created.ID {
			t.Error("an archived product remained in the default list")
		}
	}

	var one productResponse
	if status, _ := apiCall(t, c, http.MethodGet, "/api/v1/products/"+created.ID, nil, &one); status != http.StatusOK {
		t.Fatalf("archived product must stay retrievable by id, got %d", status)
	}
	if one.Status != "archived" {
		t.Errorf("status: want archived, got %q", one.Status)
	}
}
