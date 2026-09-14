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

// rewrite runs after the default director points the request at target;
// nil preserves the inbound path.
func newProxy(logger *slog.Logger, name string, target *url.URL, rewrite func(*http.Request)) *httputil.ReverseProxy {
	proxy := httputil.NewSingleHostReverseProxy(target)

	baseDirector := proxy.Director
	proxy.Director = func(out *http.Request) {
		baseDirector(out)
		if rewrite != nil {
			rewrite(out)
		}
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

// "/_platform/{service}" and "/api/v1/*" use separate directors so a change
// to one cannot alter the other. "/internal/*" is never routed here.
func New(logger *slog.Logger, upstreams map[string]string, timeout time.Duration) (http.Handler, error) {
	targets := make(map[string]*url.URL, len(upstreams))
	for name, raw := range upstreams {
		target, err := url.Parse(raw)
		if err != nil || target.Scheme == "" || target.Host == "" {
			return nil, errs.Wrap(errs.OpParseUpstreamURL, fmt.Errorf("%s: %w", name, errs.ErrInvalidUpstreamURL))
		}
		targets[name] = target
	}

	// Identity has real routes now, so its walking skeleton was retired; catalog
	// and orders keep theirs until M3 and M4 replace them.
	platformProxies := make(map[string]*httputil.ReverseProxy, len(targets))
	for name, target := range targets {
		if name == "identity" {
			continue
		}
		platformProxies[name] = newProxy(logger, name, target, func(out *http.Request) {
			out.URL.Path, out.URL.RawPath = "/_platform", ""
		})
	}

	r := chi.NewRouter()

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	r.Get("/readyz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	r.Get("/_platform/{service}", func(w http.ResponseWriter, req *http.Request) {
		name := chi.URLParam(req, "service")
		proxy, ok := platformProxies[name]
		if !ok {
			httperr.WriteUnknownRoute(w)
			return
		}
		withTimeout(proxy, timeout)(w, req)
	})

	if identityTarget, ok := targets["identity"]; ok {
		identityAPI := newProxy(logger, "identity", identityTarget, nil)
		r.Handle("/api/v1/*", withTimeout(identityAPI, timeout))
	}

	r.NotFound(func(w http.ResponseWriter, _ *http.Request) { httperr.WriteUnknownRoute(w) })

	return r, nil
}
