package routing_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/loloDawit/go-admin/platform/httpx"
	"github.com/loloDawit/go-admin/platform/observability"
	"github.com/loloDawit/go-admin/platform/requestid"
	"github.com/loloDawit/go-admin/services/gateway/internal/routing"
)

func TestRoutesToTheNamedUpstream(t *testing.T) {
	logger, _ := observability.NewCaptured()
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/_platform" {
			t.Errorf("upstream path: want /_platform, got %q", r.URL.Path)
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"service": "catalog"})
	}))
	defer upstream.Close()

	h, err := routing.New(logger, map[string]string{"catalog": upstream.URL}, time.Second)
	if err != nil {
		t.Fatalf("routing.New: %v", err)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/_platform/catalog", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", rec.Code)
	}
	var body map[string]string
	json.NewDecoder(rec.Body).Decode(&body)
	if body["service"] != "catalog" {
		t.Errorf("body: got %v", body)
	}
}

func TestUnknownServiceReturnsTheStandardEnvelope(t *testing.T) {
	logger, _ := observability.NewCaptured()
	h, _ := routing.New(logger, map[string]string{"catalog": "http://127.0.0.1:1"}, time.Second)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/_platform/nosuchservice", nil))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status: want 404, got %d", rec.Code)
	}
	var body httpx.ErrorBody
	json.NewDecoder(rec.Body).Decode(&body)
	if body.Code == "" {
		t.Error("an unknown route must still return the standard envelope")
	}
}

// An upstream that is down must not leak its address to the client.
func TestUnreachableUpstreamReturns502WithoutLeakingTheAddress(t *testing.T) {
	logger, _ := observability.NewCaptured()
	h, _ := routing.New(logger, map[string]string{"catalog": "http://127.0.0.1:1"}, time.Second)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/_platform/catalog", nil))

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status: want 502, got %d", rec.Code)
	}
	body := rec.Body.String()
	for _, leak := range []string{"127.0.0.1", "connection refused", "dial tcp"} {
		if strings.Contains(body, leak) {
			t.Errorf("response leaks %q: %s", leak, body)
		}
	}

	var envelope httpx.ErrorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("body is not the standard envelope: %v (%s)", err, body)
	}
	if envelope.Code != "upstream_unavailable" {
		t.Errorf("code: want upstream_unavailable, got %q", envelope.Code)
	}
}

// routing.New installs a custom ErrorHandler to keep the cause out of the
// response body, so it must log the dial error itself instead.
func TestUnreachableUpstreamLogsTheCauseWithUpstreamNameAndRequestID(t *testing.T) {
	logger, captured := observability.NewCaptured()
	h, _ := routing.New(logger, map[string]string{"catalog": "http://127.0.0.1:1"}, time.Second)
	wrapped := requestid.Middleware(h)

	req := httptest.NewRequest(http.MethodGet, "/_platform/catalog", nil)
	req.Header.Set(requestid.Header, "known-id")

	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status: want 502, got %d", rec.Code)
	}

	records := captured.Records()
	if len(records) != 1 {
		t.Fatalf("want exactly one logged error record, got %d", len(records))
	}
	if v, ok := captured.Attr(0, "upstream"); !ok || v.String() != "catalog" {
		t.Errorf("upstream: want %q, got %v (ok=%v)", "catalog", v, ok)
	}
	if v, ok := captured.Attr(0, "request_id"); !ok || v.String() != "known-id" {
		t.Errorf("request_id: want %q, got %v (ok=%v)", "known-id", v, ok)
	}
	if v, ok := captured.Attr(0, "error"); !ok || v.String() == "" {
		t.Error("error attribute missing or empty")
	}
}

