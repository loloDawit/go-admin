package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/loloDawit/go-admin/platform/observability"
	"github.com/loloDawit/go-admin/platform/principal"
	"github.com/loloDawit/go-admin/platform/readiness"
	"github.com/loloDawit/go-admin/services/identity/internal/httperr"
	"github.com/loloDawit/go-admin/services/identity/internal/permission"
	"github.com/loloDawit/go-admin/services/identity/internal/role"
	"github.com/loloDawit/go-admin/services/identity/internal/schemacheck"
	"github.com/loloDawit/go-admin/services/identity/internal/session"
	"github.com/loloDawit/go-admin/services/identity/internal/staff"
)

// testPrincipalKey is the HMAC key newRouter's principal.Middleware group verifies against.
var testPrincipalKey = []byte("0123456789012345678901234567890123456789")

// newTestAuthzHandlers builds handlers over nil repositories: none of the routes these router-level tests exercise reach one.
func newTestAuthzHandlers(errWriter func(context.Context, http.ResponseWriter, error)) (*staff.Handler, *role.Handler, *permission.Handler) {
	staffHandler := staff.NewHandler(staff.NewService(nil, session.NewHasher(4)), 1<<20, errWriter)
	roleHandler := role.NewHandler(role.NewService(nil), 1<<20, errWriter)
	return staffHandler, roleHandler, permission.NewHandler()
}

// emptySessionRepository matches no staff and no session; the routes these
// tests exercise (/healthz, /readyz, /boom) never call it.
type emptySessionRepository struct{}

func (emptySessionRepository) AuthByEmail(context.Context, string) (session.StaffAuth, error) {
	return session.StaffAuth{}, session.ErrNotFound
}

func (emptySessionRepository) AuthByTokenHash(context.Context, []byte) (session.StaffAuth, error) {
	return session.StaffAuth{}, session.ErrNotFound
}

func (emptySessionRepository) AuthByID(context.Context, int64) (session.StaffAuth, error) {
	return session.StaffAuth{}, session.ErrNotFound
}

func (emptySessionRepository) CreateSession(context.Context, int64, []byte, time.Time) error {
	return nil
}

func (emptySessionRepository) RevokeSession(context.Context, []byte) error {
	return nil
}

func newTestSessionHandler(t *testing.T, writeErr func(context.Context, http.ResponseWriter, error)) *session.Handler {
	t.Helper()
	svc, err := session.NewService(emptySessionRepository{}, session.NewHasher(4), time.Hour)
	if err != nil {
		t.Fatalf("session.NewService: %v", err)
	}
	return session.NewHandler(svc, false, time.Hour, 1<<20, writeErr)
}

// memorySessionRepository backs the router-level tests below, which exercise real routing and middleware, not a database.
type memorySessionRepository struct {
	byEmail  map[string]session.StaffAuth
	sessions map[string]memorySession
}

type memorySession struct {
	staffID   int64
	expiresAt time.Time
	revoked   bool
}

func newMemorySessionRepository() *memorySessionRepository {
	return &memorySessionRepository{
		byEmail:  make(map[string]session.StaffAuth),
		sessions: make(map[string]memorySession),
	}
}

func (r *memorySessionRepository) AuthByEmail(_ context.Context, email string) (session.StaffAuth, error) {
	auth, ok := r.byEmail[email]
	if !ok {
		return session.StaffAuth{}, session.ErrNotFound
	}
	return auth, nil
}

func (r *memorySessionRepository) AuthByTokenHash(_ context.Context, tokenHash []byte) (session.StaffAuth, error) {
	sess, ok := r.sessions[string(tokenHash)]
	if !ok || sess.revoked || time.Now().After(sess.expiresAt) {
		return session.StaffAuth{}, session.ErrNotFound
	}
	for _, auth := range r.byEmail {
		if auth.ID == sess.staffID {
			return auth, nil
		}
	}
	return session.StaffAuth{}, session.ErrNotFound
}

