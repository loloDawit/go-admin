package httpx_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/loloDawit/go-admin/platform/httpx"
)

func TestWriteErrorUsesTheEnvelope(t *testing.T) {
	rec := httptest.NewRecorder()

	httpx.WriteError(rec, http.StatusServiceUnavailable, "database_unavailable", "the service is not ready")

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status: want 503, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("content-type: want application/json, got %q", ct)
	}

	var body httpx.ErrorBody
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Code != "database_unavailable" {
		t.Errorf("code: got %q", body.Code)
	}
	if body.Message != "the service is not ready" {
		t.Errorf("message: got %q", body.Message)
	}
}

func TestWriteJSONSetsStatusAndBody(t *testing.T) {
	rec := httptest.NewRecorder()

	httpx.WriteJSON(rec, http.StatusOK, map[string]string{"service": "identity"})

	if rec.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", rec.Code)
	}
	var body map[string]string
	json.NewDecoder(rec.Body).Decode(&body)
	if body["service"] != "identity" {
		t.Errorf("body: got %v", body)
	}
}
