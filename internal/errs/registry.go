package errs

import "net/http"

// The complete set of application errors.
//
// Adding one: declare it here, in the group it belongs to, with a code that
// is unique across the whole file (registry_test.go enforces uniqueness).

// --- Request parsing and validation -------------------------------------

var (
	InvalidBody = define(http.StatusBadRequest, "invalid_body",
		"the request body could not be parsed")

	InvalidID = define(http.StatusBadRequest, "invalid_id",
		"id must be a positive integer")

	ValidationFailed = define(http.StatusBadRequest, "validation_failed",
		"the request is not valid")

	MissingField = define(http.StatusBadRequest, "missing_field",
		"a required field is missing")
)

// --- Field-level validation ---------------------------------------------
//
// These are named rather than folded into ValidationFailed so that a client
// can react to a specific field problem, and so that no layer needs to write
// the message inline. Domain code (models) returns these directly; handlers
// pass them straight to httpx.Fail.

var (
	NameRequired = define(http.StatusBadRequest, "name_required",
		"first and last name are required")

	EmailRequired = define(http.StatusBadRequest, "email_required",
		"email is required")

	EmailInvalid = define(http.StatusBadRequest, "email_invalid",
		"that email address is not valid")

	RoleRequired = define(http.StatusBadRequest, "role_required",
		"a role is required")

	TitleRequired = define(http.StatusBadRequest, "title_required",
		"title is required")

	PriceInvalid = define(http.StatusBadRequest, "price_invalid",
		"price must not be negative")
)

// --- Authentication ------------------------------------------------------

var (
	// InvalidCredentials is returned for BOTH an unknown email and a wrong
	// password. Distinguishing them turns the login endpoint into a user
	// enumeration oracle (ASSESSMENT 4i), so there is deliberately no
	// separate "user not found" error for the login path.
	InvalidCredentials = define(http.StatusUnauthorized, "invalid_credentials",
		"email or password is incorrect")

	Unauthenticated = define(http.StatusUnauthorized, "unauthenticated",
		"you are not signed in")

	SessionInvalid = define(http.StatusUnauthorized, "session_invalid",
		"your session is no longer valid; please sign in again")

	PasswordEmpty = define(http.StatusBadRequest, "password_empty",
		"password must not be empty")

	PasswordMismatch = define(http.StatusBadRequest, "password_mismatch",
		"the passwords do not match")

	TokenIssueFailed = define(http.StatusInternalServerError, "token_issue_failed",
		"could not issue a session")
)

// --- Authorization -------------------------------------------------------

var (
	Forbidden = define(http.StatusForbidden, "forbidden",
		"you do not have permission to perform this action")

	NoRoleAssigned = define(http.StatusForbidden, "no_role_assigned",
		"your account has no role assigned; contact an administrator")
)

// --- Resources -----------------------------------------------------------

var (
	NotFound = define(http.StatusNotFound, "not_found",
		"the requested resource was not found")

	AlreadyExists = define(http.StatusConflict, "already_exists",
		"a resource with those details already exists")

	EmailTaken = define(http.StatusConflict, "email_taken",
		"that email address is already in use")
)

// --- Uploads -------------------------------------------------------------

var (
	UploadMalformed = define(http.StatusBadRequest, "upload_malformed",
		"attach exactly one file under the field \"image\"")

	UploadTooLarge = define(http.StatusRequestEntityTooLarge, "upload_too_large",
		"the file exceeds the maximum upload size")

	UploadUnsupportedType = define(http.StatusBadRequest, "upload_unsupported_type",
		"only JPEG, PNG, GIF, and WebP images are accepted")

	UploadFailed = define(http.StatusInternalServerError, "upload_failed",
		"the file could not be stored")
)

// --- Infrastructure ------------------------------------------------------

var (
	// Database is for query failures. Its Message is deliberately vague: the
	// driver's own text can disclose schema and must never reach a client.
	// Wrap the driver error so it reaches the logs instead.
	Database = define(http.StatusInternalServerError, "database_error",
		"a database error occurred")

	ExportFailed = define(http.StatusInternalServerError, "export_failed",
		"the export could not be generated")

	Internal = define(http.StatusInternalServerError, "internal_error",
		"an unexpected error occurred")
)

// All is every registered error. It exists so tests can assert invariants
// across the whole set, and so the catalogue can be dumped for the frontend.
var All = []*Error{
	InvalidBody, InvalidID, ValidationFailed, MissingField,
	NameRequired, EmailRequired, EmailInvalid, RoleRequired,
	TitleRequired, PriceInvalid,
	InvalidCredentials, Unauthenticated, SessionInvalid,
	PasswordEmpty, PasswordMismatch, TokenIssueFailed,
	Forbidden, NoRoleAssigned,
	NotFound, AlreadyExists, EmailTaken,
	UploadMalformed, UploadTooLarge, UploadUnsupportedType, UploadFailed,
	Database, ExportFailed, Internal,
}
