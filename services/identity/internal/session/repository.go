package session

import (
	"context"
	"time"
)

// StaffAuth is exactly what login and validation need from a staff row: not
// the full staff record (that is staff.Staff, a later task), just what
// authenticates a request and shapes the principal.
type StaffAuth struct {
	ID                 int64
	Email              string
	PasswordHash       string
	IsActive           bool
	MustChangePassword bool
	Permissions        []string
}

// Repository is the data access this package needs. ErrNotFound signals "no
// matching row" for both lookups; every other error is a driver-level
// failure the service wraps rather than a business outcome.
type Repository interface {
	// AuthByEmail looks up one staff row by email, with the permissions its
	// role currently holds.
	AuthByEmail(ctx context.Context, email string) (StaffAuth, error)

	// AuthByID looks up one staff row by ID, with the permissions its role
	// currently holds. GET /api/v1/me uses this to answer with the caller's
	// current record rather than only what travelled on the signed
	// principal (which carries no email).
	AuthByID(ctx context.Context, id int64) (StaffAuth, error)

	// AuthByTokenHash looks up the staff row behind a live session: the
	// query itself excludes revoked or expired sessions, so a hit here is
	// always a valid one.
	AuthByTokenHash(ctx context.Context, tokenHash []byte) (StaffAuth, error)

	CreateSession(ctx context.Context, staffID int64, tokenHash []byte, expiresAt time.Time) error

	// RevokeSession is idempotent: revoking a session that is already gone
	// or already revoked is not an error, so a repeated logout is harmless.
	RevokeSession(ctx context.Context, tokenHash []byte) error
}
