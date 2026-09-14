package session

import (
	"context"
	"net/http"
	"time"

	"github.com/loloDawit/go-admin/platform/httpx"
	"github.com/loloDawit/go-admin/platform/principal"
)

// cookieName is shared by Login (sets it) and Logout (reads and clears it).
const cookieName = "session"

// writeErr is injected, not imported: internal/httperr imports this package for its sentinels, so importing it back would cycle.
type Handler struct {
	svc          *Service
	cookieSecure bool
	sessionTTL   time.Duration
	maxBodyBytes int64
	writeErr     func(ctx context.Context, w http.ResponseWriter, err error)
}

func NewHandler(svc *Service, cookieSecure bool, sessionTTL time.Duration, maxBodyBytes int64, writeErr func(context.Context, http.ResponseWriter, error)) *Handler {
	return &Handler{svc: svc, cookieSecure: cookieSecure, sessionTTL: sessionTTL, maxBodyBytes: maxBodyBytes, writeErr: writeErr}
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := httpx.DecodeJSON(r, &req, h.maxBodyBytes); err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}

	token, auth, err := h.svc.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(h.sessionTTL),
	})
	httpx.WriteJSON(w, http.StatusOK, newAuthResponse(auth))
}

// Logout reads the token from the cookie, not the signed principal: the principal never carries the plaintext token.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		h.writeErr(r.Context(), w, ErrUnauthenticated)
		return
	}

	if err := h.svc.Revoke(r.Context(), cookie.Value); err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
	w.WriteHeader(http.StatusNoContent)
}

// Me reads the staff row by the principal's StaffID: the signed principal carries no email.
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	p, ok := principal.FromContext(r.Context())
	if !ok {
		h.writeErr(r.Context(), w, ErrUnauthenticated)
		return
	}
	auth, err := h.svc.Current(r.Context(), p.StaffID)
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, newAuthResponse(auth))
}

// Validate serves POST /internal/sessions/validate. The token travels in the
// JSON body, never a query string: a query string reaches access logs.
func (h *Handler) Validate(w http.ResponseWriter, r *http.Request) {
	var req ValidateRequest
	if err := httpx.DecodeJSON(r, &req, h.maxBodyBytes); err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}

	p, err := h.svc.Validate(r.Context(), req.Token)
	if err != nil {
		h.writeErr(r.Context(), w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, ValidateResponse{
		StaffID:            p.StaffID,
		Permissions:        p.Permissions,
		MustChangePassword: p.MustChangePassword,
	})
}
