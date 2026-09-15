package httperr_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/loloDawit/go-admin/platform/httpx"
	"github.com/loloDawit/go-admin/platform/observability"
	"github.com/loloDawit/go-admin/platform/requestid"
	"github.com/loloDawit/go-admin/services/orders/internal/customer"
	"github.com/loloDawit/go-admin/services/orders/internal/errs"
	"github.com/loloDawit/go-admin/services/orders/internal/httperr"
	"github.com/loloDawit/go-admin/services/orders/internal/order"
	"github.com/loloDawit/go-admin/services/orders/internal/platformcheck"
)

func TestWriteMapsMalformedBodyTo400(t *testing.T) {
	logger, _ := observability.NewCaptured()
	rec := httptest.NewRecorder()
	httperr.New(logger).Write(t.Context(), rec, httpx.ErrMalformedBody)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: want 400, got %d", rec.Code)
	}
	var body httpx.ErrorBody
	if err := json.NewDecoder(strings.NewReader(rec.Body.String())).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Code != "malformed_body" {
		t.Errorf("code: want malformed_body, got %q", body.Code)
	}
}

func TestWriteMapsCustomerNotFoundTo404(t *testing.T) {
	logger, _ := observability.NewCaptured()
	rec := httptest.NewRecorder()
	httperr.New(logger).Write(t.Context(), rec, customer.ErrCustomerNotFound)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status: want 404, got %d", rec.Code)
	}
	var body httpx.ErrorBody
	if err := json.NewDecoder(strings.NewReader(rec.Body.String())).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Code != "not_found" {
		t.Errorf("code: want not_found, got %q", body.Code)
	}
}

func TestWriteMapsCustomerEmailTakenTo409(t *testing.T) {
	logger, _ := observability.NewCaptured()
	rec := httptest.NewRecorder()
	httperr.New(logger).Write(t.Context(), rec, customer.ErrCustomerEmailTaken)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status: want 409, got %d", rec.Code)
	}
	var body httpx.ErrorBody
	if err := json.NewDecoder(strings.NewReader(rec.Body.String())).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Code != "email_taken" {
		t.Errorf("code: want email_taken, got %q", body.Code)
	}
}

func TestWriteMapsInvalidCustomerEmailTo400(t *testing.T) {
	logger, _ := observability.NewCaptured()
	rec := httptest.NewRecorder()
	httperr.New(logger).Write(t.Context(), rec, customer.ErrInvalidCustomerEmail)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: want 400, got %d", rec.Code)
	}
	var body httpx.ErrorBody
	if err := json.NewDecoder(strings.NewReader(rec.Body.String())).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Code != "validation_failed" {
		t.Errorf("code: want validation_failed, got %q", body.Code)
	}
}

func TestWriteMapsOrderNotFoundTo404(t *testing.T) {
	logger, _ := observability.NewCaptured()
	rec := httptest.NewRecorder()
	httperr.New(logger).Write(t.Context(), rec, order.ErrOrderNotFound)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status: want 404, got %d", rec.Code)
	}
}

func TestWriteMapsProductUnavailableTo422(t *testing.T) {
	logger, _ := observability.NewCaptured()
	rec := httptest.NewRecorder()
	httperr.New(logger).Write(t.Context(), rec, order.ErrProductUnavailable)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status: want 422, got %d", rec.Code)
	}
}

func TestWriteMapsCurrencyMismatchTo422(t *testing.T) {
	logger, _ := observability.NewCaptured()
	rec := httptest.NewRecorder()
	httperr.New(logger).Write(t.Context(), rec, order.ErrCurrencyMismatch)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status: want 422, got %d", rec.Code)
	}
}

// A wrapped Catalog failure must still map to 502, never fall through to the
// unmapped 500 branch, and the response must never carry Catalog's host or
// driver text. §16: Catalog unreachable, or answering with a 5xx, is 502
// catalog_unavailable.
func TestWriteMapsWrappedCatalogUnavailableTo502WithNoLeak(t *testing.T) {
	logger, _ := observability.NewCaptured()
	rec := httptest.NewRecorder()
	cause := errs.Wrap(errs.OpCreateOrder, errs.ErrCatalogUnavailable)

	httperr.New(logger).Write(t.Context(), rec, cause)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status: want 502, got %d", rec.Code)
	}
	var body httpx.ErrorBody
	if err := json.NewDecoder(strings.NewReader(rec.Body.String())).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Code != "catalog_unavailable" {
		t.Errorf("code: want catalog_unavailable, got %q", body.Code)
	}
	raw := rec.Body.String()
	if strings.Contains(raw, "http://") || strings.Contains(raw, "connection refused") {
		t.Errorf("response leaked catalog transport detail: %q", raw)
	}
}

