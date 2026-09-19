// Package httperr owns the gateway's client-facing errors. Transport failures
// name upstream hosts; nothing from them reaches a response body.
package httperr

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/loloDawit/go-admin/platform/httpx"
	"github.com/loloDawit/go-admin/platform/requestid"
)

func WriteUnknownRoute(w http.ResponseWriter) {
	httpx.WriteError(w, http.StatusNotFound, "not_found", "no such route")
}

func WriteUpstreamUnavailable(w http.ResponseWriter) {
	httpx.WriteError(w, http.StatusBadGateway, "upstream_unavailable", "the service is temporarily unavailable")
}

func WriteGatewayTimeout(w http.ResponseWriter) {
	httpx.WriteError(w, http.StatusGatewayTimeout, "gateway_timeout", "the service did not respond in time")
}

// Retry-After is seconds, and one is the smallest honest answer: the bucket
// refills continuously rather than at a fixed instant.
func WriteRateLimited(w http.ResponseWriter) {
	w.Header().Set("Retry-After", "1")
	httpx.WriteError(w, http.StatusTooManyRequests, "rate_limited", "too many requests; try again shortly")
}

// Writer logs an unmapped error's cause before answering with the generic
// 500 envelope.
type Writer struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) *Writer {
	return &Writer{logger: logger}
}

func (h *Writer) Write(ctx context.Context, w http.ResponseWriter, err error) {
	h.logger.ErrorContext(ctx, "unmapped error",
		slog.String("request_id", requestid.FromContext(ctx)),
		slog.String("error", err.Error()),
	)
	httpx.WriteError(w, http.StatusInternalServerError, "internal", "something went wrong")
}
