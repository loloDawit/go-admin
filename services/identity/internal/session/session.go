// Package session builds and verifies staff sessions: login, logout, and the
// internal validation both the gateway and this service's own protected
// routes depend on.
package session

import "github.com/loloDawit/go-admin/services/identity/internal/errs"

// These re-export the identity service's registered sentinels (see
// internal/errs) under the names this package's own callers and tests
// already use; httperr's errors.Is checks still match the same values.
var (
	ErrInvalidCredentials     = errs.ErrInvalidCredentials
	ErrUnauthenticated        = errs.ErrUnauthenticated
	ErrPasswordChangeRequired = errs.ErrPasswordChangeRequired

	// ErrNotFound is returned by a Repository when a lookup matches no row:
	// an unknown email, or a token with no live session. It is exported so a
	// Repository implementation (real or test double) outside this package
	// can signal the same condition Service checks for.
	ErrNotFound = errs.ErrNotFound
)

// Authenticated is what authorizes a request, never the full staff record
// (Staff belongs to the staff package, a later task). Returning Staff from
// Login here would make session import staff, and staff already imports
// session for Hasher — an import cycle.
type Authenticated struct {
	StaffID            int64
	Email              string
	Permissions        []string
	MustChangePassword bool
}
