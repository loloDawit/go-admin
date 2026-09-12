package httperr_test

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/loloDawit/go-admin/platform/httpx"
	"github.com/loloDawit/go-admin/services/catalog/internal/httperr"
	"github.com/loloDawit/go-admin/services/catalog/internal/platformcheck"
)

func TestWriteMapsDirtySchemaTo503(t *testing.T) {
	rec := httptest.NewRecorder()
	httperr.Write(rec, platformcheck.ErrDirtySchema)

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
	rec := httptest.NewRecorder()
	httperr.Write(rec, platformcheck.ErrNoMigrations)

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

	rec := httptest.NewRecorder()
	httperr.Write(rec, cause)

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
