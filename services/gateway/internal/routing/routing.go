package routing

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/loloDawit/go-admin/platform/requestid"
	"github.com/loloDawit/go-admin/services/gateway/internal/errs"
	"github.com/loloDawit/go-admin/services/gateway/internal/httperr"
)

func newProxy(logger *slog.Logger, name string, target *url.URL) *httputil.ReverseProxy {
	proxy := httputil.NewSingleHostReverseProxy(target)

	baseDirector := proxy.Director
	proxy.Director = func(out *http.Request) {
		baseDirector(out)
		// The upstream would otherwise mint its own request ID.
		if id := requestid.FromContext(out.Context()); id != "" {
			out.Header.Set(requestid.Header, id)
		}
	}

	// ReverseProxy's default ErrorHandler is the only thing that would log a
	// transport failure; this one replaces it and must log itself.
	proxy.ErrorHandler = func(w http.ResponseWriter, req *http.Request, err error) {
		logger.ErrorContext(req.Context(), "upstream unavailable",
			slog.String("upstream", name),
			slog.String("request_id", requestid.FromContext(req.Context())),
			slog.String("error", err.Error()),
		)

		if errors.Is(err, context.DeadlineExceeded) {
			httperr.WriteGatewayTimeout(w)
			return
		}
		httperr.WriteUpstreamUnavailable(w)
	}

	// Without this, copyHeader adds a second X-Request-Id alongside the
	// upstream's echo of the one this gateway already set.
	proxy.ModifyResponse = func(resp *http.Response) error {
		resp.Header.Del(requestid.Header)
		return nil
	}

	return proxy
}

func withTimeout(proxy *httputil.ReverseProxy, timeout time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx, cancel := context.WithTimeout(req.Context(), timeout)
		defer cancel()
		proxy.ServeHTTP(w, req.WithContext(ctx))
	}
}

// apiOwners routes an /api/v1 prefix to the service that owns it. Anything not
// listed falls through to identity, which owns the rest of the surface.
var apiOwners = map[string]string{
	"/api/v1/products":    "catalog",
	"/api/v1/products/*":  "catalog",
	"/api/v1/orders":      "orders",
	"/api/v1/orders/*":    "orders",
	"/api/v1/customers":   "orders",
	"/api/v1/customers/*": "orders",
}

// "/api/v1/*" is the only surface proxied here. "/internal/*" is never routed here.
func New(logger *slog.Logger, upstreams map[string]string, timeout time.Duration) (http.Handler, error) {
	targets := make(map[string]*url.URL, len(upstreams))
	for name, raw := range upstreams {
		target, err := url.Parse(raw)
		if err != nil || target.Scheme == "" || target.Host == "" {
			return nil, errs.Wrap(errs.OpParseUpstreamURL, fmt.Errorf("%s: %w", name, errs.ErrInvalidUpstreamURL))
		}
		targets[name] = target
	}

	r := chi.NewRouter()

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	r.Get("/readyz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	// Longest-prefix first: chi matches a literal segment ahead of a wildcard,
	// so identity's catch-all cannot swallow a path another service owns.
	for prefix, name := range apiOwners {
		target, ok := targets[name]
		if !ok {
			continue
		}
		r.Handle(prefix, withTimeout(newProxy(logger, name, target), timeout))
	}

	if identityTarget, ok := targets["identity"]; ok {
		identityAPI := newProxy(logger, "identity", identityTarget)
		r.Handle("/api/v1/*", withTimeout(identityAPI, timeout))
	}

	r.NotFound(func(w http.ResponseWriter, _ *http.Request) { httperr.WriteUnknownRoute(w) })

	return r, nil
}
