package authz_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/loloDawit/go-admin/platform/principal"
	"github.com/loloDawit/go-admin/services/identity/internal/authz"
	"github.com/loloDawit/go-admin/services/identity/internal/errs"
	"github.com/loloDawit/go-admin/services/identity/internal/permission"
)

var testKey = []byte("a-test-signing-key-at-least-32-bytes-long")

func signedRequest(t *testing.T, p principal.Principal) *http.Request {
	t.Helper()
	p.IssuedAt = time.Now()
	p.ExpiresAt = time.Now().Add(time.Minute)
	header, sig, err := principal.Sign(p, testKey)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(principal.HeaderPrincipal, header)
	req.Header.Set(principal.HeaderSignature, sig)
	return req
}

// chainThroughPrincipal wires principal.Middleware in front of authz.Require exactly as the router does.
func chainThroughPrincipal(onErr func(context.Context, http.ResponseWriter, error), p permission.Permission, next http.Handler) http.Handler {
	return principal.Middleware(testKey, func(w http.ResponseWriter, r *http.Request, err error) {
		onErr(r.Context(), w, err)
	})(authz.Require(p, onErr)(next))
}

func TestRequireRefusesAPrincipalWithoutThePermission(t *testing.T) {
	req := signedRequest(t, principal.Principal{StaffID: "1", Permissions: []string{string(permission.ViewOrders)}})

	var captured error
	onErr := func(_ context.Context, w http.ResponseWriter, err error) {
		captured = err
		w.WriteHeader(http.StatusForbidden)
	}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next must not run without the required permission")
	})
	rec := httptest.NewRecorder()
	chainThroughPrincipal(onErr, permission.ViewStaff, next).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status: want 403, got %d", rec.Code)
	}
	if !errors.Is(captured, errs.ErrForbidden) {
		t.Fatalf("want errs.ErrForbidden, got %v", captured)
	}
}

func TestRequireRefusesAnAbsentPrincipal(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	var captured error
	onErr := func(_ context.Context, w http.ResponseWriter, err error) {
		captured = err
		w.WriteHeader(http.StatusUnauthorized)
	}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next must not run without a principal in context")
	})
	rec := httptest.NewRecorder()
	authz.Require(permission.ViewStaff, onErr)(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status: want 401, got %d", rec.Code)
	}
	if !errors.Is(captured, errs.ErrUnauthenticated) {
		t.Fatalf("want errs.ErrUnauthenticated, got %v", captured)
	}
}

func TestRequireAcceptsThePermission(t *testing.T) {
	req := signedRequest(t, principal.Principal{StaffID: "1", Permissions: []string{string(permission.ViewStaff)}})

	ran := false
	onErr := func(_ context.Context, w http.ResponseWriter, err error) {
		t.Fatalf("onErr must not run for a principal carrying the permission: %v", err)
	}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { ran = true; w.WriteHeader(http.StatusOK) })
	rec := httptest.NewRecorder()
	chainThroughPrincipal(onErr, permission.ViewStaff, next).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK || !ran {
		t.Fatalf("status: want 200 and next run, got %d, ran=%v", rec.Code, ran)
	}
}
