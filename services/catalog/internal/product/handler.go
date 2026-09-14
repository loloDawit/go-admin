package product

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
	var req CreateProductRequest
	if err := httpx.DecodeJSON(r, &req, h.maxBodyBytes); err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}

	created, err := h.svc.Create(r.Context(), CreateProduct{
		SKU:         req.SKU,
		Title:       req.Title,
		Description: req.Description,
		PriceMinor:  req.PriceMinor,
		Currency:    req.Currency,
	})
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, newProductResponse(created))
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := productIDParam(r)
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}
	got, err := h.svc.Get(r.Context(), id)
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, newProductResponse(got))
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := productIDParam(r)
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}
	var req UpdateProductRequest
	if err := httpx.DecodeJSON(r, &req, h.maxBodyBytes); err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}

	updated, err := h.svc.Update(r.Context(), id, UpdateProduct{
		Title:       req.Title,
		Description: req.Description,
		PriceMinor:  req.PriceMinor,
		Currency:    req.Currency,
	})
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, newProductResponse(updated))
}

func (h *Handler) Archive(w http.ResponseWriter, r *http.Request) {
	id, err := productIDParam(r)
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}
	archived, err := h.svc.Archive(r.Context(), id)
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, newProductResponse(archived))
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	status, err := statusQueryParam(r)
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
		Status:   status,
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

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	status, err := statusQueryParam(r)
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}
	page, pageSize, err := pagingQueryParams(r)
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}

	result, err := h.svc.Search(r.Context(), SearchQuery{
		Text:     r.URL.Query().Get("q"),
		Status:   status,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, newPageResponse(result))
}

func productIDParam(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		return 0, httpx.ErrMalformedBody
	}
	return id, nil
}

// statusQueryParam accepts only the three known status values, so an
// unrecognized string is refused here rather than reaching the enum column
// as a driver error.
func statusQueryParam(r *http.Request) (*Status, error) {
	raw := r.URL.Query().Get("status")
	if raw == "" {
		return nil, nil
	}
	switch Status(raw) {
	case StatusDraft, StatusActive, StatusArchived:
		s := Status(raw)
		return &s, nil
	default:
		return nil, httpx.ErrMalformedBody
	}
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
