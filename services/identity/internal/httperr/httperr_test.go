package httperr_test

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/loloDawit/go-admin/platform/httpx"
	"github.com/loloDawit/go-admin/services/identity/internal/httperr"
	"github.com/loloDawit/go-admin/services/identity/internal/platformcheck"
)

func TestWriteMapsDirtySchemaTo503(t *testing.T) {
	rec := httptest.NewRecorder()
	httperr.Write(rec, platformcheck.ErrDirtySchema)

	if rec.Code != 503 {
		t.Fatalf("status: want 503, got %d", rec.Code)
	}
	var body httpx.ErrorBody
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Code != "schema_dirty" {
		t.Errorf("code: want schema_dirty, got %q", body.Code)
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
	var body httpx.ErrorBody
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Code != "database_unavailable" {
		t.Errorf("code: want database_unavailable, got %q", body.Code)
	}
	if strings.Contains(body.Message, "10.0.0.5") || strings.Contains(body.Message, "connection refused") {
		t.Errorf("message leaked the cause: %q", body.Message)
	}
}
