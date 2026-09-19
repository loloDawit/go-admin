package observability

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func loggedRequest(t *testing.T, target string, register func(*chi.Mux)) map[string]any {
	t.Helper()
	var buf bytes.Buffer
	logger := NewLogger("test", &buf)

	r := chi.NewRouter()
	r.Use(RequestLogger(logger))
	r.Use(RouteTagger())
	register(r)

	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, target, nil))

	var record map[string]any
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatalf("log record is not JSON: %v", err)
	}
	return record
}

// A concrete id in the route field is one log stream, and one metric time
// series, per order.
func TestRequestLoggerLogsTheRoutePattern(t *testing.T) {
	record := loggedRequest(t, "/api/v1/orders/8231", func(r *chi.Mux) {
		r.Get("/api/v1/orders/{id}", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	})

	if record["route"] != "/api/v1/orders/{id}" {
		t.Fatalf("route = %v, want the pattern", record["route"])
	}
	if record["path"] != "/api/v1/orders/8231" {
		t.Fatalf("path = %v, want the concrete path", record["path"])
	}
}

// An unmatched request has no pattern; the field must be absent rather than
// falling back to the path it was introduced to avoid.
func TestUnmatchedRequestHasNoRoute(t *testing.T) {
	record := loggedRequest(t, "/nothing/here", func(r *chi.Mux) {
		r.Get("/api/v1/orders", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	})

	if got, ok := record["route"]; ok {
		t.Fatalf("route = %v, want absent", got)
	}
}
