package session_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/loloDawit/go-admin/platform/principal"
	"github.com/loloDawit/go-admin/services/identity/internal/session"
)

var handlerTestKey = []byte("a-test-signing-key-at-least-32-bytes-long")

// signedMeRequest runs through principal.Middleware exactly as the router wires it.
func signedMeRequest(t *testing.T, h *session.Handler, staffID string) *httptest.ResponseRecorder {
	t.Helper()
	p := principal.Principal{
		StaffID:   staffID,
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(time.Minute),
	}
	header, sig, err := principal.Sign(p, handlerTestKey)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	var captured error
	middleware := principal.Middleware(handlerTestKey, func(w http.ResponseWriter, r *http.Request, err error) {
		captured = err
		w.WriteHeader(http.StatusUnauthorized)
	})(http.HandlerFunc(h.Me))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	req.Header.Set(principal.HeaderPrincipal, header)
	req.Header.Set(principal.HeaderSignature, sig)
	rec := httptest.NewRecorder()
	middleware.ServeHTTP(rec, req)

	if captured != nil {
		t.Fatalf("principal middleware rejected a validly signed principal: %v", captured)
	}
	return rec
}

func TestMeHandlerReturnsTheCallersCurrentRecord(t *testing.T) {
	repo := newFakeRepository()
	seedActiveStaff(t, repo, "owner@example.com", "correct-horse-battery-staple")
	svc := newTestService(t, repo)

	var captured error
	writeErr := func(_ context.Context, _ http.ResponseWriter, err error) { captured = err }
	h := session.NewHandler(svc, false, time.Hour, 1<<20, writeErr)

	rec := signedMeRequest(t, h, "1")

	if captured != nil {
		t.Fatalf("unexpected error: %v", captured)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp session.AuthResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Email != "owner@example.com" {
		t.Errorf("email: want %q, got %q", "owner@example.com", resp.Email)
	}
}

func TestMeHandlerRejectsAPrincipalForAnUnknownStaffID(t *testing.T) {
	repo := newFakeRepository()
	svc := newTestService(t, repo)

	var captured error
	writeErr := func(_ context.Context, _ http.ResponseWriter, err error) { captured = err }
	h := session.NewHandler(svc, false, time.Hour, 1<<20, writeErr)

	signedMeRequest(t, h, "999")

	if captured == nil {
		t.Fatal("want an error for an unknown staff ID, got none")
	}
}

func TestLoginHandlerSetsACookieAndReturnsTheAuthenticatedBody(t *testing.T) {
	repo := newFakeRepository()
	seedActiveStaff(t, repo, "owner@example.com", "correct-horse-battery-staple")
	svc := newTestService(t, repo)

	var captured error
	writeErr := func(_ context.Context, _ http.ResponseWriter, err error) { captured = err }
	h := session.NewHandler(svc, false, time.Hour, 1<<20, writeErr)

	body := `{"email":"owner@example.com","password":"correct-horse-battery-staple"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/login", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	if captured != nil {
		t.Fatalf("unexpected error: %v", captured)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", rec.Code)
	}

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != "session" || cookies[0].Value == "" {
		t.Fatalf("want a non-empty session cookie, got %v", cookies)
	}

	var resp session.AuthResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Email != "owner@example.com" {
		t.Errorf("email: got %q", resp.Email)
	}
}

func TestLoginHandlerRejectsAnUnknownFieldInTheBody(t *testing.T) {
	repo := newFakeRepository()
	seedActiveStaff(t, repo, "owner@example.com", "correct-horse-battery-staple")
	svc := newTestService(t, repo)

	var captured error
	writeErr := func(_ context.Context, w http.ResponseWriter, err error) {
		captured = err
		w.WriteHeader(http.StatusTeapot)
	}
	h := session.NewHandler(svc, false, time.Hour, 1<<20, writeErr)

	body := `{"email":"owner@example.com","password":"x","role_id":1}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/login", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	if captured == nil {
		t.Fatal("want an error for an unknown field, got none")
	}
	if rec.Code != http.StatusTeapot {
		t.Fatalf("want the injected error writer to run, got status %d", rec.Code)
	}
}

func TestLogoutHandlerRevokesTheCookiedSessionAndClearsIt(t *testing.T) {
	repo := newFakeRepository()
	seedActiveStaff(t, repo, "owner@example.com", "correct-horse-battery-staple")
	svc := newTestService(t, repo)
	token, _, err := svc.Login(context.Background(), "owner@example.com", "correct-horse-battery-staple")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	var captured error
	writeErr := func(_ context.Context, _ http.ResponseWriter, err error) { captured = err }
	h := session.NewHandler(svc, false, time.Hour, 1<<20, writeErr)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/logout", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: token})
	rec := httptest.NewRecorder()

	h.Logout(rec, req)

	if captured != nil {
		t.Fatalf("unexpected error: %v", captured)
	}

	if _, err := svc.Validate(context.Background(), token); err == nil {
		t.Fatal("want the session invalid after logout")
	}

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || cookies[0].MaxAge >= 0 {
		t.Fatalf("want a cleared cookie (negative MaxAge), got %v", cookies)
	}
}

func TestLogoutHandlerWithNoCookieReportsUnauthenticated(t *testing.T) {
	repo := newFakeRepository()
	svc := newTestService(t, repo)

	var captured error
	writeErr := func(_ context.Context, _ http.ResponseWriter, err error) { captured = err }
	h := session.NewHandler(svc, false, time.Hour, 1<<20, writeErr)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/logout", nil)
	rec := httptest.NewRecorder()

	h.Logout(rec, req)

	if captured == nil {
		t.Fatal("want an error with no session cookie, got none")
	}
}

func TestValidateHandlerTakesTheTokenInTheBodyNotAQueryString(t *testing.T) {
	repo := newFakeRepository()
	seedActiveStaff(t, repo, "owner@example.com", "correct-horse-battery-staple")
	svc := newTestService(t, repo)
	token, _, err := svc.Login(context.Background(), "owner@example.com", "correct-horse-battery-staple")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	var captured error
	writeErr := func(_ context.Context, _ http.ResponseWriter, err error) { captured = err }
	h := session.NewHandler(svc, false, time.Hour, 1<<20, writeErr)

	// The token travels in the JSON body; a query string equivalent must not
	// be read at all (it would already be sitting in an access log).
	body := `{"token":"` + token + `"}`
	req := httptest.NewRequest(http.MethodPost, "/internal/sessions/validate?token=leaked-if-used", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h.Validate(rec, req)

	if captured != nil {
		t.Fatalf("unexpected error: %v", captured)
	}
	var resp session.ValidateResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.StaffID != "1" {
		t.Errorf("staffId: got %q", resp.StaffID)
	}
}
