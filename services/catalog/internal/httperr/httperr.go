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
	"github.com/loloDawit/go-admin/services/catalog/internal/errs"
	"github.com/loloDawit/go-admin/services/catalog/internal/image"
	"github.com/loloDawit/go-admin/services/catalog/internal/platformcheck"
	"github.com/loloDawit/go-admin/services/catalog/internal/product"
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
	case errors.Is(err, httpx.ErrMalformedBody):
		httpx.WriteError(w, http.StatusBadRequest, "malformed_body", "the request body is invalid")
	case errors.Is(err, errs.ErrUnauthenticated):
		httpx.WriteError(w, http.StatusUnauthorized, "unauthenticated", "sign in to continue")
	case errors.Is(err, principal.ErrMissing), errors.Is(err, principal.ErrBadSignature), errors.Is(err, principal.ErrExpired):
		httpx.WriteError(w, http.StatusUnauthorized, "unauthenticated", "sign in to continue")
	case errors.Is(err, errs.ErrForbidden):
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "you do not have permission to perform this action")
	case errors.Is(err, product.ErrProductNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "no matching product was found")
	case errors.Is(err, product.ErrSkuTaken):
		httpx.WriteError(w, http.StatusConflict, "sku_taken", "that sku is already in use")
	case errors.Is(err, product.ErrProductArchived):
		httpx.WriteError(w, http.StatusConflict, "product_archived", "an archived product cannot be edited")
	case errors.Is(err, product.ErrInvalidPrice):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_failed", "price must not be negative")
	case errors.Is(err, product.ErrInvalidSort):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_failed", "sort is not supported")
	case errors.Is(err, product.ErrEmptySearchQuery):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_failed", "q must not be empty")
	case errors.Is(err, product.ErrResolveBatchTooLarge):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_failed", "too many ids requested")
	case errors.Is(err, image.ErrImageNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "no matching image was found")
	case errors.Is(err, image.ErrUnsupportedImageType):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "unsupported_image_type", "that image type is not supported")
	case errors.Is(err, image.ErrImageTooLarge):
		httpx.WriteError(w, http.StatusRequestEntityTooLarge, "image_too_large", "the image exceeds the maximum upload size")
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
