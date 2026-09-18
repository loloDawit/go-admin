package observability

import (
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/loloDawit/go-admin/platform/requestid"
)

func NewLogger(service string, w io.Writer) *slog.Logger {
	base := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: slog.LevelInfo})
	return slog.New(traceHandler{Handler: base}).With(slog.String("service", service))
}

func RequestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(rec, r)

			// route is chi's pattern, not the concrete path: a path here is one
			// log stream, and one metric time series, per order.
			attrs := []slog.Attr{
				slog.String("request_id", requestid.FromContext(r.Context())),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", rec.status),
				slog.Int64("duration_ms", time.Since(start).Milliseconds()),
			}
			if pattern := routePattern(r); pattern != "" {
				attrs = append(attrs, slog.String("route", pattern))
			}
			logger.LogAttrs(r.Context(), slog.LevelInfo, "request", attrs...)
		})
	}
}

// statusRecorder's status field must default to 200: a handler that writes nothing at all never calls WriteHeader.
type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (r *statusRecorder) WriteHeader(status int) {
	if r.wroteHeader {
		return // net/http ignores a second call; the logged status must match.
	}
	r.wroteHeader = true
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	return r.ResponseWriter.Write(b)
}

// Unwrap lets http.NewResponseController (httputil.ReverseProxy's flushing and Hijack/websocket upgrades) reach the underlying writer.
func (r *statusRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }
