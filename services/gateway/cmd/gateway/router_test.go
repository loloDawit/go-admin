package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/loloDawit/go-admin/platform/httpx"
	"github.com/loloDawit/go-admin/platform/observability"
	"github.com/loloDawit/go-admin/platform/requestid"
	"github.com/loloDawit/go-admin/services/gateway/internal/routing"
)

func TestRouterProxiesThroughTheFullMiddlewareStack(t *testing.T) {
	var seenByUpstream string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenByUpstream = r.Header.Get(requestid.Header)
		w.Header().Set(requestid.Header, seenByUpstream)
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"service": "identity"})
	}))
	defer upstream.Close()

	upstreams, err := routing.New(map[string]string{"identity": upstream.URL}, time.Second)
	if err != nil {
		t.Fatalf("routing.New: %v", err)
	}

	logger, captured := observability.NewCaptured()
	r := newRouter(logger, upstreams)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/_platform/identity", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", rec.Code)
	}

	values := rec.Header().Values(requestid.Header)
	if got := len(values); got != 1 {
		t.Fatalf("X-Request-Id header count: want 1, got %d (%v)", got, values)
	}
	if seenByUpstream != values[0] {
		t.Errorf("request ID mismatch: response has %q, upstream saw %q", values[0], seenByUpstream)
	}

	records := captured.Records()
	if len(records) != 1 {
		t.Fatalf("want 1 logged request, got %d", len(records))
	}
	route, ok := captured.Attr(0, "route")
	if !ok || route.String() != "/_platform/identity" {
		t.Errorf("route: want /_platform/identity, got %v (ok=%v)", route, ok)
	}
}

func TestRouterRecoversFromAPanicAndStillLogs(t *testing.T) {
	upstreams, err := routing.New(map[string]string{"identity": "http://127.0.0.1:1"}, time.Second)
	if err != nil {
		t.Fatalf("routing.New: %v", err)
	}

	logger, captured := observability.NewCaptured()
	r := newRouter(logger, upstreams)
	r.Get("/panics", func(http.ResponseWriter, *http.Request) { panic("boom") })

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/panics", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status: want 500, got %d", rec.Code)
	}

	records := captured.Records()
	if len(records) != 1 {
		t.Fatalf("want 1 logged request despite the panic, got %d", len(records))
	}
	status, ok := captured.Attr(0, "status")
	if !ok || status.Int64() != http.StatusInternalServerError {
		t.Errorf("logged status: want 500, got %v (ok=%v)", status, ok)
	}
}