func (r *memorySessionRepository) AuthByID(_ context.Context, id int64) (session.StaffAuth, error) {
	for _, auth := range r.byEmail {
		if auth.ID == id {
			return auth, nil
		}
	}
	return session.StaffAuth{}, session.ErrNotFound
}

func (r *memorySessionRepository) CreateSession(_ context.Context, staffID int64, tokenHash []byte, expiresAt time.Time) error {
	r.sessions[string(tokenHash)] = memorySession{staffID: staffID, expiresAt: expiresAt}
	return nil
}

func (r *memorySessionRepository) RevokeSession(_ context.Context, tokenHash []byte) error {
	sess, ok := r.sessions[string(tokenHash)]
	if !ok {
		return nil
	}
	sess.revoked = true
	r.sessions[string(tokenHash)] = sess
	return nil
}

func newRouterWithLogin(t *testing.T) (*chi.Mux, *memorySessionRepository) {
	t.Helper()
	logger, _ := observability.NewCaptured()
	errWriter := httperr.New(logger)

	repo := newMemorySessionRepository()
	hash, err := session.NewHasher(4).Hash("correct-horse-battery-staple")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	repo.byEmail["owner@example.com"] = session.StaffAuth{
		ID:           1,
		Email:        "owner@example.com",
		PasswordHash: hash,
		IsActive:     true,
		Permissions:  []string{"edit_staff"},
	}

	svc, err := session.NewService(repo, session.NewHasher(4), time.Hour)
	if err != nil {
		t.Fatalf("session.NewService: %v", err)
	}
	sessionHandler := session.NewHandler(svc, false, time.Hour, 1<<20, errWriter.Write)

	platformSvc := schemacheck.NewService(&alwaysFailRepo{})
	ready := readiness.NewHandler(platformSvc.Probe, errWriter.Write)

	staffHandler, roleHandler, permissionHandler := newTestAuthzHandlers(errWriter.Write)
	r := newRouter(logger, ready, sessionHandler, staffHandler, roleHandler, permissionHandler, testPrincipalKey, errWriter.Write)
	return r, repo
}

// This confirms the logged-out token no longer validates: logout is a no-op unless it actually revokes the session.
func TestLoginRouteSetsACookieThatLogoutRevokes(t *testing.T) {
	r, _ := newRouterWithLogin(t)

	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/login",
		strings.NewReader(`{"email":"owner@example.com","password":"correct-horse-battery-staple"}`))
	loginRec := httptest.NewRecorder()
	r.ServeHTTP(loginRec, loginReq)

	if loginRec.Code != http.StatusOK {
		t.Fatalf("login status: want 200, got %d: %s", loginRec.Code, loginRec.Body.String())
	}
	cookies := loginRec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("want one cookie from login, got %d", len(cookies))
	}

	p := principal.Principal{
		StaffID:   "1",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(time.Minute),
	}
	header, sig, err := principal.Sign(p, testPrincipalKey)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	logoutReq := httptest.NewRequest(http.MethodPost, "/api/v1/logout", nil)
	logoutReq.AddCookie(cookies[0])
	logoutReq.Header.Set(principal.HeaderPrincipal, header)
	logoutReq.Header.Set(principal.HeaderSignature, sig)
	logoutRec := httptest.NewRecorder()
	r.ServeHTTP(logoutRec, logoutReq)
	if logoutRec.Code != http.StatusNoContent {
		t.Fatalf("logout status: want 204, got %d: %s", logoutRec.Code, logoutRec.Body.String())
	}

	validateReq := httptest.NewRequest(http.MethodPost, "/internal/sessions/validate",
		strings.NewReader(`{"token":"`+cookies[0].Value+`"}`))
	validateRec := httptest.NewRecorder()
	r.ServeHTTP(validateRec, validateReq)
	if validateRec.Code != http.StatusUnauthorized {
		t.Fatalf("validate after logout: want 401, got %d: %s", validateRec.Code, validateRec.Body.String())
	}
}

