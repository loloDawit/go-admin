package staff

import "context"

// Repository is the data access this package needs; every error not named below is a driver-level failure the service wraps.
type Repository interface {
	Create(ctx context.Context, in CreateStaff, passwordHash string) (Staff, error)
	Update(ctx context.Context, id int64, in UpdateStaff) (Staff, error)
	GetByID(ctx context.Context, id int64) (Staff, error)
	List(ctx context.Context, q ListQuery) ([]Staff, int, error)

	// PasswordHash returns the stored bcrypt hash for id, never exposed through Staff itself.
	PasswordHash(ctx context.Context, id int64) (string, error)

	// SetPasswordHash replaces the stored hash and clears must_change_password.
	SetPasswordHash(ctx context.Context, id int64, hash string) error

	// HasEditStaffPermission reports whether roleID currently carries edit_staff, independent of the role's name.
	HasEditStaffPermission(ctx context.Context, roleID int64) (bool, error)

	// LockActiveEditStaffExcluding locks the active edit_staff holders for the
	// caller's transaction and counts those other than excludeID.
	LockActiveEditStaffExcluding(ctx context.Context, excludeID int64) (int, error)

	// RevokeSessions closes out a staff member's live session rows.
	RevokeSessions(ctx context.Context, staffID int64) error

	// RunInTx runs fn against a repository bound to a single transaction.
	RunInTx(ctx context.Context, fn func(Repository) error) error
}
