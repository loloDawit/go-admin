package readiness

import (
	"context"
	"net/http"
)

// Handler runs Probe once per request; nothing is cached, so an outage is always reported fresh.
type Handler struct {
	probe    Probe
	writeErr func(ctx context.Context, w http.ResponseWriter, err error)
}

func NewHandler(probe Probe, writeErr func(context.Context, http.ResponseWriter, error)) *Handler {
	return &Handler{probe: probe, writeErr: writeErr}
}

// Ready writes only a status code on success, or the standard error envelope
// via writeErr on failure — the same as any other client-facing error.
func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	if err := h.probe(r.Context()); err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}
