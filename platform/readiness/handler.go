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

// Ready reports whether this process can serve. It returns no body: only the
// status code matters to its callers (Docker HEALTHCHECK, compose's
// depends_on condition: service_healthy, and eventually a k8s readiness
// probe).
func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	if err := h.probe(r.Context()); err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}
