package session_test

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/loloDawit/go-admin/services/identity/internal/session"
)

const testCostSvc = 4

// fakeRepository is an in-memory double: each test builds one directly, so
// no test depends on another's state and none touches a database.
type fakeRepository struct {
	byEmail  map[string]session.StaffAuth
	sessions map[string]storedSession // keyed by the raw token hash bytes
}

type storedSession struct {
	staffID   int64
	expiresAt time.Time
	revoked   bool
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{
		byEmail:  make(map[string]session.StaffAuth),
		sessions: make(map[string]storedSession),
	}
}

func (f *fakeRepository) AuthByEmail(_ context.Context, email string) (session.StaffAuth, error) {
	auth, ok := f.byEmail[email]
	if !ok {
		return session.StaffAuth{}, session.ErrNotFound
	}
	return auth, nil
}

func (f *fakeRepository) AuthByID(_ context.Context, id int64) (session.StaffAuth, error) {
	for _, auth := range f.byEmail {
		if auth.ID == id {
			return auth, nil
		}
	}
	return session.StaffAuth{}, session.ErrNotFound
}

func (f *fakeRepository) AuthByTokenHash(_ context.Context, tokenHash []byte) (session.StaffAuth, error) {
	key := string(tokenHash)
	sess, ok := f.sessions[key]
	if !ok || sess.revoked || time.Now().After(sess.expiresAt) {
		return session.StaffAuth{}, session.ErrNotFound
	}
	for _, auth := range f.byEmail {
		if auth.ID == sess.staffID {
			return auth, nil
		}
	}
	return session.StaffAuth{}, session.ErrNotFound
}

func (f *fakeRepository) CreateSession(_ context.Context, staffID int64, tokenHash []byte, expiresAt time.Time) error {
	f.sessions[string(tokenHash)] = storedSession{staffID: staffID, expiresAt: expiresAt}
	return nil
}

func (f *fakeRepository) RevokeSession(_ context.Context, tokenHash []byte) error {
	key := string(tokenHash)
	sess, ok := f.sessions[key]
	if !ok {
		return nil
	}
	sess.revoked = true
	f.sessions[key] = sess
	return nil
}

func newTestService(t *testing.T, repo session.Repository) *session.Service {
	t.Helper()
	svc, err := session.NewService(repo, session.NewHasher(testCostSvc), time.Hour)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	return svc
}

func seedActiveStaff(t *testing.T, repo *fakeRepository, email, password string) session.StaffAuth {
	t.Helper()
	hash, err := session.NewHasher(testCostSvc).Hash(password)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	auth := session.StaffAuth{
		ID:           1,
		Email:        email,
		PasswordHash: hash,
		IsActive:     true,
		Permissions:  []string{"view_staff"},
	}
	repo.byEmail[email] = auth
	return auth
}

func TestLoginRejectsAnUnknownEmailAndAWrongPasswordIdentically(t *testing.T) {
	repo := newFakeRepository()
	seedActiveStaff(t, repo, "owner@example.com", "correct-horse-battery-staple")
	svc := newTestService(t, repo)
	ctx := context.Background()

	_, _, unknown := svc.Login(ctx, "nobody@example.com", "x")
	_, _, wrong := svc.Login(ctx, "owner@example.com", "wrong-password")

	if !errors.Is(unknown, session.ErrInvalidCredentials) {
		t.Fatalf("unknown email: want ErrInvalidCredentials, got %v", unknown)
	}
	if !errors.Is(wrong, session.ErrInvalidCredentials) {
		t.Fatalf("wrong password: want ErrInvalidCredentials, got %v", wrong)
	}
}

func TestLoginRefusesADeactivatedAccount(t *testing.T) {
	repo := newFakeRepository()
	auth := seedActiveStaff(t, repo, "owner@example.com", "correct-horse-battery-staple")
	auth.IsActive = false
	repo.byEmail[auth.Email] = auth
	svc := newTestService(t, repo)

	_, _, err := svc.Login(context.Background(), "owner@example.com", "correct-horse-battery-staple")
	if !errors.Is(err, session.ErrInvalidCredentials) {
		t.Fatalf("deactivated account: want ErrInvalidCredentials, got %v", err)
	}
}

func TestLoginSucceedsAndValidateReturnsThePrincipal(t *testing.T) {
	repo := newFakeRepository()
	seedActiveStaff(t, repo, "owner@example.com", "correct-horse-battery-staple")
	svc := newTestService(t, repo)
	ctx := context.Background()

	token, auth, err := svc.Login(ctx, "owner@example.com", "correct-horse-battery-staple")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if token == "" {
		t.Fatal("want a non-empty token")
	}
	if auth.Email != "owner@example.com" {
		t.Errorf("email: got %q", auth.Email)
	}

	p, err := svc.Validate(ctx, token)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if p.StaffID != "1" {
		t.Errorf("StaffID: want %q, got %q", "1", p.StaffID)
	}
}

