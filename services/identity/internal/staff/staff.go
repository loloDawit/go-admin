// Package staff manages staff accounts: creation, self- and admin-initiated updates, deactivation, and password changes.
package staff

import "github.com/loloDawit/go-admin/services/identity/internal/errs"

var (
	ErrNotFound     = errs.ErrNotFound
	ErrEmailTaken   = errs.ErrEmailTaken
	ErrLastAdmin    = errs.ErrLastAdmin
	ErrRoleNotFound = errs.ErrRoleNotFound
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

// DefaultPageSize is the page a caller gets when it asks for no particular size.
const DefaultPageSize = 20

// ListQuery carries a listing request; a zero PageSize means the default and a
// zero Page means the first.
type ListQuery struct {
	Page     int
	PageSize int
}

// Page is List's response shape; Total comes from a separate count query,
// never a window function over the paged rows.
type Page struct {
	Items    []Staff
	Page     int
	PageSize int
	Total    int
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
