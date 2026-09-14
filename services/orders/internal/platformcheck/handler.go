package platformcheck

import (
	"context"
	"net/http"

	"github.com/loloDawit/go-admin/platform/httpx"
	"github.com/loloDawit/go-admin/platform/requestid"
)

type Handler struct {
	svc         *Service
	serviceName string
	// writeErr is injected, not imported: internal/httperr imports this package for its sentinels, so importing it back would cycle.
	writeErr func(ctx context.Context, w http.ResponseWriter, err error)
}

func NewHandler(svc *Service, serviceName string, writeErr func(context.Context, http.ResponseWriter, error)) *Handler {
	return &Handler{svc: svc, serviceName: serviceName, writeErr: writeErr}
}

type response struct {
	Service       string `json:"service"`
	SchemaVersion int    `json:"schemaVersion"`
	RequestID     string `json:"requestId"`
}

func (h *Handler) Platform(w http.ResponseWriter, r *http.Request) {
	state, err := h.svc.Check(r.Context())
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, response{
		Service:       h.serviceName,
		SchemaVersion: state.Version,
		RequestID:     requestid.FromContext(r.Context()),
	})
}