// A request carrying only the session cookie, no X-Principal/X-Principal-Signature, is correctly refused.
func TestMeRouteRejectsAnUnsignedRequest(t *testing.T) {
	r, _ := newRouterWithLogin(t)

	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/login",
		strings.NewReader(`{"email":"owner@example.com","password":"correct-horse-battery-staple"}`))
	loginRec := httptest.NewRecorder()
	r.ServeHTTP(loginRec, loginReq)
	cookies := loginRec.Result().Cookies()

	meReq := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	meReq.AddCookie(cookies[0])
	meRec := httptest.NewRecorder()
	r.ServeHTTP(meRec, meReq)
	if meRec.Code != http.StatusUnauthorized {
		t.Fatalf("me without a signed principal: want 401, got %d", meRec.Code)
	}
}

func TestMeRouteWithASignedPrincipalReturnsTheCaller(t *testing.T) {
	r, _ := newRouterWithLogin(t)

	p := principal.Principal{
		StaffID:   "1",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(time.Minute),
	}
	header, sig, err := principal.Sign(p, testPrincipalKey)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	meReq := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	meReq.Header.Set(principal.HeaderPrincipal, header)
	meReq.Header.Set(principal.HeaderSignature, sig)
	meRec := httptest.NewRecorder()
	r.ServeHTTP(meRec, meReq)

	if meRec.Code != http.StatusOK {
		t.Fatalf("me status: want 200, got %d: %s", meRec.Code, meRec.Body.String())
	}
	var resp session.AuthResponse
	if err := json.NewDecoder(meRec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Email != "owner@example.com" {
		t.Errorf("email: want %q, got %q", "owner@example.com", resp.Email)
	}
}

// TestPermissionsRouteEnforcesViewRoles exercises authz.Require through the real router: 401, 403, then 200.
func TestPermissionsRouteEnforcesViewRoles(t *testing.T) {
	r, _ := newRouterWithLogin(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/permissions", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("without a principal: want 401, got %d", rec.Code)
	}

	withoutViewRoles := principal.Principal{StaffID: "1", IssuedAt: time.Now(), ExpiresAt: time.Now().Add(time.Minute)}
	header, sig, err := principal.Sign(withoutViewRoles, testPrincipalKey)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	req = httptest.NewRequest(http.MethodGet, "/api/v1/permissions", nil)
	req.Header.Set(principal.HeaderPrincipal, header)
	req.Header.Set(principal.HeaderSignature, sig)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("without view_roles: want 403, got %d: %s", rec.Code, rec.Body.String())
	}

	withViewRoles := principal.Principal{
		StaffID: "1", Permissions: []string{"view_roles"},
		IssuedAt: time.Now(), ExpiresAt: time.Now().Add(time.Minute),
	}
	header, sig, err = principal.Sign(withViewRoles, testPrincipalKey)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	req = httptest.NewRequest(http.MethodGet, "/api/v1/permissions", nil)
	req.Header.Set(principal.HeaderPrincipal, header)
	req.Header.Set(principal.HeaderSignature, sig)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("with view_roles: want 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestValidateRouteIsRegisteredAndTakesTheTokenInTheBody(t *testing.T) {
	r, _ := newRouterWithLogin(t)

	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/login",
		strings.NewReader(`{"email":"owner@example.com","password":"correct-horse-battery-staple"}`))
	loginRec := httptest.NewRecorder()
	r.ServeHTTP(loginRec, loginReq)
	token := loginRec.Result().Cookies()[0].Value

	validateReq := httptest.NewRequest(http.MethodPost, "/internal/sessions/validate",
		strings.NewReader(`{"token":"`+token+`"}`))
	validateRec := httptest.NewRecorder()
	r.ServeHTTP(validateRec, validateReq)

	if validateRec.Code != http.StatusOK {
		t.Fatalf("validate status: want 200, got %d: %s", validateRec.Code, validateRec.Body.String())
	}
}