func TestValidateRejectsARevokedSession(t *testing.T) {
	repo := newFakeRepository()
	seedActiveStaff(t, repo, "owner@example.com", "correct-horse-battery-staple")
	svc := newTestService(t, repo)
	ctx := context.Background()

	token, _, err := svc.Login(ctx, "owner@example.com", "correct-horse-battery-staple")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if err := svc.Revoke(ctx, token); err != nil {
		t.Fatalf("revoke: %v", err)
	}

	if _, err := svc.Validate(ctx, token); !errors.Is(err, session.ErrUnauthenticated) {
		t.Fatalf("want ErrUnauthenticated for a revoked session, got %v", err)
	}
}

func TestValidateRejectsAnExpiredSession(t *testing.T) {
	repo := newFakeRepository()
	seedActiveStaff(t, repo, "owner@example.com", "correct-horse-battery-staple")
	svc, err := session.NewService(repo, session.NewHasher(testCostSvc), -time.Second)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	ctx := context.Background()

	token, _, err := svc.Login(ctx, "owner@example.com", "correct-horse-battery-staple")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	if _, err := svc.Validate(ctx, token); !errors.Is(err, session.ErrUnauthenticated) {
		t.Fatalf("want ErrUnauthenticated for an expired session, got %v", err)
	}
}

// A session created while active must stop authenticating the moment the account is deactivated, not wait to expire or be revoked.
func TestValidateRejectsADeactivatedAccountWithALiveSession(t *testing.T) {
	repo := newFakeRepository()
	auth := seedActiveStaff(t, repo, "owner@example.com", "correct-horse-battery-staple")
	svc := newTestService(t, repo)
	ctx := context.Background()

	token, _, err := svc.Login(ctx, "owner@example.com", "correct-horse-battery-staple")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	auth.IsActive = false
	repo.byEmail[auth.Email] = auth

	if _, err := svc.Validate(ctx, token); !errors.Is(err, session.ErrUnauthenticated) {
		t.Fatalf("want ErrUnauthenticated for a deactivated account with a live session, got %v", err)
	}
}

func TestCurrentReturnsTheFullRecordForAnAuthenticatedID(t *testing.T) {
	repo := newFakeRepository()
	seedActiveStaff(t, repo, "owner@example.com", "correct-horse-battery-staple")
	svc := newTestService(t, repo)

	auth, err := svc.Current(context.Background(), "1")
	if err != nil {
		t.Fatalf("current: %v", err)
	}
	if auth.Email != "owner@example.com" {
		t.Errorf("email: want %q, got %q", "owner@example.com", auth.Email)
	}
}

func TestCurrentRejectsAnUnknownStaffID(t *testing.T) {
	repo := newFakeRepository()
	svc := newTestService(t, repo)

	if _, err := svc.Current(context.Background(), "999"); !errors.Is(err, session.ErrUnauthenticated) {
		t.Fatalf("want ErrUnauthenticated for an unknown staff ID, got %v", err)
	}
}

func TestCurrentRejectsAMalformedStaffID(t *testing.T) {
	repo := newFakeRepository()
	svc := newTestService(t, repo)

	if _, err := svc.Current(context.Background(), "not-a-number"); !errors.Is(err, session.ErrUnauthenticated) {
		t.Fatalf("want ErrUnauthenticated for a malformed staff ID, got %v", err)
	}
}

func TestCurrentRejectsADeactivatedAccount(t *testing.T) {
	repo := newFakeRepository()
	auth := seedActiveStaff(t, repo, "owner@example.com", "correct-horse-battery-staple")
	auth.IsActive = false
	repo.byEmail[auth.Email] = auth
	svc := newTestService(t, repo)

	if _, err := svc.Current(context.Background(), "1"); !errors.Is(err, session.ErrUnauthenticated) {
		t.Fatalf("want ErrUnauthenticated for a deactivated account, got %v", err)
	}
}

func TestTheStoredTokenIsNotThePlaintext(t *testing.T) {
	repo := newFakeRepository()
	seedActiveStaff(t, repo, "owner@example.com", "correct-horse-battery-staple")
	svc := newTestService(t, repo)
	ctx := context.Background()

	token, _, err := svc.Login(ctx, "owner@example.com", "correct-horse-battery-staple")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	for stored := range repo.sessions {
		if bytes.Contains([]byte(stored), []byte(token)) {
			t.Fatal("the session table holds the plaintext token")
		}
	}
	if len(repo.sessions) == 0 {
		t.Fatal("no session was stored")
	}
}
