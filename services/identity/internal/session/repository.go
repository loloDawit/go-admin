package session

import (
	"context"
	"time"
)

// StaffAuth is what login and validation need from a staff row, not the full staff.Staff record.
type StaffAuth struct {
	ID                 int64
	Email              string
	PasswordHash       string
	IsActive           bool
	MustChangePassword bool
	Permissions        []string
}

// Repository: ErrNotFound signals "no matching row"; every other error is a driver-level failure the service wraps.
type Repository interface {
	AuthByEmail(ctx context.Context, email string) (StaffAuth, error)

	AuthByID(ctx context.Context, id int64) (StaffAuth, error)

	// AuthByTokenHash's query itself excludes revoked or expired sessions, so a hit here is always valid.
	AuthByTokenHash(ctx context.Context, tokenHash []byte) (StaffAuth, error)

	CreateSession(ctx context.Context, staffID int64, tokenHash []byte, expiresAt time.Time) error

	// RevokeSession is idempotent: a repeated logout is harmless.
	RevokeSession(ctx context.Context, tokenHash []byte) error
}
