package readiness

import (
	"context"
	"net/http"
)

// Handler answers a readiness check by running Probe once per request: there
// is nothing cached, so a request made during an outage always gets a fresh
// answer instead of a stale one.
type Handler struct {
	probe    Probe
	writeErr func(ctx context.Context, w http.ResponseWriter, err error)
}

func NewHandler(probe Probe, writeErr func(context.Context, http.ResponseWriter, error)) *Handler {
	return &Handler{probe: probe, writeErr: writeErr}
}

// Ready reports whether this process can serve. On success it writes no
// body, only the status code; on failure it writes the standard error
// envelope via writeErr, the same as any other client-facing error. Nothing
// in this stack currently probes this route on an interval — the Docker
// HEALTHCHECK and compose's depends_on condition: service_healthy both probe
// /healthz instead — but it is meant for an operator checking readiness by
// hand today, and for a Kubernetes readiness probe once one exists.
func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	if err := h.probe(r.Context()); err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}
