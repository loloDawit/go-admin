//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func gatewayURL() string {
	if v := os.Getenv("GATEWAY_URL"); v != "" {
		return v
	}
	return "http://localhost:8080"
}

// customerResponse mirrors customer/dto.go's CustomerResponse.
type customerResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

// customerPage mirrors customer/dto.go's PageResponse.
type customerPage struct {
	Items    []customerResponse `json:"items"`
	Page     int                `json:"page"`
	PageSize int                `json:"pageSize"`
	Total    int                `json:"total"`
}

// A freshly created product reaching the list proves the path: through the
// gateway, into catalog, to its own database and back.
func TestCatalogAnswersThroughTheGatewayFromItsOwnDatabase(t *testing.T) {
	c := loggedInClient(t)
	created := createProduct(t, c, uniqueSKU(t, "smoke"), "Smoke test desk", "", 1000)

	var page productPage
	status, env := apiCall(t, c, http.MethodGet, "/api/v1/products", nil, &page)
	if status != http.StatusOK {
		t.Fatalf("list: want 200, got %d (%s)", status, env.Code)
	}

	var found bool
	for _, p := range page.Items {
		if p.ID == created.ID {
			found = true
		}
	}
	if !found {
		t.Errorf("product created through the gateway is absent from catalog's own list: %+v", page.Items)
	}
}

// A freshly created customer reaching the list proves the path: through the
// gateway, into orders, to its own database and back.
func TestOrdersAnswersThroughTheGatewayFromItsOwnDatabase(t *testing.T) {
	c := loggedInClient(t)
	email := uniqueSKU(t, "smoke") + "@example.com"

	var created customerResponse
	status, env := apiCall(t, c, http.MethodPost, "/api/v1/customers", map[string]any{
		"email": email, "name": "Smoke Test Customer",
	}, &created)
	if status != http.StatusCreated {
		t.Fatalf("create customer: want 201, got %d (%s)", status, env.Code)
	}
	t.Cleanup(func() {
		conn := ordersConn(t)
		_, _ = conn.Exec(context.Background(), `DELETE FROM customers WHERE email = $1`, email)
	})

	var page customerPage
	status, env = apiCall(t, c, http.MethodGet, "/api/v1/customers", nil, &page)
	if status != http.StatusOK {
		t.Fatalf("list: want 200, got %d (%s)", status, env.Code)
	}

	var found bool
	for _, cust := range page.Items {
		if cust.ID == created.ID {
			found = true
		}
	}
	if !found {
		t.Errorf("customer created through the gateway is absent from orders' own list: %+v", page.Items)
	}
}

// A rejected login still proves the path: through the gateway, into identity, to its own database.
func TestIdentityAnswersThroughTheGatewayFromItsOwnDatabase(t *testing.T) {
	client := &http.Client{Timeout: 10 * time.Second}

	body := strings.NewReader(`{"email":"nobody@example.com","password":"wrong"}`)
	req, err := http.NewRequest(http.MethodPost, gatewayURL()+"/api/v1/login", body)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-Id", "smoke-identity")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status: want 401, got %d", resp.StatusCode)
	}
	if got := resp.Header.Get("X-Request-Id"); got != "smoke-identity" {
		t.Errorf("request id: want smoke-identity, got %q", got)
	}

	var envelope struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if envelope.Code != "invalid_credentials" {
		t.Errorf("code: want invalid_credentials, got %q", envelope.Code)
	}
}
