package session

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strconv"
	"time"

	"github.com/loloDawit/go-admin/platform/principal"
	"github.com/loloDawit/go-admin/services/identity/internal/errs"
)

// dummyLoginPassword is compared against on every login for an unknown email, so it costs the same bcrypt work as a real one.
const dummyLoginPassword = "identity-dummy-password-never-used-for-login"

type Service struct {
	repo      Repository
	hasher    *Hasher
	ttl       time.Duration
	dummyHash string
}

// NewService hashes dummyLoginPassword once up front; hashing it per-login would defeat the point.
func NewService(repo Repository, hasher *Hasher, ttl time.Duration) (*Service, error) {
	dummyHash, err := hasher.Hash(dummyLoginPassword)
	if err != nil {
		return nil, errs.Wrap(errs.OpBuildDummyLoginHash, err)
	}
	return &Service{repo: repo, hasher: hasher, ttl: ttl, dummyHash: dummyHash}, nil
}

// Login always runs a bcrypt compare, even against a dummy hash for an unknown email: otherwise response timing enumerates accounts.
func (s *Service) Login(ctx context.Context, email, password string) (string, Authenticated, error) {
	auth, err := s.repo.AuthByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			_ = s.hasher.Compare(s.dummyHash, password)
			return "", Authenticated{}, ErrInvalidCredentials
		}
		return "", Authenticated{}, errs.Wrap(errs.OpLookupStaffByEmail, err)
	}

	if err := s.hasher.Compare(auth.PasswordHash, password); err != nil {
		return "", Authenticated{}, ErrInvalidCredentials
	}
	if !auth.IsActive {
		return "", Authenticated{}, ErrInvalidCredentials
	}

	plain, hash, err := newToken()
	if err != nil {
		return "", Authenticated{}, err
	}
	if err := s.repo.CreateSession(ctx, auth.ID, hash, time.Now().Add(s.ttl)); err != nil {
		return "", Authenticated{}, errs.Wrap(errs.OpCreateSession, err)
	}

	return plain, Authenticated{
		StaffID:            auth.ID,
		Email:              auth.Email,
		Permissions:        auth.Permissions,
		MustChangePassword: auth.MustChangePassword,
	}, nil
}

// Validate returns platform/principal.Principal, never a session-local type: that is what downstream permission checks understand.
func (s *Service) Validate(ctx context.Context, token string) (principal.Principal, error) {
	auth, err := s.repo.AuthByTokenHash(ctx, hashToken(token))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return principal.Principal{}, ErrUnauthenticated
		}
		return principal.Principal{}, errs.Wrap(errs.OpLookupSession, err)
	}
	if !auth.IsActive {
		return principal.Principal{}, ErrUnauthenticated
	}

	now := time.Now()
	return principal.Principal{
		StaffID:            strconv.FormatInt(auth.ID, 10),
		Permissions:        auth.Permissions,
		MustChangePassword: auth.MustChangePassword,
		IssuedAt:           now,
		ExpiresAt:          now.Add(s.ttl),
	}, nil
}

// Current fails closed for a deactivated account, in case a deactivation lands after the gateway last cached the principal.
func (s *Service) Current(ctx context.Context, staffID string) (Authenticated, error) {
	id, err := strconv.ParseInt(staffID, 10, 64)
	if err != nil {
		return Authenticated{}, ErrUnauthenticated
	}

	auth, err := s.repo.AuthByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return Authenticated{}, ErrUnauthenticated
		}
		return Authenticated{}, errs.Wrap(errs.OpLookupStaffByID, err)
	}
	if !auth.IsActive {
		return Authenticated{}, ErrUnauthenticated
	}

	return Authenticated{
		StaffID:            auth.ID,
		Email:              auth.Email,
		Permissions:        auth.Permissions,
		MustChangePassword: auth.MustChangePassword,
	}, nil
}

// Revoke is idempotent: revoking a token with no live session (already
// logged out, already expired) is not an error.
func (s *Service) Revoke(ctx context.Context, token string) error {
	if err := s.repo.RevokeSession(ctx, hashToken(token)); err != nil {
		return errs.Wrap(errs.OpRevokeSession, err)
	}
	return nil
}

// newToken uses SHA-256, not bcrypt: the input already has 256 bits of entropy, so there is nothing for a work factor to protect.
func newToken() (plain string, hash []byte, err error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, errs.Wrap(errs.OpGenerateSessionToken, err)
	}
	plain = base64.RawURLEncoding.EncodeToString(raw)
	sum := sha256.Sum256([]byte(plain))
	return plain, sum[:], nil
}

func hashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}
