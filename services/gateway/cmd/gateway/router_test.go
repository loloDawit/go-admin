package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/loloDawit/go-admin/platform/httpx"
	"github.com/loloDawit/go-admin/platform/observability"
	"github.com/loloDawit/go-admin/platform/principal"
	"github.com/loloDawit/go-admin/platform/requestid"
	"github.com/loloDawit/go-admin/services/gateway/internal/auth"
	"github.com/loloDawit/go-admin/services/gateway/internal/httperr"
	"github.com/loloDawit/go-admin/services/gateway/internal/routing"
)

const testSigningKey = "test-signing-key-at-least-32-bytes-long"

func noSessionValidator(t *testing.T, logger *slog.Logger) *auth.Validator {
	t.Helper()
	stub := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("identity must not be called for a request with no session cookie")
	}))
	t.Cleanup(stub.Close)
	return auth.NewValidator(stub.Client(), stub.URL+"/internal/sessions/validate", []byte(testSigningKey), time.Minute, auth.NewCache(time.Minute), httperr.New(logger), logger)
}

func TestRouterProxiesThroughTheFullMiddlewareStack(t *testing.T) {
	var seenByUpstream string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenByUpstream = r.Header.Get(requestid.Header)
		w.Header().Set(requestid.Header, seenByUpstream)
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"service": "orders"})
	}))
	defer upstream.Close()

	logger, captured := observability.NewCaptured()
	upstreams, err := routing.New(logger, map[string]string{"orders": upstream.URL}, time.Second, nil)
	if err != nil {
		t.Fatalf("routing.New: %v", err)
	}

	r := newRouter(logger, upstreams, noSessionValidator(t, logger))

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil))

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
	if !ok || route.String() != "/api/v1/orders" {
		t.Errorf("route: want /api/v1/orders, got %v (ok=%v)", route, ok)
	}
}

func TestRouterStripsAClientSuppliedPrincipal(t *testing.T) {
	var seenHeader, seenSig string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenHeader = r.Header.Get(principal.HeaderPrincipal)
		seenSig = r.Header.Get(principal.HeaderSignature)
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"service": "identity"})
	}))
	defer upstream.Close()

	logger, _ := observability.NewCaptured()
	upstreams, err := routing.New(logger, map[string]string{"identity": upstream.URL}, time.Second, nil)
	if err != nil {
		t.Fatalf("routing.New: %v", err)
	}
	r := newRouter(logger, upstreams, noSessionValidator(t, logger))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	req.Header.Set(principal.HeaderPrincipal, "forged-principal")
	req.Header.Set(principal.HeaderSignature, "forged-signature")

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if seenHeader == "forged-principal" {
		t.Fatal("the gateway forwarded a client-supplied principal")
	}
	if seenSig == "forged-signature" {
		t.Fatal("the gateway forwarded a client-supplied signature")
	}
}

func TestRouterRecoversFromAPanicAndStillLogs(t *testing.T) {
	logger, captured := observability.NewCaptured()
	upstreams, err := routing.New(logger, map[string]string{"identity": "http://127.0.0.1:1"}, time.Second, nil)
	if err != nil {
		t.Fatalf("routing.New: %v", err)
	}

	r := newRouter(logger, upstreams, noSessionValidator(t, logger))
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

func TestHealthzReturns200(t *testing.T) {
	logger, _ := observability.NewCaptured()
	upstreams, err := routing.New(logger, map[string]string{"identity": "http://127.0.0.1:1"}, time.Second, nil)
	if err != nil {
		t.Fatalf("routing.New: %v", err)
	}
	r := newRouter(logger, upstreams, noSessionValidator(t, logger))

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", rec.Code)
	}
}

func TestReadyzReturns200(t *testing.T) {
	logger, _ := observability.NewCaptured()
	upstreams, err := routing.New(logger, map[string]string{"identity": "http://127.0.0.1:1"}, time.Second, nil)
	if err != nil {
		t.Fatalf("routing.New: %v", err)
	}
	r := newRouter(logger, upstreams, noSessionValidator(t, logger))

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", rec.Code)
	}
}

func TestRouterForwardsASignedPrincipalForAValidSession(t *testing.T) {
	var seenHeader, seenSig string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenHeader = r.Header.Get(principal.HeaderPrincipal)
		seenSig = r.Header.Get(principal.HeaderSignature)
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"service": "identity"})
	}))
	defer upstream.Close()

	identityValidate := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"staffId":            "staff-1",
			"permissions":        []string{"orders_view"},
			"mustChangePassword": false,
		})
	}))
	defer identityValidate.Close()

	logger, _ := observability.NewCaptured()
	upstreams, err := routing.New(logger, map[string]string{"identity": upstream.URL}, time.Second, nil)
	if err != nil {
		t.Fatalf("routing.New: %v", err)
	}

	key := []byte(testSigningKey)
	validator := auth.NewValidator(identityValidate.Client(), identityValidate.URL+"/internal/sessions/validate", key, time.Minute, auth.NewCache(time.Minute), httperr.New(logger), logger)
	r := newRouter(logger, upstreams, validator)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "a-live-session-token"})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if seenHeader == "" || seenSig == "" {
		t.Fatal("want a signed principal forwarded for a valid session")
	}
	p, err := principal.Verify(seenHeader, seenSig, key, time.Now())
	if err != nil {
		t.Fatalf("the forwarded principal must verify against the gateway's own key: %v", err)
	}
	if p.StaffID != "staff-1" {
		t.Errorf("StaffID: want staff-1, got %q", p.StaffID)
	}
}
