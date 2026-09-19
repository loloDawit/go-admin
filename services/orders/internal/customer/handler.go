package customer

import (
	"context"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/loloDawit/go-admin/platform/httpx"
)

type Handler struct {
	svc          *Service
	maxBodyBytes int64
	writeErr     func(context.Context, http.ResponseWriter, error)
}

func NewHandler(svc *Service, maxBodyBytes int64, writeErr func(context.Context, http.ResponseWriter, error)) *Handler {
	return &Handler{svc: svc, maxBodyBytes: maxBodyBytes, writeErr: writeErr}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateCustomerRequest
	if err := httpx.DecodeJSON(r, &req, h.maxBodyBytes); err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}

	created, err := h.svc.Create(r.Context(), CreateCustomer{Email: req.Email, Name: req.Name})
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, newCustomerResponse(created))
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := customerIDParam(r)
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}
	got, err := h.svc.Get(r.Context(), id)
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, newCustomerResponse(got))
}

// ByEmail serves GET /api/v1/customers?email=...; a missing param is a
// malformed request, not an empty result.
func (h *Handler) ByEmail(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	if email == "" {
		h.writeErr(r.Context(), w, httpx.ErrMalformedBody)
		return
	}

	got, err := h.svc.ByEmail(r.Context(), email)
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, newCustomerResponse(got))
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	page, pageSize, err := pagingQueryParams(r)
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}

	result, err := h.svc.List(r.Context(), ListQuery{
		Q:        r.URL.Query().Get("q"),
		Sort:     r.URL.Query().Get("sort"),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, newPageResponse(result))
}

func (h *Handler) LifetimeValue(w http.ResponseWriter, r *http.Request) {
	id, err := customerIDParam(r)
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}
	total, currency, err := h.svc.LifetimeValue(r.Context(), id)
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, LifetimeValueResponse{LifetimeValueMinor: total, Currency: currency})
}

func (h *Handler) OrderHistory(w http.ResponseWriter, r *http.Request) {
	id, err := customerIDParam(r)
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}
	page, pageSize, err := pagingQueryParams(r)
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}

	result, err := h.svc.OrderHistory(r.Context(), id, OrderHistoryQuery{Page: page, PageSize: pageSize})
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, newOrderHistoryPageResponse(result))
}

func customerIDParam(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		return 0, httpx.ErrMalformedBody
	}
	return id, nil
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
