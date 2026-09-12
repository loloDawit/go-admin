package observability_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/loloDawit/go-admin/platform/observability"
	"github.com/loloDawit/go-admin/platform/requestid"
)

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
