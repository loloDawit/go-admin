// Package auth resolves a session cookie into a signed principal, once per
// request, and forwards it downstream.
package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/loloDawit/go-admin/platform/principal"
	"github.com/loloDawit/go-admin/services/gateway/internal/errs"
	"github.com/loloDawit/go-admin/services/gateway/internal/httperr"
)

// cookieName matches identity's own session cookie name.
const cookieName = "session"

// Validator resolves a plaintext session token to a signed principal,
// consulting the cache before ever calling identity.
type Validator struct {
	httpClient   *http.Client
	validateURL  string
	key          []byte
	principalTTL time.Duration
	cache        *Cache
	errWriter    *httperr.Writer
	logger       *slog.Logger
}

func NewValidator(httpClient *http.Client, validateURL string, key []byte, principalTTL time.Duration, cache *Cache, errWriter *httperr.Writer, logger *slog.Logger) *Validator {
	return &Validator{
		httpClient:   httpClient,
		validateURL:  validateURL,
		key:          key,
		principalTTL: principalTTL,
		cache:        cache,
		errWriter:    errWriter,
		logger:       logger,
	}
}

// Mirrors identity's session.ValidateRequest, duplicated so the gateway need
// not import identity's internal package.
type validateRequest struct {
	Token string `json:"token"`
}

// resolved is what identity's validate endpoint answers with — the part of a
// session's outcome that doesn't change between calls within its TTL.
type resolved struct {
	StaffID            string   `json:"staffId"`
	Permissions        []string `json:"permissions"`
	MustChangePassword bool     `json:"mustChangePassword"`
}

// IssuedAt/ExpiresAt are set from this call's clock, every call, cache hit or
// miss: only the identity fields are cached.
func (v *Validator) Resolve(ctx context.Context, token string) (principal.Principal, error) {
	r, ok := v.cache.Get(token)
	if !ok {
		var err error
		r, err = v.validate(ctx, token)
		if err != nil {
			return principal.Principal{}, err
		}
		v.cache.Set(token, r)
	}

	now := time.Now()
	return principal.Principal{
		StaffID:            r.StaffID,
		Permissions:        r.Permissions,
		MustChangePassword: r.MustChangePassword,
		IssuedAt:           now,
		ExpiresAt:          now.Add(v.principalTTL),
	}, nil
}

// A 401 is ErrSessionInvalid and unlogged: a rejected token is expected.
// Any other non-200 is logged: identity failed, and that must not look like
// an ordinary logout to whoever reads the gateway's logs.
func (v *Validator) validate(ctx context.Context, token string) (resolved, error) {
	body, err := json.Marshal(validateRequest{Token: token})
	if err != nil {
		return resolved{}, errs.Wrap(errs.OpBuildValidateRequest, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, v.validateURL, bytes.NewReader(body))
	if err != nil {
		return resolved{}, errs.Wrap(errs.OpBuildValidateRequest, err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := v.httpClient.Do(req)
	if err != nil {
		v.logger.ErrorContext(ctx, "identity unreachable", slog.String("error", err.Error()))
		return resolved{}, errs.Wrap(errs.OpCallIdentityValidate, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return resolved{}, errs.ErrSessionInvalid
	}
	if resp.StatusCode != http.StatusOK {
		v.logger.ErrorContext(ctx, "identity validate returned an unexpected status",
			slog.Int("status", resp.StatusCode))
		return resolved{}, errs.Wrap(errs.OpCallIdentityValidate, fmt.Errorf("unexpected status %d", resp.StatusCode))
	}

	var out resolved
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		v.logger.ErrorContext(ctx, "malformed identity validate response", slog.String("error", err.Error()))
		return resolved{}, errs.Wrap(errs.OpDecodeValidateResponse, err)
	}
	return out, nil
}

// Middleware never decides whether an unauthenticated request may proceed;
// a downstream service does, by whether it finds X-Principal.
func Middleware(v *Validator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Must run before the cookie check: a forged header with no
			// cookie must still be stripped.
			r.Header.Del(principal.HeaderPrincipal)
			r.Header.Del(principal.HeaderSignature)

			c, err := r.Cookie(cookieName)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			p, err := v.Resolve(r.Context(), c.Value)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			header, sig, err := principal.Sign(p, v.key)
			if err != nil {
				v.errWriter.Write(r.Context(), w, errs.Wrap(errs.OpSignPrincipal, err))
				return
			}
			r.Header.Set(principal.HeaderPrincipal, header)
			r.Header.Set(principal.HeaderSignature, sig)
			next.ServeHTTP(w, r)
		})
	}
}
