// Package errs is the identity service's one error registry: every operation
// string a wrap names lives here. Most sentinels below are mapped by
// httperr to a client-facing status; the ones marked startup-only are not,
// since they can only ever occur before the server starts serving requests.
package errs

import (
	"errors"
	"fmt"
)

// Sentinels. Where httperr maps one to a status code, nothing else decides one.
var (
	// ErrInvalidCredentials covers an unknown email, wrong password, or deactivated account under one sentinel, so a caller cannot enumerate accounts.
	ErrInvalidCredentials = errors.New("email or password is incorrect")

	ErrUnauthenticated = errors.New("session is not authenticated")

	// ErrPasswordChangeRequired is part of session's contract; the staff package's middleware is what returns it.
	ErrPasswordChangeRequired = errors.New("password change is required")

	ErrNotFound = errors.New("no matching row")

	// ErrEmptyPassword: bcrypt itself accepts an empty password, so this rejects it before a blank password reaches a stored hash.
	ErrEmptyPassword = errors.New("password must not be empty")

	// Startup-only: raised before the server accepts any request, so httperr has no case for these.
	ErrMissingDatabaseURL   = errors.New("DATABASE_URL is required")
	ErrMissingOwnerEmail    = errors.New("OWNER_EMAIL is required")
	ErrMissingOwnerPassword = errors.New("OWNER_PASSWORD is required")

	// ErrNoAdminRole means migrations have not run: 000002 seeds the role. Startup-only, like the three above.
	ErrNoAdminRole = errors.New("no admin role exists")

	// httperr maps all three to 503: each means not ready, never a request fault.
	ErrNoMigrations        = errors.New("no migrations applied")
	ErrDirtySchema         = errors.New("schema is in a dirty state")
	ErrDatabaseUnavailable = errors.New("database is unavailable")

	ErrEmailTaken = errors.New("email is already in use")

	// ErrRoleNotFound: a staff create or update named a roleId no role row satisfies.
	ErrRoleNotFound = errors.New("role does not exist")

	// ErrLastAdmin: the check is by edit_staff permission, not by a role literally named "admin".
	ErrLastAdmin = errors.New("cannot remove the last active admin")

	ErrCurrentPasswordIncorrect = errors.New("current password is incorrect")

	ErrForbidden = errors.New("caller lacks the required permission")

	ErrRoleNameTaken = errors.New("role name is already in use")

	ErrRoleInUse = errors.New("role is assigned to staff and cannot be deleted")

	// ErrUnknownPermission: a role request named a permission outside permission.All().
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

// Wrap names the step that failed, so a log line reads "look up staff by email: connection refused", not a bare driver message.
func Wrap(op string, err error) error {
	return fmt.Errorf("%s: %w", op, err)
}
