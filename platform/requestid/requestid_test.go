package requestid_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/loloDawit/go-admin/platform/requestid"
)

func TestMiddlewarePreservesAClientSuppliedID(t *testing.T) {
	var seen string
	h := requestid.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = requestid.FromContext(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(requestid.Header, "client-supplied-id")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if seen != "client-supplied-id" {
		t.Errorf("context: want client-supplied-id, got %q", seen)
	}
	if echoed := rec.Header().Get(requestid.Header); echoed != "client-supplied-id" {
		t.Errorf("response header: want client-supplied-id, got %q", echoed)
	}
}

func TestMiddlewareGeneratesAnIDWhenAbsent(t *testing.T) {
	var seen string
	h := requestid.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = requestid.FromContext(r.Context())
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if seen == "" {
		t.Fatal("an ID must be generated when the client sends none")
	}
	if rec.Header().Get(requestid.Header) != seen {
		t.Error("the generated ID must be echoed in the response header")
	}
}

func TestFromContextIsEmptyWithoutMiddleware(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if got := requestid.FromContext(req.Context()); got != "" {
		t.Errorf("want empty, got %q", got)
	}
}