// §16: a Catalog timeout is distinct from an unreachable Catalog — 504
// catalog_timeout, not 502 — so a caller can tell "struggling" from "down".
func TestWriteMapsCatalogTimeoutTo504(t *testing.T) {
	logger, _ := observability.NewCaptured()
	rec := httptest.NewRecorder()
	cause := errs.Wrap(errs.OpCreateOrder, errs.ErrCatalogTimeout)

	httperr.New(logger).Write(t.Context(), rec, cause)

	if rec.Code != http.StatusGatewayTimeout {
		t.Fatalf("status: want 504, got %d", rec.Code)
	}
	var body httpx.ErrorBody
	if err := json.NewDecoder(strings.NewReader(rec.Body.String())).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Code != "catalog_timeout" {
		t.Errorf("code: want catalog_timeout, got %q", body.Code)
	}
}

// A cancelled caller is not a failed dependency: it must not join the 5xx
// error-rate metric a real Catalog outage would trip, and it must not be
// logged as an error either.
func TestWriteMapsACancelledContextToANonFiveXXWithNoErrorLog(t *testing.T) {
	logger, captured := observability.NewCaptured()
	rec := httptest.NewRecorder()
	cause := errs.Wrap(errs.OpCreateOrder, context.Canceled)

	httperr.New(logger).Write(t.Context(), rec, cause)

	if rec.Code >= 500 {
		t.Fatalf("status: want a non-5xx, got %d", rec.Code)
	}
	for _, rec := range captured.Records() {
		if rec.Level.String() == "ERROR" {
			t.Errorf("a cancelled request must not be logged as an error: %v", rec.Message)
		}
	}
}

func TestWriteMapsInvalidTransitionTo409(t *testing.T) {
	logger, _ := observability.NewCaptured()
	rec := httptest.NewRecorder()
	httperr.New(logger).Write(t.Context(), rec, order.ErrInvalidTransition)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status: want 409, got %d", rec.Code)
	}
	var body httpx.ErrorBody
	if err := json.NewDecoder(strings.NewReader(rec.Body.String())).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Code != "invalid_transition" {
		t.Errorf("code: want invalid_transition, got %q", body.Code)
	}
}

func TestWriteMapsInvalidSortTo422(t *testing.T) {
	logger, _ := observability.NewCaptured()
	rec := httptest.NewRecorder()
	httperr.New(logger).Write(t.Context(), rec, order.ErrInvalidSort)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status: want 422, got %d", rec.Code)
	}
}

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

func TestUnmappedErrorIsInternalNotUnavailable(t *testing.T) {
	logger, _ := observability.NewCaptured()
	rec := httptest.NewRecorder()

	httperr.New(logger).Write(t.Context(), rec, errors.New("wrong password"))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status: want 500, got %d", rec.Code)
	}
	var body httpx.ErrorBody
	if err := json.NewDecoder(strings.NewReader(rec.Body.String())).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Code != "internal" {
		t.Errorf("code: want internal, got %q", body.Code)
	}
	if strings.Contains(body.Message, "wrong password") {
		t.Errorf("cause leaked to the client: %q", body.Message)
	}
}

// The default branch is where an unmapped, driver-level error would otherwise
// leak connection details to a client; this pins that it never does.
func TestWriteNeverLeaksTheCauseOfAnUnmappedError(t *testing.T) {
	cause := errors.New("dial tcp 10.0.0.5:5432: connect: connection refused")

	logger, _ := observability.NewCaptured()
	rec := httptest.NewRecorder()
	httperr.New(logger).Write(t.Context(), rec, cause)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status: want 500, got %d", rec.Code)
	}
	raw := rec.Body.String()

	var body httpx.ErrorBody
	if err := json.NewDecoder(strings.NewReader(raw)).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Code != "internal" {
		t.Errorf("code: want internal, got %q", body.Code)
	}
	// Checked against the whole raw response, not just body.Message: a leak
	// through any future envelope field must be caught too.
	if strings.Contains(raw, "10.0.0.5") || strings.Contains(raw, "connection refused") {
		t.Errorf("response leaked the cause: %q", raw)
	}
}

// An unmapped error's log line must carry the same request_id as RequestLogger's request line, or the two cannot be correlated.
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
