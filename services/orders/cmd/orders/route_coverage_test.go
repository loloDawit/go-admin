package main

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/loloDawit/go-admin/platform/observability"
	"github.com/loloDawit/go-admin/platform/principal"
	"github.com/loloDawit/go-admin/platform/readiness"
	"github.com/loloDawit/go-admin/services/orders/internal/httperr"
	"github.com/loloDawit/go-admin/services/orders/internal/platformcheck"
)

// publicRoutes is the entire declared-unauthenticated surface; adding to it is a deliberate, reviewed edit.
var publicRoutes = map[string]bool{
	"GET /healthz":   true,
	"GET /readyz":    true,
	"GET /_platform": true,
}

var routeParam = regexp.MustCompile(`\{[^}]+\}`)

func testRouter(t *testing.T) *chi.Mux {
	t.Helper()
	logger, _ := observability.NewCaptured()
	errWriter := httperr.New(logger)
	svc := platformcheck.NewService(&alwaysFailRepo{})
	handler := platformcheck.NewHandler(svc, "orders", errWriter.Write)
	ready := readiness.NewHandler(svc.Probe, errWriter.Write)

	return newRouter(logger, handler, ready, newTestCustomerHandler(), newTestOrderHandler(), testPrincipalKey, errWriter.Write)
}

// This drives a real request rather than comparing middleware slices, which could pass on a route enforcing nothing.
func TestEveryRouteIsAuthenticatedOrDeclaredPublic(t *testing.T) {
	r := testRouter(t)

	err := chi.Walk(r, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		key := method + " " + route
		if publicRoutes[key] {
			return nil
		}

		path := routeParam.ReplaceAllString(route, "1")
		req := httptest.NewRequest(method, path, nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s answered %d with no principal, want 401: not declared public and not actually protected", key, rec.Code)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("chi.Walk: %v", err)
	}
}

// A signed principal with no permissions must still be refused: 401 alone
// only proves principal.Middleware runs, not that authz.Require does too.
func TestEveryProtectedRouteRefusesAPrincipalWithNoPermissions(t *testing.T) {
	r := testRouter(t)

	p := principal.Principal{StaffID: "1", IssuedAt: time.Now(), ExpiresAt: time.Now().Add(time.Minute)}
	header, sig, err := principal.Sign(p, testPrincipalKey)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	walkErr := chi.Walk(r, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		key := method + " " + route
		if publicRoutes[key] {
			return nil
		}

		path := routeParam.ReplaceAllString(route, "1")
		req := httptest.NewRequest(method, path, nil)
		req.Header.Set(principal.HeaderPrincipal, header)
		req.Header.Set(principal.HeaderSignature, sig)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("%s answered %d for a principal with no permissions, want 403: authz.Require may be missing from this route", key, rec.Code)
		}
		return nil
	})
	if walkErr != nil {
		t.Fatalf("chi.Walk: %v", walkErr)
	}
}
