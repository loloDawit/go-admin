package role

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
	var req CreateRoleRequest
	if err := httpx.DecodeJSON(r, &req, h.maxBodyBytes); err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}
	created, err := h.svc.Create(r.Context(), CreateRole{Name: req.Name, Permissions: req.Permissions})
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, newRoleResponse(created))
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	page, err := h.svc.List(r.Context(), ListQuery{
		Page:     positiveIntParam(r, "page"),
		PageSize: positiveIntParam(r, "pageSize"),
	})
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, newRolePageResponse(page))
}

// A value that is absent, unparsable or not positive means "unset", which the
// service turns into its default; a bad page is not worth a 400.
func positiveIntParam(r *http.Request, key string) int {
	v, err := strconv.Atoi(r.URL.Query().Get(key))
	if err != nil || v < 1 {
		return 0
	}
	return v
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := roleIDParam(r)
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}
	got, err := h.svc.Get(r.Context(), id)
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, newRoleResponse(got))
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := roleIDParam(r)
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}
	var req UpdateRoleRequest
	if err := httpx.DecodeJSON(r, &req, h.maxBodyBytes); err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}
	updated, err := h.svc.Update(r.Context(), id, UpdateRole{Name: req.Name, Permissions: req.Permissions})
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, newRoleResponse(updated))
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := roleIDParam(r)
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}
	if err := h.svc.Delete(r.Context(), id); err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func roleIDParam(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		return 0, httpx.ErrMalformedBody
	}
	return id, nil
}
