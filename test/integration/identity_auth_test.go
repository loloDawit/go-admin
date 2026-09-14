//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

type meResponse struct {
	StaffID            string   `json:"staffId"`
	Email              string   `json:"email"`
	Permissions        []string `json:"permissions"`
	MustChangePassword bool     `json:"mustChangePassword"`
}

// required fails rather than defaulting: a default here would let the suite
// pass against a stack configured differently from what it asserts.
func required(t *testing.T, key string) string {
	t.Helper()
	v := os.Getenv(key)
	if v == "" {
		t.Fatalf("%s must be set; `make test-integration` exports it", key)
	}
	return v
}

func loggedInClient(t *testing.T) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookie jar: %v", err)
	}
	c := &http.Client{Jar: jar, Timeout: 10 * time.Second}

	body, _ := json.Marshal(map[string]string{
		"email":    required(t, "OWNER_EMAIL"),
		"password": required(t, "OWNER_PASSWORD"),
	})
	resp, err := c.Post(gatewayURL()+"/api/v1/login", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login status: want 200, got %d", resp.StatusCode)
	}
	return c
}

func getStatus(t *testing.T, c *http.Client, path string) int {
	t.Helper()
	resp, err := c.Get(gatewayURL() + path)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

func TestLoginThroughTheGatewayReturnsTheCaller(t *testing.T) {
	c := loggedInClient(t)

	resp, err := c.Get(gatewayURL() + "/api/v1/me")
	if err != nil {
		t.Fatalf("me: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("me status: want 200, got %d", resp.StatusCode)
	}

	var me meResponse
	if err := json.NewDecoder(resp.Body).Decode(&me); err != nil {
		t.Fatalf("decode me: %v", err)
	}
	if me.Email != required(t, "OWNER_EMAIL") {
		t.Errorf("email: want %q, got %q", os.Getenv("OWNER_EMAIL"), me.Email)
	}
	if !slices.Contains(me.Permissions, "edit_staff") {
		t.Errorf("owner lacks edit_staff: %v", me.Permissions)
	}
}

// A well-formed payload with an invalid signature must be refused. The gateway
// also strips client-supplied principal headers before it reads the cookie, so
// neither half of the forgery survives.
func TestAForgedPrincipalIsRejected(t *testing.T) {
	payload, _ := json.Marshal(map[string]any{
		"staffId":     "1",
		"permissions": []string{"edit_staff"},
		"issuedAt":    time.Now().Format(time.RFC3339),
		"expiresAt":   time.Now().Add(time.Hour).Format(time.RFC3339),
	})

	req, err := http.NewRequest(http.MethodGet, gatewayURL()+"/api/v1/me", nil)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	req.Header.Set("X-Principal", base64.RawURLEncoding.EncodeToString(payload))
	req.Header.Set("X-Principal-Signature", "not-a-signature")

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		t.Fatalf("me: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		t.Fatal("a forged principal was accepted")
	}
}

func TestLogoutRevokesWithinTheCacheBudget(t *testing.T) {
	ttl, err := time.ParseDuration(required(t, "SESSION_CACHE_TTL"))
	if err != nil {
		t.Fatalf("SESSION_CACHE_TTL: %v", err)
	}

	c := loggedInClient(t)
	resp, err := c.Post(gatewayURL()+"/api/v1/logout", "application/json", nil)
	if err != nil {
		t.Fatalf("logout: %v", err)
	}
	resp.Body.Close()

	time.Sleep(ttl + 2*time.Second)

	if got := getStatus(t, c, "/api/v1/me"); got != http.StatusUnauthorized {
		t.Fatalf("after logout and %s: want 401, got %d", ttl, got)
	}
}

// seedSessionFor inserts a staff member and one session row directly, so the
// SQL guards in authByTokenHashQuery are exercised against real Postgres. The
// session unit tests run against an in-memory fake and pass with those guards
// deleted.
func seedSessionFor(t *testing.T, active bool, revoked bool, expires time.Time) string {
	t.Helper()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn(t, "identity_user", "dev_only_identity", "identity_db"))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { conn.Close(ctx) })

	email := "guard-" + time.Now().Format("150405.000000000") + "@example.com"
	var staffID int64
	err = conn.QueryRow(ctx, `
		INSERT INTO staff (email, first_name, last_name, password_hash, role_id, is_active)
		SELECT $1, 'Guard', 'Case', 'x', r.id, $2 FROM roles r WHERE r.name = 'admin'
		RETURNING id`, email, active).Scan(&staffID)
	if err != nil {
		t.Fatalf("insert staff: %v", err)
	}
	t.Cleanup(func() {
		_, _ = conn.Exec(context.Background(), `DELETE FROM staff WHERE id = $1`, staffID)
	})

	token := "guard-token-" + email
	sum := sha256.Sum256([]byte(token))
	var revokedAt *time.Time
	if revoked {
		now := time.Now()
		revokedAt = &now
	}
	_, err = conn.Exec(ctx, `
		INSERT INTO sessions (token_hash, staff_id, expires_at, revoked_at)
		VALUES ($1, $2, $3, $4)`, sum[:], staffID, expires, revokedAt)
	if err != nil {
		t.Fatalf("insert session: %v", err)
	}
	return token
}

func meWithToken(t *testing.T, token string) int {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, gatewayURL()+"/api/v1/me", nil)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	req.AddCookie(&http.Cookie{Name: "session", Value: token})
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		t.Fatalf("me: %v", err)
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

func TestALiveSessionForAnActiveStaffMemberIsAccepted(t *testing.T) {
	token := seedSessionFor(t, true, false, time.Now().Add(time.Hour))
	if got := meWithToken(t, token); got != http.StatusOK {
		t.Fatalf("want 200, got %d", got)
	}
}

func TestADeactivatedStaffMemberWithALiveSessionIsRefused(t *testing.T) {
	token := seedSessionFor(t, false, false, time.Now().Add(time.Hour))
	if got := meWithToken(t, token); got != http.StatusUnauthorized {
		t.Fatalf("a deactivated staff member authenticated: want 401, got %d", got)
	}
}

func TestARevokedSessionIsRefused(t *testing.T) {
	token := seedSessionFor(t, true, true, time.Now().Add(time.Hour))
	if got := meWithToken(t, token); got != http.StatusUnauthorized {
		t.Fatalf("a revoked session authenticated: want 401, got %d", got)
	}
}

func TestAnExpiredSessionIsRefused(t *testing.T) {
	token := seedSessionFor(t, true, false, time.Now().Add(-time.Hour))
	if got := meWithToken(t, token); got != http.StatusUnauthorized {
		t.Fatalf("an expired session authenticated: want 401, got %d", got)
	}
}

func identityURL() string {
	if u := os.Getenv("IDENTITY_URL"); u != "" {
		return u
	}
	return "http://localhost:8081"
}

// validateStatus calls identity's internal endpoint directly, not through the
// gateway. /api/v1/me also refuses a deactivated caller via Service.Current,
// so only this path isolates the guard on Validate itself.
func validateStatus(t *testing.T, token string) int {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"token": token})
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Post(
		identityURL()+"/internal/sessions/validate", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

func TestValidateRefusesADeactivatedStaffMember(t *testing.T) {
	token := seedSessionFor(t, false, false, time.Now().Add(time.Hour))
	if got := validateStatus(t, token); got == http.StatusOK {
		t.Fatal("Validate resolved a principal for a deactivated staff member")
	}
}

func TestValidateResolvesAnActiveStaffMember(t *testing.T) {
	token := seedSessionFor(t, true, false, time.Now().Add(time.Hour))
	if got := validateStatus(t, token); got != http.StatusOK {
		t.Fatalf("want 200, got %d", got)
	}
}