// A slow upstream must not hang the client past the configured timeout.
func TestSlowUpstreamTripsTheDeadlineAndReturnsGatewayTimeout(t *testing.T) {
	const timeout = 50 * time.Millisecond

	done := make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(done)
		select {
		case <-r.Context().Done():
			// proves the proxy actually canceled the outbound request
		case <-time.After(2 * time.Second):
			t.Error("upstream handler was not canceled when the gateway's deadline fired")
		}
	}))
	defer upstream.Close()

	logger, _ := observability.NewCaptured()
	h, err := routing.New(logger, map[string]string{"catalog": upstream.URL}, timeout)
	if err != nil {
		t.Fatalf("routing.New: %v", err)
	}

	rec := httptest.NewRecorder()
	start := time.Now()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/_platform/catalog", nil))
	elapsed := time.Since(start)

	<-done // wait for the upstream handler to actually observe cancellation

	if elapsed > 500*time.Millisecond {
		t.Fatalf("request took %v; the gateway must not wait past its own timeout", elapsed)
	}
	if rec.Code != http.StatusGatewayTimeout {
		t.Fatalf("status: want 504, got %d", rec.Code)
	}

	body := rec.Body.String()
	for _, leak := range []string{upstream.URL, "127.0.0.1", "deadline", "dial tcp"} {
		if strings.Contains(body, leak) {
			t.Errorf("response leaks %q: %s", leak, body)
		}
	}

	var envelope httpx.ErrorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("body is not the standard envelope: %v (%s)", err, body)
	}
	if envelope.Code != "gateway_timeout" {
		t.Errorf("code: want gateway_timeout, got %q", envelope.Code)
	}
}

func TestAPIV1RoutesToIdentityPreservingThePath(t *testing.T) {
	logger, _ := observability.NewCaptured()
	var gotPath string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"service": "identity"})
	}))
	defer upstream.Close()

	h, err := routing.New(logger, map[string]string{"identity": upstream.URL}, time.Second)
	if err != nil {
		t.Fatalf("routing.New: %v", err)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/me", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", rec.Code)
	}
	if gotPath != "/api/v1/me" {
		t.Errorf("upstream path: want /api/v1/me (preserved), got %q", gotPath)
	}
}

func TestInternalRoutesAreNotProxied(t *testing.T) {
	logger, _ := observability.NewCaptured()
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("internal routes must never reach an upstream through the gateway")
	}))
	defer upstream.Close()

	h, err := routing.New(logger, map[string]string{"identity": upstream.URL}, time.Second)
	if err != nil {
		t.Fatalf("routing.New: %v", err)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/internal/sessions/validate", nil))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status: want 404, got %d", rec.Code)
	}
}

func TestRejectsAnUnparseableUpstream(t *testing.T) {
	logger, _ := observability.NewCaptured()
	if _, err := routing.New(logger, map[string]string{"identity": "://bad"}, time.Second); err == nil {
		t.Fatal("a malformed upstream URL must be rejected at construction")
	}
}

// Both the upstream and the gateway's own requestid.Middleware set the
// header; only one must survive onto the client response.
func TestProxiedResponseHasExactlyOneRequestIDHeader(t *testing.T) {
	var seenByUpstream string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenByUpstream = r.Header.Get(requestid.Header)
		w.Header().Set(requestid.Header, seenByUpstream)
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"service": "catalog"})
	}))
	defer upstream.Close()

	logger, _ := observability.NewCaptured()
	h, err := routing.New(logger, map[string]string{"catalog": upstream.URL}, time.Second)
	if err != nil {
		t.Fatalf("routing.New: %v", err)
	}
	wrapped := requestid.Middleware(h)

	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/_platform/catalog", nil))

	values := rec.Header().Values(requestid.Header)
	if got := len(values); got != 1 {
		t.Fatalf("X-Request-Id header count: want 1, got %d (%v)", got, values)
	}
	if seenByUpstream == "" {
		t.Error("the gateway must forward the request ID it originated to the upstream")
	}
	if seenByUpstream != values[0] {
		t.Errorf("request ID mismatch: gateway response has %q, upstream saw %q", values[0], seenByUpstream)
	}
}
