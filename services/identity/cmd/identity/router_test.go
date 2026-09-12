package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/loloDawit/go-admin/platform/observability"
	"github.com/loloDawit/go-admin/services/identity/internal/httperr"
	"github.com/loloDawit/go-admin/services/identity/internal/platformcheck"
)

// alwaysFailRepo simulates a database that is unreachable: every call fails
// with a driver-shaped error, and calls are counted so a test can assert a
// route never touched it.
type alwaysFailRepo struct {
	calls int
}

func (r *alwaysFailRepo) SchemaState(context.Context) (platformcheck.SchemaState, error) {
	r.calls++
	return platformcheck.SchemaState{}, errors.New("dial tcp 10.0.0.5:5432: connect: connection refused")
}

func TestPanickingHandlerStillProducesALogLineWithStatus500(t *testing.T) {
	logger, captured := observability.NewCaptured()
	handler := platformcheck.NewHandler(platformcheck.NewService(&alwaysFailRepo{}), "identity", httperr.Write)

	r := newRouter(logger, handler)
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

// This pins spec §6: liveness never touches the database, and readiness
// reports failure without leaking driver detail, even against a database that
// is down for the whole life of the request.
func TestStartupContractHealthzSkipsTheDatabaseAndReadyzReportsFailureSafely(t *testing.T) {
	logger, _ := observability.NewCaptured()
	repo := &alwaysFailRepo{}
	handler := platformcheck.NewHandler(platformcheck.NewService(repo), "identity", httperr.Write)

	r := newRouter(logger, handler)

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
