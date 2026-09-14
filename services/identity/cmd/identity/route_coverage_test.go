package main

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/go-chi/chi/v5"
)

// publicRoutes is the entire declared-unauthenticated surface; adding to it is a deliberate, reviewed edit.
var publicRoutes = map[string]bool{
	"POST /api/v1/login":               true,
	"GET /healthz":                     true,
	"GET /readyz":                      true,
	"POST /internal/sessions/validate": true, // called by the gateway to resolve a cookie; the caller cannot itself hold a session
}

var routeParam = regexp.MustCompile(`\{[^}]+\}`)

// This drives a real request rather than comparing middleware slices, which could pass on a route enforcing nothing.
func TestEveryRouteIsAuthenticatedOrDeclaredPublic(t *testing.T) {
	r, _ := newRouterWithLogin(t)

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
