// Package role manages roles and the permission sets assigned to them.
package role

import "github.com/loloDawit/go-admin/services/identity/internal/errs"

var (
	ErrNotFound              = errs.ErrNotFound
	ErrNameTaken             = errs.ErrRoleNameTaken
	ErrInUse                 = errs.ErrRoleInUse
	ErrLastAdmin             = errs.ErrLastAdmin
	ErrUnknownPermission     = errs.ErrUnknownPermission
	ErrPermissionRowMismatch = errs.ErrPermissionRowMismatch
)

type Role struct {
	ID          int64
	Name        string
	Permissions []string
	// MemberCount is filled by the listing query. Counting in SQL keeps the
	// roles screen from fetching every staff row to total them in the browser.
	MemberCount int
}

// DefaultPageSize is the page a caller gets when it asks for no particular size.
const DefaultPageSize = 20

// ListQuery carries a listing request; a zero PageSize means the default.
type ListQuery struct {
	Page     int
	PageSize int
}

// Page is List's response shape.
type Page struct {
	Items    []Role
	Page     int
	PageSize int
	Total    int
}

// CreateRole is Create's input.
type CreateRole struct {
	Name        string
	Permissions []string
}

// UpdateRole is Update's input. A nil field means "leave unchanged"; a
// non-nil Permissions field replaces the whole set, including an empty one.
type UpdateRole struct {
	Name        *string
	Permissions *[]string
}
