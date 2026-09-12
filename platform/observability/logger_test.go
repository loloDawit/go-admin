package observability_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/loloDawit/go-admin/platform/observability"
	"github.com/loloDawit/go-admin/platform/requestid"
)

func TestServiceAttrSurvivesWith(t *testing.T) {
	logger, captured := observability.NewCaptured()
	logger = logger.With(slog.String("service", "catalog"))

	h := observability.RequestLogger(logger)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	v, ok := captured.Attr(0, "service")
	if !ok || v.String() != "catalog" {
		t.Errorf("service: want %q, got %v (ok=%v)", "catalog", v, ok)
	}
}

func TestRequestLoggerRecordsTheExpectedAttributes(t *testing.T) {
	logger, captured := observability.NewCaptured()

	h := requestid.Middleware(
		observability.RequestLogger(logger)(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusTeapot)
			}),
		),
	)

	req := httptest.NewRequest(http.MethodGet, "/_platform", nil)
	req.Header.Set(requestid.Header, "known-id")
	h.ServeHTTP(httptest.NewRecorder(), req)

	records := captured.Records()
	if len(records) != 1 {
		t.Fatalf("want exactly one log record, got %d", len(records))
	}

	for _, tc := range []struct{ key, want string }{
		{"request_id", "known-id"},
		{"route", "/_platform"},
		{"method", "GET"},
	} {
		v, ok := captured.Attr(0, tc.key)
		if !ok {
			t.Errorf("attribute %q missing", tc.key)
			continue
		}
		if v.String() != tc.want {
			t.Errorf("%s: want %q, got %q", tc.key, tc.want, v.String())
		}
	}

	if v, ok := captured.Attr(0, "status"); !ok || v.Int64() != http.StatusTeapot {
		t.Errorf("status: want 418, got %v", v)
	}
	if _, ok := captured.Attr(0, "duration_ms"); !ok {
		t.Error("duration_ms missing")
	}
}

func TestRequestLoggerDefaultsStatusTo200WhenWriteHeaderIsNeverCalled(t *testing.T) {
	logger, captured := observability.NewCaptured()

	h := observability.RequestLogger(logger)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("ok"))
		}),
	)

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	v, ok := captured.Attr(0, "status")
	if !ok || v.Int64() != http.StatusOK {
		t.Errorf("status: want 200, got %v (ok=%v)", v, ok)
	}
}

func TestRequestLoggerLogs200WhenTheHandlerWritesNothing(t *testing.T) {
	logger, captured := observability.NewCaptured()
	h := observability.RequestLogger(logger)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	v, ok := captured.Attr(0, "status")
	if !ok || v.Int64() != http.StatusOK {
		t.Fatalf("a handler that writes nothing must log 200, got %v (ok=%v)", v, ok)
	}
}

func TestRequestLoggerRecordsOnlyTheFirstWriteHeaderCall(t *testing.T) {
	logger, captured := observability.NewCaptured()

	h := observability.RequestLogger(logger)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.WriteHeader(http.StatusInternalServerError)
		}),
	)

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	v, ok := captured.Attr(0, "status")
	if !ok || v.Int64() != http.StatusOK {
		t.Errorf("status: want 200 (the first call net/http honors), got %v (ok=%v)", v, ok)
	}
}