// The two response bodies must be byte-identical, not just the same status code, or the account is enumerable.
func TestLoginRouteRejectsWrongPasswordWithByteIdenticalResponses(t *testing.T) {
	r, _ := newRouterWithLogin(t)

	bodies := []string{
		`{"email":"owner@example.com","password":"wrong"}`,
		`{"email":"nobody@example.com","password":"wrong"}`,
	}
	var responses []string
	for _, body := range bodies {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/login", strings.NewReader(body))
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("body %s: want 401, got %d", body, rec.Code)
		}
		responses = append(responses, rec.Body.String())
	}
	if responses[0] != responses[1] {
		t.Fatalf("responses differ, enumerating the account:\nknown email:   %q\nunknown email: %q", responses[0], responses[1])
	}
}

// alwaysFailRepo counts calls so a test can assert a route never touched it.
type alwaysFailRepo struct {
	calls int
}

func (r *alwaysFailRepo) SchemaState(context.Context) (schemacheck.SchemaState, error) {
	r.calls++
	return schemacheck.SchemaState{}, errors.New("dial tcp 10.0.0.5:5432: connect: connection refused")
}

func TestPanickingHandlerStillProducesALogLineWithStatus500(t *testing.T) {
	logger, captured := observability.NewCaptured()
	errWriter := httperr.New(logger)
	svc := schemacheck.NewService(&alwaysFailRepo{})
	ready := readiness.NewHandler(svc.Probe, errWriter.Write)

	sessionHandler := newTestSessionHandler(t, errWriter.Write)
	staffHandler, roleHandler, permissionHandler := newTestAuthzHandlers(errWriter.Write)
	r := newRouter(logger, ready, sessionHandler, staffHandler, roleHandler, permissionHandler, testPrincipalKey, errWriter.Write)
	r.Get("/boom", func(http.ResponseWriter, *http.Request) {
		panic("kaboom")
	})

	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("response status: want 500, got %d", rec.Code)
	}

	records := captured.Records()
	if len(records) != 1 {
		t.Fatalf("want exactly one log record for the panicking request, got %d", len(records))
	}

	v, ok := captured.Attr(0, "status")
	if !ok {
		t.Fatal("log record missing status attribute")
	}
	if v.Int64() != http.StatusInternalServerError {
		t.Errorf("logged status: want 500, got %d", v.Int64())
	}
}

// This pins spec §6: liveness never touches the database, and readiness reports failure without leaking driver detail.
func TestStartupContractHealthzSkipsTheDatabaseAndReadyzReportsFailureSafely(t *testing.T) {
	logger, captured := observability.NewCaptured()
	errWriter := httperr.New(logger)
	repo := &alwaysFailRepo{}
	svc := schemacheck.NewService(repo)
	ready := readiness.NewHandler(svc.Probe, errWriter.Write)

	sessionHandler := newTestSessionHandler(t, errWriter.Write)
	staffHandler, roleHandler, permissionHandler := newTestAuthzHandlers(errWriter.Write)
	r := newRouter(logger, ready, sessionHandler, staffHandler, roleHandler, permissionHandler, testPrincipalKey, errWriter.Write)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("healthz status: want 200, got %d", rec.Code)
	}
	if repo.calls != 0 {
		t.Errorf("healthz touched the repository: called %d times", repo.calls)
	}

	for _, route := range []string{"/readyz"} {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, route, nil))
		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("%s status: want 503, got %d", route, rec.Code)
		}
		if body := rec.Body.String(); strings.Contains(body, "10.0.0.5") || strings.Contains(body, "connection refused") {
			t.Errorf("%s body leaked driver detail: %q", route, body)
		}
	}

	// A wrap that only hides the detail, instead of routing it to the log, would still pass the body assertions above.
	var loggedCause bool
	for _, record := range captured.Records() {
		record.Attrs(func(a slog.Attr) bool {
			if a.Key == "error" && strings.Contains(a.Value.String(), "connection refused") {
				loggedCause = true
			}
			return true
		})
	}
	if !loggedCause {
		t.Error("want the driver cause reachable in a log record's error attribute, found none")
	}
}
