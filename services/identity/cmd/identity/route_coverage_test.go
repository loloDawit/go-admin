package main

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/go-chi/chi/v5"
)

// publicRoutes is the identity service's entire declared-unauthenticated
// surface; adding to it is a deliberate, reviewed edit. POST
// /internal/sessions/validate is network-isolated rather than
// gateway-routed — the gateway never proxies /internal/*.
var publicRoutes = map[string]bool{
	"POST /api/v1/login":               true,
	"GET /healthz":                     true,
	"GET /readyz":                      true,
	"GET /_platform":                   true,
	"POST /internal/sessions/validate": true,
}

var routeParam = regexp.MustCompile(`\{[^}]+\}`)

// TestEveryRouteIsAuthenticatedOrDeclaredPublic walks the live route table
// and sends each non-public route a request carrying no principal header. A
// route that answers 200 is neither declared public nor actually protected:
// a middleware-slice comparison could pass on a route enforcing nothing, so
// this drives a real request instead.
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

		if rec.Code == http.StatusOK {
			t.Errorf("%s answered 200 with no principal: not declared public and not actually protected", key)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("chi.Walk: %v", err)
	}
}
