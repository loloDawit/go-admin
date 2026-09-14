package principal_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/loloDawit/go-admin/platform/principal"
)

func TestMiddlewareRejectsARequestWithNoHeaders(t *testing.T) {
	called := false
	var gotErr error
	h := principal.Middleware(key, func(w http.ResponseWriter, r *http.Request, err error) {
		called = true
		gotErr = err
		w.WriteHeader(http.StatusUnauthorized)
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next must not run without a verified principal")
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if !called {
		t.Fatal("onErr was not called")
	}
	if !errors.Is(gotErr, principal.ErrMissing) {
		t.Errorf("want ErrMissing, got %v", gotErr)
	}
}

func TestMiddlewarePassesTheVerifiedPrincipalToNext(t *testing.T) {
	now := time.Now()
	p := testPrincipal(now)
	h, sig, err := principal.Sign(p, key)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	var seen principal.Principal
	var ok bool
	handler := principal.Middleware(key, func(w http.ResponseWriter, r *http.Request, err error) {
		t.Fatalf("onErr must not run for a valid principal: %v", err)
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen, ok = principal.FromContext(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(principal.HeaderPrincipal, h)
	req.Header.Set(principal.HeaderSignature, sig)

	handler.ServeHTTP(httptest.NewRecorder(), req)

	if !ok {
		t.Fatal("FromContext: want ok=true")
	}
	if seen.StaffID != p.StaffID {
		t.Errorf("StaffID: want %q, got %q", p.StaffID, seen.StaffID)
	}
}

func TestFromContextIsEmptyWithoutMiddleware(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if _, ok := principal.FromContext(req.Context()); ok {
		t.Error("want ok=false without middleware")
	}
}
