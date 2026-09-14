package auth_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/loloDawit/go-admin/platform/observability"
	"github.com/loloDawit/go-admin/platform/principal"
	"github.com/loloDawit/go-admin/services/gateway/internal/auth"
	"github.com/loloDawit/go-admin/services/gateway/internal/httperr"
)

const testKey = "test-signing-key-at-least-32-bytes-long"

func newValidator(t *testing.T, identityURL string, principalTTL, cacheTTL time.Duration) *auth.Validator {
	t.Helper()
	logger, _ := observability.NewCaptured()
	return auth.NewValidator(
		http.DefaultClient,
		identityURL+"/internal/sessions/validate",
		[]byte(testKey),
		principalTTL,
		auth.NewCache(cacheTTL),
		httperr.New(logger),
		logger,
	)
}

func fakeIdentity(t *testing.T, onCall func()) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		onCall()
		var req struct {
			Token string `json:"token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("identity stub: decode request: %v", err)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"staffId":            "staff-1",
			"permissions":        []string{"orders_view"},
			"mustChangePassword": false,
		})
	}))
}

func TestAClientSuppliedPrincipalIsStripped(t *testing.T) {
	identity := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("identity must not be called for a request with no session cookie")
	}))
	defer identity.Close()

	v := newValidator(t, identity.URL, time.Minute, time.Minute)

	var gotHeader, gotSig string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get(principal.HeaderPrincipal)
		gotSig = r.Header.Get(principal.HeaderSignature)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	req.Header.Set(principal.HeaderPrincipal, "forged-principal")
	req.Header.Set(principal.HeaderSignature, "forged-signature")

	auth.Middleware(v)(next).ServeHTTP(httptest.NewRecorder(), req)

	if gotHeader == "forged-principal" {
		t.Fatal("the gateway forwarded a client-supplied principal")
	}
	if gotSig == "forged-signature" {
		t.Fatal("the gateway forwarded a client-supplied signature")
	}
}

func TestAValidatedSessionIsCachedForTheTTL(t *testing.T) {
	var calls int32
	identity := fakeIdentity(t, func() { atomic.AddInt32(&calls, 1) })
	defer identity.Close()

	v := newValidator(t, identity.URL, time.Minute, time.Minute)

	var lastHeader string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastHeader = r.Header.Get(principal.HeaderPrincipal)
	})
	h := auth.Middleware(v)(next)

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
		req.AddCookie(&http.Cookie{Name: "session", Value: "tok-1"})
		h.ServeHTTP(httptest.NewRecorder(), req)
	}

	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("identity calls: want 1 (second request served from cache), got %d", got)
	}
	if lastHeader == "" {
		t.Fatal("want a signed principal header forwarded")
	}
}

func TestACacheHitMintsAFreshPrincipalNotAStaleOne(t *testing.T) {
	identity := fakeIdentity(t, func() {})
	defer identity.Close()

	v := newValidator(t, identity.URL, time.Minute, time.Minute)

	first, err := v.Resolve(t.Context(), "tok-1")
	if err != nil {
		t.Fatalf("Resolve (miss): %v", err)
	}
	time.Sleep(10 * time.Millisecond)
	second, err := v.Resolve(t.Context(), "tok-1")
	if err != nil {
		t.Fatalf("Resolve (hit): %v", err)
	}

	if !second.ExpiresAt.After(first.ExpiresAt) {
		t.Fatalf("cache hit returned a stale principal: first ExpiresAt=%v, second ExpiresAt=%v", first.ExpiresAt, second.ExpiresAt)
	}
}

func TestTheCacheExpires(t *testing.T) {
	var calls int32
	identity := fakeIdentity(t, func() { atomic.AddInt32(&calls, 1) })
	defer identity.Close()

	const cacheTTL = 20 * time.Millisecond
	v := newValidator(t, identity.URL, time.Minute, cacheTTL)
	h := auth.Middleware(v)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))

	req := func() *http.Request {
		r := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
		r.AddCookie(&http.Cookie{Name: "session", Value: "tok-1"})
		return r
	}

	h.ServeHTTP(httptest.NewRecorder(), req())
	time.Sleep(cacheTTL * 3)
	h.ServeHTTP(httptest.NewRecorder(), req())

	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("identity calls: want 2 (cache expired between requests), got %d", got)
	}
}

func TestAMissingCookieIsNotAuthenticated(t *testing.T) {
	identity := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("identity must not be called for a request with no session cookie")
	}))
	defer identity.Close()

	v := newValidator(t, identity.URL, time.Minute, time.Minute)

	var gotHeader string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get(principal.HeaderPrincipal)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil) // no cookie set
	auth.Middleware(v)(next).ServeHTTP(httptest.NewRecorder(), req)

	if gotHeader != "" {
		t.Fatalf("want no principal forwarded without a cookie, got %q", gotHeader)
	}
}

func TestARejectedSessionIsNotAuthenticated(t *testing.T) {
	identity := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer identity.Close()

	v := newValidator(t, identity.URL, time.Minute, time.Minute)

	var gotHeader string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get(principal.HeaderPrincipal)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "revoked-token"})
	auth.Middleware(v)(next).ServeHTTP(httptest.NewRecorder(), req)

	if gotHeader != "" {
		t.Fatalf("want no principal forwarded for a rejected session, got %q", gotHeader)
	}
}
