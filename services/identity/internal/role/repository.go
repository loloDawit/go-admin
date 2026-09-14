package role

import "context"

// Repository is the data access this package needs; every error not named
// below is a driver-level failure the service wraps.
type Repository interface {
	Create(ctx context.Context, in CreateRole) (Role, error)
	Update(ctx context.Context, id int64, in UpdateRole) (Role, error)
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (Role, error)
	List(ctx context.Context) ([]Role, error)

	// HasEditStaffPermission reports whether roleID currently carries
	// edit_staff.
	HasEditStaffPermission(ctx context.Context, roleID int64) (bool, error)

	// CountActiveStaffWithEditStaffOutsideRole counts active staff who hold
	// edit_staff through a role other than roleID.
	CountActiveStaffWithEditStaffOutsideRole(ctx context.Context, roleID int64) (int, error)

	// LockActiveStaffWithEditStaffOutsideRole locks the active edit_staff
	// holders for the caller's transaction and counts those outside roleID.
	LockActiveStaffWithEditStaffOutsideRole(ctx context.Context, roleID int64) (int, error)

	// RunInTx runs fn against a repository bound to a single transaction.
	RunInTx(ctx context.Context, fn func(Repository) error) error
}
