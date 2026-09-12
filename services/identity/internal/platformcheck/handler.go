package platformcheck

import (
	"net/http"

	"github.com/loloDawit/go-admin/platform/httpx"
	"github.com/loloDawit/go-admin/platform/requestid"
)

// writeErr is injected rather than imported: internal/httperr must import this
// package for its sentinels, so importing it back would be a cycle.
type Handler struct {
	svc         *Service
	serviceName string
	writeErr    func(http.ResponseWriter, error)
}

func NewHandler(svc *Service, serviceName string, writeErr func(http.ResponseWriter, error)) *Handler {
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
		h.writeErr(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, response{
		Service:       h.serviceName,
		SchemaVersion: state.Version,
		RequestID:     requestid.FromContext(r.Context()),
	})
}

// Ready reports whether this process can serve. It checks the same conditions
// as Platform but returns no body.
func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	if _, err := h.svc.Check(r.Context()); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}
