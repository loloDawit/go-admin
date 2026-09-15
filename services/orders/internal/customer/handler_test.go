package customer_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/loloDawit/go-admin/services/orders/internal/customer"
)

func newTestHandler() (*customer.Handler, *fakeRepository, *error) {
	svc, repo := newTestService()
	var captured error
	h := customer.NewHandler(svc, 1<<20, func(_ context.Context, w http.ResponseWriter, err error) {
		captured = err
		w.WriteHeader(http.StatusInternalServerError)
	})
	return h, repo, &captured
}

func withChiURLParam(req *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func TestCreateHandlerReturnsTheCreatedCustomer(t *testing.T) {
	h, _, _ := newTestHandler()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/customers", strings.NewReader(`{"email":"handler@example.com","name":"Handler Test"}`))
	rec := httptest.NewRecorder()
	h.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status: want 201, got %d, body=%s", rec.Code, rec.Body.String())
	}
	var resp customer.CustomerResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Email != "handler@example.com" {
		t.Fatalf("email: want handler@example.com, got %q", resp.Email)
	}
}

func TestCreateHandlerRefusesAnEmptyEmail(t *testing.T) {
	h, _, captured := newTestHandler()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/customers", strings.NewReader(`{"email":"","name":"No Email"}`))
	rec := httptest.NewRecorder()
	h.Create(rec, req)

	if *captured == nil {
		t.Fatal("want an error for an empty email")
	}
}

func TestGetHandlerRefusesANonNumericID(t *testing.T) {
	h, _, captured := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/customers/not-a-number", nil)
	req = withChiURLParam(req, "id", "not-a-number")
	rec := httptest.NewRecorder()
	h.Get(rec, req)

	if *captured == nil {
		t.Fatal("want an error for a non-numeric id")
	}
}

func TestByEmailHandlerFindsARegardlessOfCase(t *testing.T) {
	h, repo, _ := newTestHandler()
	created, err := repo.Create(t.Context(), customer.CreateCustomer{Email: "Mixed-Case@Example.com", Name: "Mixed"})
	if err != nil {
		t.Fatalf("seed create: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/customers?email=mixed-case@example.com", nil)
	rec := httptest.NewRecorder()
	h.ByEmail(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d, body=%s", rec.Code, rec.Body.String())
	}
	var resp customer.CustomerResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ID != strconv.FormatInt(created.ID, 10) {
		t.Fatalf("id: want %d, got %s", created.ID, resp.ID)
	}
}

func TestByEmailHandlerRefusesAMissingParam(t *testing.T) {
	h, _, captured := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/customers", nil)
	rec := httptest.NewRecorder()
	h.ByEmail(rec, req)

	if *captured == nil {
		t.Fatal("want an error for a missing email param")
	}
}

func TestListHandlerRefusesANonNumericPageSize(t *testing.T) {
	h, _, captured := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/customers?pageSize=lots", nil)
	rec := httptest.NewRecorder()
	h.List(rec, req)

	if *captured == nil {
		t.Fatal("want an error for a non-numeric pageSize")
	}
}

func TestLifetimeValueHandlerRefusesANonNumericID(t *testing.T) {
	h, _, captured := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/customers/not-a-number/lifetime-value", nil)
	req = withChiURLParam(req, "id", "not-a-number")
	rec := httptest.NewRecorder()
	h.LifetimeValue(rec, req)

	if *captured == nil {
		t.Fatal("want an error for a non-numeric id")
	}
}
