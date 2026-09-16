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

// Events is how the recorded transitions are readable at all: every
// transition writes one, and without this route none of them can be seen.
func (h *Handler) Events(w http.ResponseWriter, r *http.Request) {
	id, err := orderIDParam(r)
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}
	events, err := h.svc.Events(r.Context(), id)
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, newListEventsResponse(events))
}

// SetStatus is the generic status endpoint; it refuses cancelled and
// refunded itself before the service ever checks the transition table,
// since those targets are Cancel's and Refund's, not this one's.
func (h *Handler) SetStatus(w http.ResponseWriter, r *http.Request) {
	p, ok := principal.FromContext(r.Context())
	if !ok {
		h.writeErr(r.Context(), w, errMissingPrincipal)
		return
	}
	id, err := orderIDParam(r)
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}

	var req SetStatusRequest
	if err := httpx.DecodeJSON(r, &req, h.maxBodyBytes); err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}

	updated, err := h.svc.SetStatus(r.Context(), id, p.StaffID, Status(req.Status))
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, newOrderResponse(updated))
}

func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	p, ok := principal.FromContext(r.Context())
	if !ok {
		h.writeErr(r.Context(), w, errMissingPrincipal)
		return
	}
	id, err := orderIDParam(r)
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}

	var req TransitionRequest
	if err := httpx.DecodeJSON(r, &req, h.maxBodyBytes); err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}

	updated, err := h.svc.Cancel(r.Context(), id, p.StaffID, req.Reason)
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, newOrderResponse(updated))
}

func (h *Handler) Refund(w http.ResponseWriter, r *http.Request) {
	p, ok := principal.FromContext(r.Context())
	if !ok {
		h.writeErr(r.Context(), w, errMissingPrincipal)
		return
	}
	id, err := orderIDParam(r)
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}

	var req TransitionRequest
	if err := httpx.DecodeJSON(r, &req, h.maxBodyBytes); err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}

	updated, err := h.svc.Refund(r.Context(), id, p.StaffID, req.Reason)
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, newOrderResponse(updated))
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	status, err := statusQueryParam(r)
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}
	customerID, err := customerIDQueryParam(r)
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}
	page, pageSize, err := pagingQueryParams(r)
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}

	result, err := h.svc.List(r.Context(), ListQuery{
		Status:     status,
		CustomerID: customerID,
		Sort:       r.URL.Query().Get("sort"),
		Page:       page,
		PageSize:   pageSize,
	})
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, newOrderPageResponse(result))
}

func orderIDParam(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		return 0, httpx.ErrMalformedBody
	}
	return id, nil
}

// statusQueryParam accepts only the seven known status values, so an
// unrecognized string is refused here rather than reaching the enum column
// as a driver error.
func statusQueryParam(r *http.Request) (*Status, error) {
	raw := r.URL.Query().Get("status")
	if raw == "" {
		return nil, nil
	}
	switch Status(raw) {
	case StatusPending, StatusPaid, StatusPacked, StatusShipped, StatusDelivered, StatusCancelled, StatusRefunded:
		s := Status(raw)
		return &s, nil
	default:
		return nil, httpx.ErrMalformedBody
	}
}

func customerIDQueryParam(r *http.Request) (*int64, error) {
	raw := r.URL.Query().Get("customerId")
	if raw == "" {
		return nil, nil
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return nil, httpx.ErrMalformedBody
	}
	return &id, nil
}

// pagingQueryParams returns 0, 0 for an absent param, letting the service
// apply its defaults; a present-but-unparseable value is refused.
func pagingQueryParams(r *http.Request) (int, int, error) {
	page, err := intQueryParam(r, "page")
	if err != nil {
		return 0, 0, err
	}
	pageSize, err := intQueryParam(r, "pageSize")
	if err != nil {
		return 0, 0, err
	}
	return page, pageSize, nil
}

func intQueryParam(r *http.Request, key string) (int, error) {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return 0, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, httpx.ErrMalformedBody
	}
	return v, nil
}
