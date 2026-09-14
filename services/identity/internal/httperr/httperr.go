// Package httperr is the only place in the service that decides a
// client-facing status code or message.
package httperr

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/loloDawit/go-admin/platform/httpx"
	"github.com/loloDawit/go-admin/platform/principal"
	"github.com/loloDawit/go-admin/platform/requestid"
	"github.com/loloDawit/go-admin/services/identity/internal/errs"
	"github.com/loloDawit/go-admin/services/identity/internal/schemacheck"
	"github.com/loloDawit/go-admin/services/identity/internal/session"
)

// Writer's logger is injected, not read off the slog default, so log lines are attributed to this service.
type Writer struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) *Writer {
	return &Writer{logger: logger}
}

// Write takes ctx so the logged error carries the same request_id as the request line RequestLogger emits.
func (h *Writer) Write(ctx context.Context, w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, session.ErrInvalidCredentials):
		httpx.WriteError(w, http.StatusUnauthorized, "invalid_credentials", "email or password is incorrect")
	case errors.Is(err, session.ErrUnauthenticated):
		httpx.WriteError(w, http.StatusUnauthorized, "unauthenticated", "sign in to continue")
	case errors.Is(err, session.ErrPasswordChangeRequired):
		httpx.WriteError(w, http.StatusForbidden, "password_change_required", "a password change is required before continuing")
	case errors.Is(err, principal.ErrMissing), errors.Is(err, principal.ErrBadSignature), errors.Is(err, principal.ErrExpired):
		httpx.WriteError(w, http.StatusUnauthorized, "unauthenticated", "sign in to continue")
	case errors.Is(err, httpx.ErrMalformedBody):
		httpx.WriteError(w, http.StatusBadRequest, "malformed_body", "the request body is invalid")
	case errors.Is(err, errs.ErrCurrentPasswordIncorrect):
		httpx.WriteError(w, http.StatusUnauthorized, "invalid_credentials", "current password is incorrect")
	case errors.Is(err, errs.ErrEmailTaken):
		httpx.WriteError(w, http.StatusConflict, "email_taken", "that email is already in use")
	case errors.Is(err, errs.ErrLastAdmin):
		httpx.WriteError(w, http.StatusConflict, "last_admin", "cannot remove the last active admin")
	case errors.Is(err, errs.ErrForbidden):
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "you do not have permission to perform this action")
	case errors.Is(err, errs.ErrRoleNameTaken):
		httpx.WriteError(w, http.StatusConflict, "name_taken", "that role name is already in use")
	case errors.Is(err, errs.ErrRoleInUse):
		httpx.WriteError(w, http.StatusConflict, "role_in_use", "role is assigned to staff and cannot be deleted")
	case errors.Is(err, errs.ErrUnknownPermission):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_failed", "unknown permission")
	case errors.Is(err, errs.ErrPermissionRowMismatch):
		// A valid permission name matching no permissions row is a deployment defect, not a client mistake; log the cause, tell the client nothing.
		h.logger.ErrorContext(ctx, "permission row mismatch",
			slog.String("request_id", requestid.FromContext(ctx)),
			slog.String("error", err.Error()),
		)
		httpx.WriteError(w, http.StatusInternalServerError, "internal", "something went wrong")
	case errors.Is(err, errs.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "no matching record was found")
	case errors.Is(err, errs.ErrEmptyPassword):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_failed", "password must not be empty")
	case errors.Is(err, schemacheck.ErrDirtySchema):
		httpx.WriteError(w, http.StatusServiceUnavailable, "schema_dirty", "the service is not ready")
	case errors.Is(err, schemacheck.ErrNoMigrations):
		httpx.WriteError(w, http.StatusServiceUnavailable, "schema_not_migrated", "the service is not ready")
	case errors.Is(err, schemacheck.ErrDatabaseUnavailable):
		// The only place the driver cause reaches the log before the client gets the generic message.
		h.logger.ErrorContext(ctx, "database unavailable",
			slog.String("request_id", requestid.FromContext(ctx)),
			slog.String("error", err.Error()),
		)
		httpx.WriteError(w, http.StatusServiceUnavailable, "database_unavailable", "the service is not ready")
	default:
		h.logger.ErrorContext(ctx, "unmapped error",
			slog.String("request_id", requestid.FromContext(ctx)),
			slog.String("error", err.Error()),
		)
		httpx.WriteError(w, http.StatusInternalServerError, "internal", "something went wrong")
	}
}
