// Package httperr is the only place in the service that decides a
// client-facing status code or message.
package httperr

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/loloDawit/go-admin/platform/httpx"
	"github.com/loloDawit/go-admin/platform/requestid"
	"github.com/loloDawit/go-admin/services/orders/internal/platformcheck"
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
	case errors.Is(err, platformcheck.ErrDirtySchema):
		httpx.WriteError(w, http.StatusServiceUnavailable, "schema_dirty", "the service is not ready")
	case errors.Is(err, platformcheck.ErrNoMigrations):
		httpx.WriteError(w, http.StatusServiceUnavailable, "schema_not_migrated", "the service is not ready")
	case errors.Is(err, platformcheck.ErrDatabaseUnavailable):
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
