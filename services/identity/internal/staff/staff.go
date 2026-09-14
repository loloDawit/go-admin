// Package staff manages staff accounts: creation, self- and admin-initiated updates, deactivation, and password changes.
package staff

import "github.com/loloDawit/go-admin/services/identity/internal/errs"

var (
	ErrNotFound   = errs.ErrNotFound
	ErrEmailTaken = errs.ErrEmailTaken
	ErrLastAdmin  = errs.ErrLastAdmin
)

// Staff never carries a password hash; that stays inside the repository, reachable only through ChangePassword.
type Staff struct {
	ID                 int64
	Email              string
	FirstName          string
	LastName           string
	RoleID             int64
	IsActive           bool
	MustChangePassword bool
}

// CreateStaff is Create's input. RoleID is required: every staff member has
// a role from the moment they exist.
type CreateStaff struct {
	Email     string
	FirstName string
	LastName  string
	RoleID    int64
}

// UpdateStaff is Update's input; a nil field means "leave unchanged".
type UpdateStaff struct {
	FirstName *string
	LastName  *string
	Email     *string
	RoleID    *int64
	IsActive  *bool
}
