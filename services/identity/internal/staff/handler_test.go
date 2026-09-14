package staff_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/loloDawit/go-admin/platform/principal"
	"github.com/loloDawit/go-admin/services/identity/internal/staff"
)

var handlerTestKey = []byte("a-test-signing-key-at-least-32-bytes-long")

func newTestHandler() (*staff.Handler, *fakeRepository) {
	svc, repo := newTestService()
	return staff.NewHandler(svc, 1<<20, func(context.Context, http.ResponseWriter, error) {}), repo
}

// withChiURLParam attaches an "id" (or any named) URL param the way chi's
// router would after matching a pattern like /api/v1/staff/{id}, so a
// handler under test can call chi.URLParam without a real router in front.
func withChiURLParam(req *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

// signedRequest runs next through principal.Middleware exactly as the router
// wires it, signing p with the same key the middleware verifies against.
func signedRequest(t *testing.T, req *http.Request, p principal.Principal, next http.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	p.IssuedAt = time.Now()
	p.ExpiresAt = time.Now().Add(time.Minute)
	header, sig, err := principal.Sign(p, handlerTestKey)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	req.Header.Set(principal.HeaderPrincipal, header)
	req.Header.Set(principal.HeaderSignature, sig)

	var captured error
	middleware := principal.Middleware(handlerTestKey, func(w http.ResponseWriter, r *http.Request, err error) {
		captured = err
		w.WriteHeader(http.StatusUnauthorized)
	})(next)

	rec := httptest.NewRecorder()
	middleware.ServeHTTP(rec, req)
	if captured != nil {
		t.Fatalf("principal middleware rejected a validly signed principal: %v", captured)
	}
	return rec
}

func TestCreateHandlerReturnsThePasswordOnceInTheResponse(t *testing.T) {
	h, _ := newTestHandler()

	body := `{"email":"a@example.com","firstName":"A","lastName":"B","roleId":"1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/staff", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status: want 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp staff.CreateStaffResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Password == "" {
		t.Fatal("want a non-empty password in the create response")
	}
	if !resp.Staff.MustChangePassword {
		t.Fatal("want mustChangePassword true for a newly created staff member")
	}
}

// TestSelfUpdateCannotChangeRole is the direct descendant of the original
// application's worst defect: PATCH against the caller's own ID with a
// roleId field must be rejected outright — the self-update DTO carries no
// such field, so httpx.DecodeJSON's DisallowUnknownFields rejects the body
// rather than silently dropping the field.
func TestSelfUpdateCannotChangeRole(t *testing.T) {
	svc, _ := newTestService()
	var captured error
	writeErr := func(_ context.Context, w http.ResponseWriter, err error) {
		captured = err
		w.WriteHeader(http.StatusTeapot)
	}
	h := staff.NewHandler(svc, 1<<20, writeErr)

	created, _, err := svc.Create(context.Background(), staff.CreateStaff{Email: "self@example.com", FirstName: "A", LastName: "B", RoleID: 2})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	id := strconv.FormatInt(created.ID, 10)
	body := `{"firstName":"A","lastName":"B","email":"self@example.com","roleId":"1"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/staff/"+id, strings.NewReader(body))
	req = withChiURLParam(req, "id", id)

	rec := signedRequest(t, req, principal.Principal{StaffID: id}, h.Update)

	if captured == nil {
		t.Fatal("want an error for a self-update body carrying roleId, got none")
	}
	if rec.Code == http.StatusOK {
		t.Fatalf("a self-update carrying roleId must not be accepted as 200, got body %s", rec.Body.String())
	}

	got, err := svc.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.RoleID != 2 {
		t.Fatalf("role changed via self-update: want 2, got %d", got.RoleID)
	}
}

// TestAdminUpdateCanChangeAnotherStaffMembersRole exercises the same route
// with a different caller ID: the admin path (a different principal StaffID
// than the path ID) does accept roleId, so the guard above is about
// self-update specifically, not roleId in general.
func TestAdminUpdateCanChangeAnotherStaffMembersRole(t *testing.T) {
	svc, _ := newTestService()
	h := staff.NewHandler(svc, 1<<20, func(context.Context, http.ResponseWriter, error) {})

	target, _, err := svc.Create(context.Background(), staff.CreateStaff{Email: "target@example.com", FirstName: "A", LastName: "B", RoleID: 2})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	id := strconv.FormatInt(target.ID, 10)
	body := `{"firstName":"A","lastName":"B","email":"target@example.com","roleId":"1","isActive":true}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/staff/"+id, strings.NewReader(body))
	req = withChiURLParam(req, "id", id)

	rec := signedRequest(t, req, principal.Principal{StaffID: "999"}, h.Update)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d: %s", rec.Code, rec.Body.String())
	}
	got, err := svc.Get(context.Background(), target.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.RoleID != 1 {
		t.Fatalf("role: want 1, got %d", got.RoleID)
	}
}

// TestAdminUpdatePartialBodyLeavesOtherFieldsUnchanged pins the partial-update
// contract: an admin PATCH supplying only firstName must not blank out or
// deactivate anything else.
func TestAdminUpdatePartialBodyLeavesOtherFieldsUnchanged(t *testing.T) {
	svc, _ := newTestService()
	h := staff.NewHandler(svc, 1<<20, func(context.Context, http.ResponseWriter, error) {})

	target, _, err := svc.Create(context.Background(), staff.CreateStaff{Email: "target@example.com", FirstName: "A", LastName: "B", RoleID: 2})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	id := strconv.FormatInt(target.ID, 10)
	body := `{"firstName":"Changed"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/staff/"+id, strings.NewReader(body))
	req = withChiURLParam(req, "id", id)

	rec := signedRequest(t, req, principal.Principal{StaffID: "999"}, h.Update)
	if rec.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d: %s", rec.Code, rec.Body.String())
	}

	got, err := svc.Get(context.Background(), target.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.FirstName != "Changed" {
		t.Fatalf("firstName: want Changed, got %q", got.FirstName)
	}
	if got.Email != "target@example.com" {
		t.Fatalf("email changed by an omitted field: want target@example.com, got %q", got.Email)
	}
	if got.RoleID != 2 {
		t.Fatalf("roleId changed by an omitted field: want 2, got %d", got.RoleID)
	}
	if !got.IsActive {
		t.Fatal("isActive changed by an omitted field: want true (still active), got false")
	}
}

func TestChangePasswordHandlerUsesThePrincipalNotAPathID(t *testing.T) {
	svc, _ := newTestService()
	h := staff.NewHandler(svc, 1<<20, func(context.Context, http.ResponseWriter, error) {})

	created, pw, err := svc.Create(context.Background(), staff.CreateStaff{Email: "a@example.com", FirstName: "A", LastName: "B", RoleID: 1})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	id := strconv.FormatInt(created.ID, 10)

	body := `{"currentPassword":"` + pw + `","newPassword":"a-brand-new-password"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/me/password", strings.NewReader(body))

	rec := signedRequest(t, req, principal.Principal{StaffID: id}, h.ChangePassword)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status: want 204, got %d: %s", rec.Code, rec.Body.String())
	}

	got, err := svc.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.MustChangePassword {
		t.Fatal("want must_change_password cleared after a successful change")
	}
}

// requirePasswordChangedChain wires principal.Middleware in front of
// RequirePasswordChanged exactly as the router will (Task 10): the gate
// reads the principal that middleware verified, never a database.
func requirePasswordChangedChain(onErr func(context.Context, http.ResponseWriter, error), next http.Handler) http.Handler {
	return principal.Middleware(handlerTestKey, func(w http.ResponseWriter, r *http.Request, err error) {
		onErr(r.Context(), w, err)
	})(staff.RequirePasswordChanged(onErr)(next))
}

func TestRequirePasswordChangedBlocksEveryRouteExceptMePasswordAndLogout(t *testing.T) {
	var captured error
	onErr := func(_ context.Context, w http.ResponseWriter, err error) {
		captured = err
		w.WriteHeader(http.StatusForbidden)
	}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	chain := requirePasswordChangedChain(onErr, next)

	run := func(method, path string) *httptest.ResponseRecorder {
		captured = nil
		req := httptest.NewRequest(method, path, nil)
		header, sig, err := principal.Sign(principal.Principal{
			StaffID: "1", MustChangePassword: true,
			IssuedAt: time.Now(), ExpiresAt: time.Now().Add(time.Minute),
		}, handlerTestKey)
		if err != nil {
			t.Fatalf("sign: %v", err)
		}
		req.Header.Set(principal.HeaderPrincipal, header)
		req.Header.Set(principal.HeaderSignature, sig)
		rec := httptest.NewRecorder()
		chain.ServeHTTP(rec, req)
		return rec
	}

	if rec := run(http.MethodGet, "/api/v1/me"); rec.Code != http.StatusForbidden || captured == nil {
		t.Fatalf("want /me blocked with an error while a change is pending, got status %d, err %v", rec.Code, captured)
	}
	if rec := run(http.MethodPost, "/api/v1/me/password"); rec.Code != http.StatusOK {
		t.Fatalf("want /me/password allowed while a change is pending, got status %d", rec.Code)
	}
	if rec := run(http.MethodPost, "/api/v1/logout"); rec.Code != http.StatusOK {
		t.Fatalf("want /logout allowed while a change is pending, got status %d", rec.Code)
	}
}

// TestRequirePasswordChangedBlocksAGETToMePasswordPath pins that the gate
// matches method and path together: only POST mePasswordPath is exempt.
func TestRequirePasswordChangedBlocksAGETToMePasswordPath(t *testing.T) {
	var captured error
	onErr := func(_ context.Context, w http.ResponseWriter, err error) {
		captured = err
		w.WriteHeader(http.StatusForbidden)
	}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	chain := requirePasswordChangedChain(onErr, next)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/me/password", nil)
	header, sig, err := principal.Sign(principal.Principal{
		StaffID: "1", MustChangePassword: true,
		IssuedAt: time.Now(), ExpiresAt: time.Now().Add(time.Minute),
	}, handlerTestKey)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	req.Header.Set(principal.HeaderPrincipal, header)
	req.Header.Set(principal.HeaderSignature, sig)
	rec := httptest.NewRecorder()
	chain.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden || captured == nil {
		t.Fatalf("want GET /api/v1/me/password blocked while a change is pending, got status %d, err %v", rec.Code, captured)
	}
}

func TestRequirePasswordChangedAllowsEverythingOnceTheFlagIsClear(t *testing.T) {
	onErr := func(context.Context, http.ResponseWriter, error) {
		t.Fatal("onErr must not run once must_change_password is false")
	}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	chain := requirePasswordChangedChain(onErr, next)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	header, sig, err := principal.Sign(principal.Principal{
		StaffID: "1", MustChangePassword: false,
		IssuedAt: time.Now(), ExpiresAt: time.Now().Add(time.Minute),
	}, handlerTestKey)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	req.Header.Set(principal.HeaderPrincipal, header)
	req.Header.Set(principal.HeaderSignature, sig)
	rec := httptest.NewRecorder()
	chain.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", rec.Code)
	}
}
