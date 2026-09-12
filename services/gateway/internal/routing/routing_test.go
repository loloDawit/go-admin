package routing_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/loloDawit/go-admin/platform/httpx"
	"github.com/loloDawit/go-admin/platform/requestid"
	"github.com/loloDawit/go-admin/services/gateway/internal/routing"
)

func TestRoutesToTheNamedUpstream(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/_platform" {
			t.Errorf("upstream path: want /_platform, got %q", r.URL.Path)
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"service": "identity"})
	}))
	defer upstream.Close()

	h, err := routing.New(map[string]string{"identity": upstream.URL}, time.Second)
	if err != nil {
		t.Fatalf("routing.New: %v", err)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/_platform/identity", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", rec.Code)
	}
	var body map[string]string
	json.NewDecoder(rec.Body).Decode(&body)
	if body["service"] != "identity" {
		t.Errorf("body: got %v", body)
	}
}

func TestUnknownServiceReturnsTheStandardEnvelope(t *testing.T) {
	h, _ := routing.New(map[string]string{"identity": "http://127.0.0.1:1"}, time.Second)

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
	h, _ := routing.New(map[string]string{"identity": "http://127.0.0.1:1"}, time.Second)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/_platform/identity", nil))

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

// A slow upstream must not hang the client past the configured timeout, and the
// client must get the standard error envelope rather than the connection being
// left open or any transport detail leaking into the body.
func TestSlowUpstreamTripsTheDeadlineAndReturnsGatewayTimeout(t *testing.T) {
	const timeout = 50 * time.Millisecond

	done := make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(done)
		select {
		case <-r.Context().Done():
			// Proves the proxy actually canceled the outbound request instead of
			// the handler completing normally and then the client just being made
			// to wait for it.
		case <-time.After(2 * time.Second):
			t.Error("upstream handler was not canceled when the gateway's deadline fired")
		}
	}))
	defer upstream.Close()

	h, err := routing.New(map[string]string{"identity": upstream.URL}, timeout)
	if err != nil {
		t.Fatalf("routing.New: %v", err)
	}

	rec := httptest.NewRecorder()
	start := time.Now()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/_platform/identity", nil))
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

func TestRejectsAnUnparseableUpstream(t *testing.T) {
	if _, err := routing.New(map[string]string{"identity": "://bad"}, time.Second); err == nil {
		t.Fatal("a malformed upstream URL must be rejected at construction")
	}
}

// The upstream echoes X-Request-Id back (as every service's own requestid
// middleware does); the gateway's requestid.Middleware also sets it before the
// proxy runs. Only one must survive onto the client response.
func TestProxiedResponseHasExactlyOneRequestIDHeader(t *testing.T) {
	var seenByUpstream string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenByUpstream = r.Header.Get(requestid.Header)
		w.Header().Set(requestid.Header, seenByUpstream)
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"service": "identity"})
	}))
	defer upstream.Close()

	h, err := routing.New(map[string]string{"identity": upstream.URL}, time.Second)
	if err != nil {
		t.Fatalf("routing.New: %v", err)
	}
	wrapped := requestid.Middleware(h)

	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/_platform/identity", nil))

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
