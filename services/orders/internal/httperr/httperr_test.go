package httperr_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/loloDawit/go-admin/platform/httpx"
	"github.com/loloDawit/go-admin/platform/observability"
	"github.com/loloDawit/go-admin/platform/requestid"
	"github.com/loloDawit/go-admin/services/orders/internal/httperr"
	"github.com/loloDawit/go-admin/services/orders/internal/platformcheck"
)

func TestWriteMapsDirtySchemaTo503(t *testing.T) {
	logger, _ := observability.NewCaptured()
	rec := httptest.NewRecorder()
	httperr.New(logger).Write(t.Context(), rec, platformcheck.ErrDirtySchema)

	if rec.Code != 503 {
		t.Fatalf("status: want 503, got %d", rec.Code)
	}
	var body httpx.ErrorBody
	if err := json.NewDecoder(strings.NewReader(rec.Body.String())).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Code != "schema_dirty" {
		t.Errorf("code: want schema_dirty, got %q", body.Code)
	}
}

func TestWriteMapsNoMigrationsTo503(t *testing.T) {
	logger, _ := observability.NewCaptured()
	rec := httptest.NewRecorder()
	httperr.New(logger).Write(t.Context(), rec, platformcheck.ErrNoMigrations)

	if rec.Code != 503 {
		t.Fatalf("status: want 503, got %d", rec.Code)
	}
	var body httpx.ErrorBody
	if err := json.NewDecoder(strings.NewReader(rec.Body.String())).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Code != "schema_not_migrated" {
		t.Errorf("code: want schema_not_migrated, got %q", body.Code)
	}
}

// The default branch is where an unmapped, driver-level error would otherwise
// leak connection details to a client; this pins that it never does.
func TestWriteNeverLeaksTheCauseOfAnUnmappedError(t *testing.T) {
	cause := errors.New("dial tcp 10.0.0.5:5432: connect: connection refused")

	logger, _ := observability.NewCaptured()
	rec := httptest.NewRecorder()
	httperr.New(logger).Write(t.Context(), rec, cause)

	if rec.Code != 503 {
		t.Fatalf("status: want 503, got %d", rec.Code)
	}
	raw := rec.Body.String()

	var body httpx.ErrorBody
	if err := json.NewDecoder(strings.NewReader(raw)).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Code != "database_unavailable" {
		t.Errorf("code: want database_unavailable, got %q", body.Code)
	}
	// Checked against the whole raw response, not just body.Message: a leak
	// through any future envelope field must be caught too.
	if strings.Contains(raw, "10.0.0.5") || strings.Contains(raw, "connection refused") {
		t.Errorf("response leaked the cause: %q", raw)
	}
}

// This pins spec §10's request-ID groundwork: an unmapped error's log line
// must carry the same request_id as the request line RequestLogger emits for
// the same request, or the two cannot be correlated during an outage.
func TestWriteLogsTheCauseCorrelatedByRequestID(t *testing.T) {
	logger, captured := observability.NewCaptured()
	writer := httperr.New(logger)
	cause := errors.New("dial tcp 10.0.0.5:5432: connect: connection refused")

	h := requestid.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writer.Write(r.Context(), w, cause)
	}))

	req := httptest.NewRequest(http.MethodGet, "/_platform", nil)
	req.Header.Set(requestid.Header, "known-id")
	h.ServeHTTP(httptest.NewRecorder(), req)

	records := captured.Records()
	if len(records) != 1 {
		t.Fatalf("want exactly one log record, got %d", len(records))
	}
	v, ok := captured.Attr(0, "request_id")
	if !ok || v.String() != "known-id" {
		t.Errorf("request_id: want %q, got %v (ok=%v)", "known-id", v, ok)
	}
}
