package httpx_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/loloDawit/go-admin/platform/httpx"
)

type decodeTarget struct {
	Email string `json:"email"`
}

// testMaxBytes stands in for a caller's configured limit; its value is
// arbitrary except for TestDecodeJSONRefusesABodyOverTheCap, which needs a
// body larger than it.
const testMaxBytes = 1 << 20 // 1 MiB

func TestDecodeJSONRejectsAnUnknownField(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"email":"a@example.com","role_id":1}`))

	var v decodeTarget
	err := httpx.DecodeJSON(req, &v, testMaxBytes)
	if !errors.Is(err, httpx.ErrMalformedBody) {
		t.Fatalf("want ErrMalformedBody, got %v", err)
	}
}

func TestDecodeJSONAcceptsAKnownBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"email":"a@example.com"}`))

	var v decodeTarget
	if err := httpx.DecodeJSON(req, &v, testMaxBytes); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if v.Email != "a@example.com" {
		t.Errorf("email: got %q", v.Email)
	}
}

func TestDecodeJSONRefusesABodyOverTheCap(t *testing.T) {
	huge := `{"email":"` + strings.Repeat("a", 2*testMaxBytes) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(huge))

	var v decodeTarget
	err := httpx.DecodeJSON(req, &v, testMaxBytes)
	if !errors.Is(err, httpx.ErrMalformedBody) {
		t.Fatalf("want ErrMalformedBody for an oversized body, got %v", err)
	}
}

func TestDecodeJSONRejectsMalformedJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{not-json`))

	var v decodeTarget
	err := httpx.DecodeJSON(req, &v, testMaxBytes)
	if !errors.Is(err, httpx.ErrMalformedBody) {
		t.Fatalf("want ErrMalformedBody, got %v", err)
	}
}
