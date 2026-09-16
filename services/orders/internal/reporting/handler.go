package reporting

import (
	"context"
	"net/http"

	"github.com/loloDawit/go-admin/platform/httpx"
)

type Handler struct {
	svc      *Service
	writeErr func(context.Context, http.ResponseWriter, error)
}

func NewHandler(svc *Service, writeErr func(context.Context, http.ResponseWriter, error)) *Handler {
	return &Handler{svc: svc, writeErr: writeErr}
}

func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	report, err := h.svc.Dashboard(r.Context())
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, newDashboardResponse(report))
}
