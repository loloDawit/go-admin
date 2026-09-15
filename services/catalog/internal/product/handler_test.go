package product_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/loloDawit/go-admin/services/catalog/internal/product"
)

func newTestHandler() (*product.Handler, *fakeRepository, *error) {
	svc, repo := newTestService()
	var captured error
	h := product.NewHandler(svc, 1<<20, func(_ context.Context, w http.ResponseWriter, err error) {
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

// A JSON number round-trips int64 exactly; this pins that neither the
// request DTO nor the response DTO substitutes a float64 anywhere on the way.
func TestCreateHandlerRoundTripsAnInt64BeyondFloat64Precision(t *testing.T) {
	h, _, _ := newTestHandler()
	const exact = 9007199254740993

	body := `{"sku":"h-1","title":"Handler precision","priceMinor":` + strconv.FormatInt(exact, 10) + `,"currency":"GBP"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/products", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status: want 201, got %d, body=%s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "9007199254740992") {
		t.Fatalf("response rounded through float64: %s", rec.Body.String())
	}
	var resp product.ProductResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.PriceMinor != exact {
		t.Fatalf("priceMinor: want %d, got %d", int64(exact), resp.PriceMinor)
	}
}

func TestUpdateHandlerPatchLeavesOmittedFieldsUnchanged(t *testing.T) {
	h, repo, _ := newTestHandler()
	created, err := repo.Create(t.Context(), product.CreateProduct{SKU: "h-2", Title: "Original", Description: "Keep me", PriceMinor: 100, Currency: "GBP"})
	if err != nil {
		t.Fatalf("seed create: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/products/1", strings.NewReader(`{"title":"New title"}`))
	req = withChiURLParam(req, "id", strconv.FormatInt(created.ID, 10))
	rec := httptest.NewRecorder()
	h.Update(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d, body=%s", rec.Code, rec.Body.String())
	}
	var resp product.ProductResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Description != "Keep me" {
		t.Fatalf("description must survive an omitting PATCH, got %q", resp.Description)
	}
}

func TestGetHandlerRefusesANonNumericID(t *testing.T) {
	h, _, captured := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/products/not-a-number", nil)
	req = withChiURLParam(req, "id", "not-a-number")
	rec := httptest.NewRecorder()
	h.Get(rec, req)

	if *captured == nil {
		t.Fatal("want an error for a non-numeric id")
	}
}

// A status value outside {draft, active, archived} must be refused before it
// reaches the query, not passed through to the enum column as a driver error.
func TestListHandlerRefusesAnUnknownStatus(t *testing.T) {
	h, _, captured := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/products?status=deleted", nil)
	rec := httptest.NewRecorder()
	h.List(rec, req)

	if *captured == nil {
		t.Fatal("want an error for an unrecognized status value")
	}
}

func TestListHandlerRefusesANonNumericPageSize(t *testing.T) {
	h, _, captured := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/products?pageSize=lots", nil)
	rec := httptest.NewRecorder()
	h.List(rec, req)

	if *captured == nil {
		t.Fatal("want an error for a non-numeric pageSize")
	}
}

// The sort string reaches the handler untouched; the allowlist rejection
// happens at the postgres layer (proven in postgres_test.go), not here. This
// only pins that an invalid sort surfaces as the handler's error, not a panic.
func TestListHandlerSurfacesAnInvalidSortAsAnError(t *testing.T) {
	h, _, captured := newTestHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/products?sort=title%3BDROP+TABLE+products", nil)
	rec := httptest.NewRecorder()
	h.List(rec, req)

	if *captured == nil {
		t.Fatal("want an error for an injected sort value")
	}
}

func TestResolveHandlerReturnsKnownProductsAndOmitsUnknownOnes(t *testing.T) {
	h, repo, _ := newTestHandler()
	created, err := repo.Create(t.Context(), product.CreateProduct{SKU: "resolve-h-1", Title: "Resolvable", PriceMinor: 500, Currency: "GBP"})
	if err != nil {
		t.Fatalf("seed create: %v", err)
	}

	body := `{"ids":["` + strconv.FormatInt(created.ID, 10) + `","999999"]}`
	req := httptest.NewRequest(http.MethodPost, "/internal/products/resolve", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.Resolve(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d, body=%s", rec.Code, rec.Body.String())
	}
	var resp product.ResolveResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Products) != 1 || resp.Products[0].Title != "Resolvable" {
		t.Fatalf("want only the known product resolved, got %+v", resp.Products)
	}
}

func TestResolveHandlerRefusesANonNumericID(t *testing.T) {
	h, _, captured := newTestHandler()
	req := httptest.NewRequest(http.MethodPost, "/internal/products/resolve", strings.NewReader(`{"ids":["not-a-number"]}`))
	rec := httptest.NewRecorder()
	h.Resolve(rec, req)

	if *captured == nil {
		t.Fatal("want an error for a non-numeric id")
	}
}
