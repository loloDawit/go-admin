// Package session builds and verifies staff sessions: login, logout, and the internal validation the gateway depends on.
package session

import "github.com/loloDawit/go-admin/services/identity/internal/errs"

// These re-export the identity service's registered sentinels under this package's existing names.
var (
	ErrInvalidCredentials     = errs.ErrInvalidCredentials
	ErrUnauthenticated        = errs.ErrUnauthenticated
	ErrPasswordChangeRequired = errs.ErrPasswordChangeRequired
	ErrNotFound               = errs.ErrNotFound
)

// Authenticated, not staff.Staff, is returned here: staff already imports session for Hasher, so the reverse import would cycle.
type Authenticated struct {
	StaffID            int64
	Email              string
	Permissions        []string
	MustChangePassword bool
}
