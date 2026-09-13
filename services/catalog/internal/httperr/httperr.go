// Package httperr maps this service's sentinel errors to client-facing
// responses. It is the only place in the service that decides a status code or
// a client-visible message.
package httperr

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/loloDawit/go-admin/platform/httpx"
	"github.com/loloDawit/go-admin/platform/requestid"
	"github.com/loloDawit/go-admin/services/catalog/internal/platformcheck"
)

// Writer holds the logger an unmapped error's cause is written to. It is
// injected rather than read off the slog default so the log line can be
// attributed to this service the same way every other log line is.
type Writer struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) *Writer {
	return &Writer{logger: logger}
}

// Write takes ctx so the log line below can carry the same request_id as the
// request line RequestLogger emits for this request. Without it, an
// unmapped error logs an ERROR line with no way to correlate it to the
// request that triggered it, defeating the request-ID groundwork exactly
// when correlating errors to requests matters most.
func (h *Writer) Write(ctx context.Context, w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, platformcheck.ErrDirtySchema):
		httpx.WriteError(w, http.StatusServiceUnavailable, "schema_dirty", "the service is not ready")
	case errors.Is(err, platformcheck.ErrNoMigrations):
		httpx.WriteError(w, http.StatusServiceUnavailable, "schema_not_migrated", "the service is not ready")
	case errors.Is(err, platformcheck.ErrDatabaseUnavailable):
		// readiness.Handler.Ready has no logging of its own; this is the only
		// place the driver cause (kept reachable via %w in Service.Check)
		// reaches the log before the client gets the generic message.
		h.logger.ErrorContext(ctx, "database unavailable",
			slog.String("request_id", requestid.FromContext(ctx)),
			slog.String("error", err.Error()),
		)
		httpx.WriteError(w, http.StatusServiceUnavailable, "database_unavailable", "the service is not ready")
	default:
		// The cause reaches the log; the client gets none of it.
		h.logger.ErrorContext(ctx, "unmapped error",
			slog.String("request_id", requestid.FromContext(ctx)),
			slog.String("error", err.Error()),
		)
		httpx.WriteError(w, http.StatusServiceUnavailable, "database_unavailable", "the service is not ready")
	}
}
