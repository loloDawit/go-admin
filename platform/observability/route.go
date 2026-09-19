package observability

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/loloDawit/go-admin/platform/requestid"
)

// RouteTagger names the span and labels the metrics from chi's route pattern.
// It runs after the handler because the pattern is only known once routing has
// matched, and the span is still live at that point.
func RouteTagger() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)

			ctx := r.Context()
			pattern := routePattern(r)
			if pattern == "" {
				return
			}
			route := attribute.String("http.route", pattern)

			if span := trace.SpanFromContext(ctx); span.IsRecording() {
				span.SetName(r.Method + " " + pattern)
				span.SetAttributes(route)
				if id := requestid.FromContext(ctx); id != "" {
					span.SetAttributes(attribute.String("request_id", id))
				}
			}
			// The labeler is how otelhttp's own metrics learn the route: it
			// records duration after the handler returns.
			labeler, _ := otelhttp.LabelerFromContext(ctx)
			labeler.Add(route)
		})
	}
}

func routePattern(r *http.Request) string {
	rctx := chi.RouteContext(r.Context())
	if rctx == nil {
		return ""
	}
	return rctx.RoutePattern()
}

// HTTPMiddleware instruments inbound requests. Health and readiness probes are
// excluded: compose polls them every few seconds per service, and at
// AlwaysSample they would bury the traces an operator is looking for.
func HTTPMiddleware(service string) func(http.Handler) http.Handler {
	return otelhttp.NewMiddleware(service, otelhttp.WithFilter(func(r *http.Request) bool {
		return r.URL.Path != "/healthz" && r.URL.Path != "/readyz"
	}))
}
