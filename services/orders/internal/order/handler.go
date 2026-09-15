package order

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/loloDawit/go-admin/platform/httpx"
	"github.com/loloDawit/go-admin/platform/principal"
)

// errMissingPrincipal signals a route wired without principal.Middleware
// ahead of it: a wiring bug, never a client fault.
var errMissingPrincipal = errors.New("no verified principal on request context")

type Handler struct {
	svc          *Service
	maxBodyBytes int64
	writeErr     func(context.Context, http.ResponseWriter, error)
}

func NewHandler(svc *Service, maxBodyBytes int64, writeErr func(context.Context, http.ResponseWriter, error)) *Handler {
	return &Handler{svc: svc, maxBodyBytes: maxBodyBytes, writeErr: writeErr}
}

// Create's actor comes from the verified principal, never the request body.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	p, ok := principal.FromContext(r.Context())
	if !ok {
		h.writeErr(r.Context(), w, errMissingPrincipal)
		return
	}

	var req CreateOrderRequest
	if err := httpx.DecodeJSON(r, &req, h.maxBodyBytes); err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}

	customerID, err := strconv.ParseInt(req.CustomerID, 10, 64)
	if err != nil {
		h.writeErr(r.Context(), w, httpx.ErrMalformedBody)
		return
	}

	items := make([]CreateOrderItem, len(req.Items))
	for i, it := range req.Items {
		items[i] = CreateOrderItem{ProductID: it.ProductID, Quantity: it.Quantity}
	}

	created, err := h.svc.Create(r.Context(), CreateOrder{
		CustomerID: customerID,
		ActorID:    p.StaffID,
		Items:      items,
	})
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, newOrderResponse(created))
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := orderIDParam(r)
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}
	got, err := h.svc.Get(r.Context(), id)
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, newOrderResponse(got))
}

func orderIDParam(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		return 0, httpx.ErrMalformedBody
	}
	return id, nil
}
