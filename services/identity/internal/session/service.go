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

// dummyLoginPassword is hashed once at construction and compared against on
// every login for an email that does not exist, so an unknown address costs
// the same bcrypt work as a real one. Its value is arbitrary; nothing ever
// authenticates with it.
const dummyLoginPassword = "identity-dummy-password-never-used-for-login"

type Service struct {
	repo      Repository
	hasher    *Hasher
	ttl       time.Duration
	dummyHash string
}

// NewService hashes dummyLoginPassword once up front: doing it per-login
// would defeat the point (Login must run a bcrypt compare of equal cost on
// every call, known email or not).
func NewService(repo Repository, hasher *Hasher, ttl time.Duration) (*Service, error) {
	dummyHash, err := hasher.Hash(dummyLoginPassword)
	if err != nil {
		return nil, errs.Wrap(errs.OpBuildDummyLoginHash, err)
	}
	return &Service{repo: repo, hasher: hasher, ttl: ttl, dummyHash: dummyHash}, nil
}

// Login compares the supplied password with bcrypt even when email matches no
// account, against a fixed dummy hash of equal cost — otherwise an unknown
// email returns faster than a wrong password for a real one, and response
// timing enumerates which addresses have accounts.
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

// Validate resolves a plaintext token to the principal the caller is
// authorized as. It never returns a session-local type: the platform
// principal.Principal is what platform/principal.Middleware and every
// downstream permission check already understand.
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

// Current returns the caller's own current record for GET /api/v1/me. It
// takes the principal's string StaffID (platform/principal carries no email,
// so /me cannot be answered from the signed principal alone) and fails
// closed for a deactivated account, in case a deactivation lands after the
// gateway last cached the principal but before this read.
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

// newToken generates the plaintext session token and the hash stored in its
// place. SHA-256, not bcrypt: this runs on the request path and the input
// already has 256 bits of entropy, so there is nothing for a work factor to
// protect.
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
