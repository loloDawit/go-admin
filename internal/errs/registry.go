package errs

import "net/http"

// Declare new errors here. Codes must be unique; registry_test.go enforces it.

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
	// Covers both an unknown email and a wrong password. Splitting these
	// makes login a user-enumeration oracle.
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

	ResourceInUse = define(http.StatusConflict, "resource_in_use",
		"this resource cannot be deleted because other records depend on it")
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
	// Message stays vague; Wrap the driver error so schema details reach the
	// log instead of the client.
	Database = define(http.StatusInternalServerError, "database_error",
		"a database error occurred")

	ExportFailed = define(http.StatusInternalServerError, "export_failed",
		"the export could not be generated")

	Internal = define(http.StatusInternalServerError, "internal_error",
		"an unexpected error occurred")
)

// All backs the registry invariant tests.
var All = []*Error{
	InvalidBody, InvalidID, ValidationFailed, MissingField,
	NameRequired, EmailRequired, EmailInvalid, RoleRequired,
	TitleRequired, PriceInvalid,
	InvalidCredentials, Unauthenticated, SessionInvalid,
	PasswordEmpty, PasswordMismatch, TokenIssueFailed,
	Forbidden, NoRoleAssigned,
	NotFound, AlreadyExists, EmailTaken, ResourceInUse,
	UploadMalformed, UploadTooLarge, UploadUnsupportedType, UploadFailed,
	Database, ExportFailed, Internal,
}
