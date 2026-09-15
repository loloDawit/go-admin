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
	"github.com/loloDawit/go-admin/services/orders/internal/customer"
	"github.com/loloDawit/go-admin/services/orders/internal/errs"
	"github.com/loloDawit/go-admin/services/orders/internal/order"
	"github.com/loloDawit/go-admin/services/orders/internal/platformcheck"
)

// statusClientClosedRequest mirrors nginx's 499: there is no standard HTTP
// status for a client that disconnected before an answer was ready.
const statusClientClosedRequest = 499

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
	case errors.Is(err, customer.ErrCustomerNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "customer not found")
	case errors.Is(err, customer.ErrCustomerEmailTaken):
		httpx.WriteError(w, http.StatusConflict, "email_taken", "email is already in use")
	case errors.Is(err, customer.ErrInvalidCustomerEmail):
		httpx.WriteError(w, http.StatusBadRequest, "validation_failed", "email must not be empty")
	case errors.Is(err, order.ErrOrderNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "order not found")
	case errors.Is(err, order.ErrProductUnavailable):
		// err.Error() here is just "product unavailable: product <id>", built
		// in service.go from caller-supplied ids; safe to show, unlike a
		// driver or SQL message.
		httpx.WriteError(w, http.StatusUnprocessableEntity, "product_unavailable", err.Error())
	case errors.Is(err, order.ErrCurrencyMismatch):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "currency_mismatch", "an order's lines must share one currency")
	case errors.Is(err, order.ErrEmptyOrder):
		httpx.WriteError(w, http.StatusBadRequest, "validation_failed", "an order must have at least one item")
	case errors.Is(err, order.ErrInvalidQuantity):
		httpx.WriteError(w, http.StatusBadRequest, "validation_failed", "order line quantity must be positive")
	case errors.Is(err, order.ErrInvalidTransition):
		httpx.WriteError(w, http.StatusConflict, "invalid_transition", "that status change is not allowed from the order's current status")
	case errors.Is(err, order.ErrInvalidSort):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_failed", "sort is not supported")
	case errors.Is(err, context.Canceled):
		// The client disconnected before Catalog answered; this is not a
		// dependency failure, so it must not join a 5xx error-rate metric
		// the way a real Catalog outage would, and it is not logged as one.
		httpx.WriteError(w, statusClientClosedRequest, "request_cancelled", "the request was cancelled")
	case errors.Is(err, errs.ErrCatalogTimeout):
		httpx.WriteError(w, http.StatusGatewayTimeout, "catalog_timeout", "the service is not ready")
	case errors.Is(err, errs.ErrCatalogUnavailable):
		httpx.WriteError(w, http.StatusBadGateway, "catalog_unavailable", "the service is not ready")
	case errors.Is(err, errs.ErrCatalogRejected):
		// Catalog rejected our own request: a bug on this side, not a client fault, hence 502 not 4xx.
		httpx.WriteError(w, http.StatusBadGateway, "catalog_rejected", "something went wrong")
	default:
		h.logger.ErrorContext(ctx, "unmapped error",
			slog.String("request_id", requestid.FromContext(ctx)),
			slog.String("error", err.Error()),
		)
		httpx.WriteError(w, http.StatusInternalServerError, "internal", "something went wrong")
	}
}
