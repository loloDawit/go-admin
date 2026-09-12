package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/loloDawit/go-admin/platform/observability"
)

func TestPanickingHandlerStillProducesALogLineWithStatus500(t *testing.T) {
	logger, captured := observability.NewCaptured()

	r := newRouter(logger)
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
