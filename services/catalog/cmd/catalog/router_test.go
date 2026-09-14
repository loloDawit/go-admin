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
	"github.com/loloDawit/go-admin/services/catalog/internal/httperr"
	"github.com/loloDawit/go-admin/services/catalog/internal/image"
	"github.com/loloDawit/go-admin/services/catalog/internal/platformcheck"
	"github.com/loloDawit/go-admin/services/catalog/internal/product"
)

// testPrincipalKey is the HMAC key newRouter's principal.Middleware group verifies against.
var testPrincipalKey = []byte("0123456789012345678901234567890123456789")

// nilProductRepository lets tests build a real *product.Handler without a
// database; every route this file exercises never calls it.
type nilProductRepository struct{ product.Repository }

func newTestProductHandler() *product.Handler {
	svc := product.NewService(nilProductRepository{}, 100, "GBP", 100)
	return product.NewHandler(svc, 1<<20, func(context.Context, http.ResponseWriter, error) {})
}

// nilImageRepository and nilImageStore let tests build a real *image.Handler
// without a database or object store; no route these tests exercise calls either.
type nilImageRepository struct{ image.Repository }
type nilImageStore struct{ image.Store }

func newTestImageHandler() *image.Handler {
	svc := image.NewService(nilImageRepository{}, nilImageStore{}, 1<<20)
	return image.NewHandler(svc, 1<<20, func(context.Context, http.ResponseWriter, error) {})
}

// alwaysFailRepo counts calls so a test can assert a route never touched it.
type alwaysFailRepo struct {
	calls int
}

func (r *alwaysFailRepo) SchemaState(context.Context) (platformcheck.SchemaState, error) {
	r.calls++
	return platformcheck.SchemaState{}, errors.New("dial tcp 10.0.0.5:5432: connect: connection refused")
}

func TestPanickingHandlerStillProducesALogLineWithStatus500(t *testing.T) {
	logger, captured := observability.NewCaptured()
	errWriter := httperr.New(logger)
	svc := platformcheck.NewService(&alwaysFailRepo{})
	handler := platformcheck.NewHandler(svc, "catalog", errWriter.Write)
	ready := readiness.NewHandler(svc.Probe, errWriter.Write)

	r := newRouter(logger, handler, ready, newTestProductHandler(), newTestImageHandler(), testPrincipalKey, errWriter.Write)
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
	svc := platformcheck.NewService(repo)
	handler := platformcheck.NewHandler(svc, "catalog", errWriter.Write)
	ready := readiness.NewHandler(svc.Probe, errWriter.Write)

	r := newRouter(logger, handler, ready, newTestProductHandler(), newTestImageHandler(), testPrincipalKey, errWriter.Write)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("healthz status: want 200, got %d", rec.Code)
	}
	if repo.calls != 0 {
		t.Errorf("healthz touched the repository: called %d times", repo.calls)
	}

	for _, route := range []string{"/readyz", "/_platform"} {
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

// The resolve route is reachable directly against the service (the gateway
// is what refuses it; see routing_test.go's TestResolveIsNotReachableThroughTheGateway).
func TestResolveRouteIsRegisteredOutsideAnyAuthenticatedGroup(t *testing.T) {
	logger, _ := observability.NewCaptured()
	errWriter := httperr.New(logger)
	svc := platformcheck.NewService(&alwaysFailRepo{})
	handler := platformcheck.NewHandler(svc, "catalog", errWriter.Write)
	ready := readiness.NewHandler(svc.Probe, errWriter.Write)

	r := newRouter(logger, handler, ready, newTestProductHandler(), newTestImageHandler(), testPrincipalKey, errWriter.Write)

	req := httptest.NewRequest(http.MethodPost, "/internal/products/resolve", strings.NewReader(`{"ids":["not-a-number"]}`))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code == http.StatusNotFound {
		t.Fatal("the resolve route must be registered, not 404")
	}
}
