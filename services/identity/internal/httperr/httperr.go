// Package httperr maps this service's sentinel errors to client-facing
// responses. It is the only place in the service that decides a status code or
// a client-visible message.
package httperr

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/loloDawit/go-admin/platform/httpx"
	"github.com/loloDawit/go-admin/services/identity/internal/platformcheck"
)

func Write(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, platformcheck.ErrDirtySchema):
		httpx.WriteError(w, http.StatusServiceUnavailable, "schema_dirty", "the service is not ready")
	case errors.Is(err, platformcheck.ErrNoMigrations):
		httpx.WriteError(w, http.StatusServiceUnavailable, "schema_not_migrated", "the service is not ready")
	default:
		// The cause reaches the log; the client gets none of it.
		slog.Error("unmapped error", slog.String("error", err.Error()))
		httpx.WriteError(w, http.StatusServiceUnavailable, "database_unavailable", "the service is not ready")
	}
}
