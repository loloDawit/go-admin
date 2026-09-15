package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/loloDawit/go-admin/platform/observability"
	"github.com/loloDawit/go-admin/platform/readiness"
	"github.com/loloDawit/go-admin/services/orders/internal/customer"
	"github.com/loloDawit/go-admin/services/orders/internal/httperr"
	"github.com/loloDawit/go-admin/services/orders/internal/order"
	"github.com/loloDawit/go-admin/services/orders/internal/schemacheck"
)

// testPrincipalKey is the HMAC key newRouter's principal.Middleware group verifies against.
var testPrincipalKey = []byte("0123456789012345678901234567890123456789")

// alwaysFailRepo counts calls so a test can assert a route never touched it.
type alwaysFailRepo struct {
	calls int
}

func (r *alwaysFailRepo) SchemaState(context.Context) (schemacheck.SchemaState, error) {
	r.calls++
	return schemacheck.SchemaState{}, errors.New("dial tcp 10.0.0.5:5432: connect: connection refused")
}

// nilCustomerRepository lets tests build a real *customer.Handler without a
// database; no route these tests exercise calls it.
type nilCustomerRepository struct{ customer.Repository }

func newTestCustomerHandler() *customer.Handler {
	svc := customer.NewService(nilCustomerRepository{}, 100)
	return customer.NewHandler(svc, 1<<20, func(context.Context, http.ResponseWriter, error) {})
}

// nilOrderRepository and nilProductResolver let tests build a real
// *order.Handler without a database or Catalog; no route these tests
// exercise calls either.
type nilOrderRepository struct{ order.Repository }
type nilProductResolver struct{ order.ProductResolver }

func newTestOrderHandler() *order.Handler {
	svc := order.NewService(nilOrderRepository{}, nilProductResolver{}, 100)
	return order.NewHandler(svc, 1<<20, func(context.Context, http.ResponseWriter, error) {})
}

func TestPanickingHandlerStillProducesALogLineWithStatus500(t *testing.T) {
	logger, captured := observability.NewCaptured()
	errWriter := httperr.New(logger)
	svc := schemacheck.NewService(&alwaysFailRepo{})
	ready := readiness.NewHandler(svc.Probe, errWriter.Write)

	r := newRouter(logger, ready, newTestCustomerHandler(), newTestOrderHandler(), testPrincipalKey, errWriter.Write)
	r.Get("/boom", func(http.ResponseWriter, *http.Request) {
		panic("kaboom")
	})

	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("response status: want 500, got %d", rec.Code)
	}

	records := captured.Records()
	if len(records) != 1 {
		t.Fatalf("want exactly one log record for the panicking request, got %d", len(records))
	}

	v, ok := captured.Attr(0, "status")
	if !ok {
		t.Fatal("log record missing status attribute")
	}
	if v.Int64() != http.StatusInternalServerError {
		t.Errorf("logged status: want 500, got %d", v.Int64())
	}
}

// This pins spec §6: liveness never touches the database, and readiness reports failure without leaking driver detail.
func TestStartupContractHealthzSkipsTheDatabaseAndReadyzReportsFailureSafely(t *testing.T) {
	logger, _ := observability.NewCaptured()
	errWriter := httperr.New(logger)
	repo := &alwaysFailRepo{}
	svc := schemacheck.NewService(repo)
	ready := readiness.NewHandler(svc.Probe, errWriter.Write)

	r := newRouter(logger, ready, newTestCustomerHandler(), newTestOrderHandler(), testPrincipalKey, errWriter.Write)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("healthz status: want 200, got %d", rec.Code)
	}
	if repo.calls != 0 {
		t.Errorf("healthz touched the repository: called %d times", repo.calls)
	}

	for _, route := range []string{"/readyz"} {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, route, nil))
		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("%s status: want 503, got %d", route, rec.Code)
		}
		if body := rec.Body.String(); strings.Contains(body, "10.0.0.5") || strings.Contains(body, "connection refused") {
			t.Errorf("%s body leaked driver detail: %q", route, body)
		}
	}
}
