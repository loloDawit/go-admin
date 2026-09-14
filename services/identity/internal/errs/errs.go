// Package errs is the identity service's one error registry: every sentinel
// httperr maps to a client-facing response, and every operation string a
// service-layer wrap names. A message written inline in a service file is one
// nobody can grep, audit, or keep consistent with its siblings; this package
// is where that consistency is enforced.
package errs

import (
	"errors"
	"fmt"
)

// Sentinels. httperr maps these to a status code; nothing else decides one.
var (
	// ErrInvalidCredentials covers every reason a login attempt fails: an
	// unknown email, a wrong password, or a deactivated account. They share
	// one sentinel deliberately — a caller that could tell them apart could
	// enumerate which email addresses have accounts.
	ErrInvalidCredentials = errors.New("email or password is incorrect")

	// ErrUnauthenticated is returned when a token does not resolve to a live
	// session: unknown, revoked, or expired.
	ErrUnauthenticated = errors.New("session is not authenticated")

	// ErrPasswordChangeRequired is part of the session package's published
	// error contract; the enforcement middleware that returns it belongs to
	// the staff package (must_change_password is a staff concern), added
	// later.
	ErrPasswordChangeRequired = errors.New("password change is required")

	// ErrNotFound is returned by a session.Repository when a lookup matches
	// no row: an unknown email, an unknown staff ID, or a token with no live
	// session.
	ErrNotFound = errors.New("no matching row")

	// ErrEmptyPassword is returned when Hasher.Hash is called with an empty
	// string. bcrypt itself accepts an empty password; rejecting it here
	// keeps a blank password from ever reaching a stored hash.
	ErrEmptyPassword = errors.New("password must not be empty")

	ErrMissingDatabaseURL   = errors.New("DATABASE_URL is required")
	ErrMissingOwnerEmail    = errors.New("OWNER_EMAIL is required")
	ErrMissingOwnerPassword = errors.New("OWNER_PASSWORD is required")

	// ErrNoAdminRole means migrations have not run: 000002 seeds the role.
	ErrNoAdminRole = errors.New("no admin role exists")

	// ErrNoMigrations, ErrDirtySchema, and ErrDatabaseUnavailable are
	// platformcheck's sentinels (the M1 walking skeleton's schema-state
	// probe); httperr maps all three to a 503.
	ErrNoMigrations        = errors.New("no migrations applied")
	ErrDirtySchema         = errors.New("schema is in a dirty state")
	ErrDatabaseUnavailable = errors.New("database is unavailable")

	// ErrEmailTaken is returned when a staff email collides with an existing
	// row's unique constraint.
	ErrEmailTaken = errors.New("email is already in use")

	// ErrLastAdmin is returned when a change would leave no active staff
	// member holding edit_staff: deactivating the last such member, or
	// reassigning their role or active flag away from that permission. The
	// check is by permission, not by a role literally named "admin".
	ErrLastAdmin = errors.New("cannot remove the last active admin")

	// ErrCurrentPasswordIncorrect is returned by staff.ChangePassword when
	// the supplied current password does not match the stored hash.
	ErrCurrentPasswordIncorrect = errors.New("current password is incorrect")

	// ErrForbidden is returned by authz.Require when a verified principal
	// does not carry the permission a route requires.
	ErrForbidden = errors.New("caller lacks the required permission")

	// ErrRoleNameTaken is returned when a role name collides with an
	// existing row's unique constraint.
	ErrRoleNameTaken = errors.New("role name is already in use")

	// ErrRoleInUse is returned when a role cannot be deleted because at
	// least one staff row still references it.
	ErrRoleInUse = errors.New("role is assigned to staff and cannot be deleted")

	// ErrUnknownPermission is returned when a role request names a
	// permission outside permission.All().
	ErrUnknownPermission = errors.New("unknown permission")

	// ErrPermissionRowMismatch means a name valid per permission.All() matched no row in permissions: a deployment inconsistency, not a client mistake.
	ErrPermissionRowMismatch = errors.New("a permission name matched no row in permissions")
)

// Operations name the step that failed. Wrap puts one of these ahead of the
// underlying error; they reach the log, never the client.
const (
	OpBuildDummyLoginHash  = "build dummy login hash"
	OpLookupStaffByEmail   = "look up staff by email"
	OpLookupStaffByID      = "look up staff by id"
	OpLookupSession        = "look up session"
	OpCreateSession        = "create session"
	OpRevokeSession        = "revoke session"
	OpGenerateSessionToken = "generate session token"

	OpReadBcryptCost      = "read BCRYPT_COST"
	OpConnectDatabase     = "connect to the database"
	OpCountAdminRole      = "count the admin role"
	OpHashOwnerPassword   = "hash the owner password"
	OpInsertOwner         = "insert the owner"
	OpRevokeStaffSessions = "revoke the staff member's sessions"

	OpGenerateStaffPassword = "generate staff password"
	OpHashStaffPassword     = "hash staff password"
	OpCreateStaff           = "create staff"
	OpUpdateStaff           = "update staff"
	OpListStaff             = "list staff"
	OpChangeStaffPassword   = "change staff password"
	OpCheckAdminRole        = "check edit_staff permission"
	OpCountActiveAdmins     = "count other active edit_staff holders"

	OpCreateRole               = "create role"
	OpUpdateRole               = "update role"
	OpDeleteRole               = "delete role"
	OpListRoles                = "list roles"
	OpLookupRoleByID           = "look up role by id"
	OpCheckRoleEditStaff       = "check role edit_staff permission"
	OpCountActiveAdminsOutside = "count active edit_staff holders outside role"
)

// Wrap names the step that failed ahead of the underlying error, so a log
// line reads as "look up staff by email: connection refused" rather than a
// bare driver message with no indication of what the service was doing.
func Wrap(op string, err error) error {
	return fmt.Errorf("%s: %w", op, err)
}
