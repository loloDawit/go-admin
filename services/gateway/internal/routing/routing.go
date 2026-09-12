package routing

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/loloDawit/go-admin/platform/requestid"
	"github.com/loloDawit/go-admin/services/gateway/internal/httperr"
)

// New builds the fixed route-to-upstream mapping. Upstreams are statically
// configured; there is deliberately no service registry.
func New(upstreams map[string]string, timeout time.Duration) (http.Handler, error) {
	proxies := make(map[string]*httputil.ReverseProxy, len(upstreams))

	for name, raw := range upstreams {
		target, err := url.Parse(raw)
		if err != nil || target.Scheme == "" || target.Host == "" {
			return nil, fmt.Errorf("upstream %q is not a valid URL", name)
		}

		proxy := httputil.NewSingleHostReverseProxy(target)
		proxy.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, _ error) {
			// The transport error names the upstream host; the client gets none of it.
			httperr.WriteUpstreamUnavailable(w)
		}

		// ReverseProxy.ServeHTTP runs Director on a clone of the inbound request
		// (req.Clone), so rewriting the path here — rather than on the request
		// the route handler holds — cannot leak "/_platform" back into the
		// gateway's own request-logger route label.
		director := proxy.Director
		proxy.Director = func(out *http.Request) {
			director(out)
			out.URL.Path, out.URL.RawPath = "/_platform", ""
			// requestid.Middleware stamps the response header and the request
			// context, not the inbound request's own header; without this the
			// upstream would mint its own ID and the two logs wouldn't correlate.
			if id := requestid.FromContext(out.Context()); id != "" {
				out.Header.Set(requestid.Header, id)
			}
		}

		// The gateway's own requestid.Middleware already set this header on the
		// response before the proxy ran; without deleting the upstream's echo of
		// it here, copyHeader adds a second identical one instead of replacing it.
		proxy.ModifyResponse = func(resp *http.Response) error {
			resp.Header.Del(requestid.Header)
			return nil
		}
		proxies[name] = proxy
	}

	r := chi.NewRouter()

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	r.Get("/readyz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	r.Get("/_platform/{service}", func(w http.ResponseWriter, req *http.Request) {
		name := chi.URLParam(req, "service")
		proxy, ok := proxies[name]
		if !ok {
			httperr.WriteUnknownRoute(w)
			return
		}

		ctx, cancel := context.WithTimeout(req.Context(), timeout)
		defer cancel()

		proxy.ServeHTTP(w, req.WithContext(ctx))
	})

	r.NotFound(func(w http.ResponseWriter, _ *http.Request) { httperr.WriteUnknownRoute(w) })

	return r, nil
}
