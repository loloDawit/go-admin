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
