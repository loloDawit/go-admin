package staff

import (
	"context"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/loloDawit/go-admin/platform/httpx"
	"github.com/loloDawit/go-admin/platform/principal"
	"github.com/loloDawit/go-admin/services/identity/internal/errs"
)

// mePasswordPath and logoutPath are the only routes RequirePasswordChanged lets through while a change is pending.
const (
	mePasswordPath = "/api/v1/me/password"
	logoutPath     = "/api/v1/logout"
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
	var req CreateStaffRequest
	if err := httpx.DecodeJSON(r, &req, h.maxBodyBytes); err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}
	roleID, err := strconv.ParseInt(req.RoleID, 10, 64)
	if err != nil {
		h.writeErr(r.Context(), w, httpx.ErrMalformedBody)
		return
	}

	created, password, err := h.svc.Create(r.Context(), CreateStaff{
		Email:     req.Email,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		RoleID:    roleID,
	})
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, CreateStaffResponse{Staff: newStaffResponse(created), Password: password})
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
	httpx.WriteJSON(w, http.StatusOK, newStaffPageResponse(page))
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
	id, err := staffIDParam(r)
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}
	got, err := h.svc.Get(r.Context(), id)
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, newStaffResponse(got))
}

// Update binds against SelfUpdateStaffRequest or AdminUpdateStaffRequest, decided by path ID vs. the verified principal's StaffID.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := staffIDParam(r)
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}
	p, ok := principal.FromContext(r.Context())
	if !ok {
		h.writeErr(r.Context(), w, errs.ErrUnauthenticated)
		return
	}

	var in UpdateStaff
	if strconv.FormatInt(id, 10) == p.StaffID {
		var req SelfUpdateStaffRequest
		if err := httpx.DecodeJSON(r, &req, h.maxBodyBytes); err != nil {
			h.writeErr(r.Context(), w, err)
			return
		}
		in = UpdateStaff{FirstName: req.FirstName, LastName: req.LastName, Email: req.Email}
	} else {
		var req AdminUpdateStaffRequest
		if err := httpx.DecodeJSON(r, &req, h.maxBodyBytes); err != nil {
			h.writeErr(r.Context(), w, err)
			return
		}
		var roleID *int64
		if req.RoleID != nil {
			parsed, err := strconv.ParseInt(*req.RoleID, 10, 64)
			if err != nil {
				h.writeErr(r.Context(), w, httpx.ErrMalformedBody)
				return
			}
			roleID = &parsed
		}
		in = UpdateStaff{
			FirstName: req.FirstName,
			LastName:  req.LastName,
			Email:     req.Email,
			RoleID:    roleID,
			IsActive:  req.IsActive,
		}
	}

	updated, err := h.svc.Update(r.Context(), id, in)
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, newStaffResponse(updated))
}

func (h *Handler) Deactivate(w http.ResponseWriter, r *http.Request) {
	id, err := staffIDParam(r)
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}
	deactivated, err := h.svc.Deactivate(r.Context(), id)
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, newStaffResponse(deactivated))
}

// ChangePassword always acts on the caller's own password, identified from the verified principal, never a path parameter.
func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	p, ok := principal.FromContext(r.Context())
	if !ok {
		h.writeErr(r.Context(), w, errs.ErrUnauthenticated)
		return
	}
	id, err := strconv.ParseInt(p.StaffID, 10, 64)
	if err != nil {
		h.writeErr(r.Context(), w, errs.ErrUnauthenticated)
		return
	}

	var req ChangePasswordRequest
	if err := httpx.DecodeJSON(r, &req, h.maxBodyBytes); err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}

	if err := h.svc.ChangePassword(r.Context(), id, req.CurrentPassword, req.NewPassword); err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func staffIDParam(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		return 0, httpx.ErrMalformedBody
	}
	return id, nil
}

// RequirePasswordChanged refuses every route except POST mePasswordPath and POST logoutPath while MustChangePassword is set; it reads the principal, never the database.
func RequirePasswordChanged(onErr func(context.Context, http.ResponseWriter, error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, ok := principal.FromContext(r.Context())
			if ok && p.MustChangePassword && !isAllowedDuringForcedChange(r) {
				onErr(r.Context(), w, errs.ErrPasswordChangeRequired)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func isAllowedDuringForcedChange(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	return r.URL.Path == mePasswordPath || r.URL.Path == logoutPath
}
