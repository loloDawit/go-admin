package observability

import (
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/loloDawit/go-admin/platform/requestid"
)

func NewLogger(service string, w io.Writer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{Level: slog.LevelInfo})).
		With(slog.String("service", service))
}

func RequestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(rec, r)

			logger.InfoContext(r.Context(), "request",
				slog.String("request_id", requestid.FromContext(r.Context())),
				slog.String("method", r.Method),
				slog.String("route", r.URL.Path),
				slog.Int("status", rec.status),
				slog.Int64("duration_ms", time.Since(start).Milliseconds()),
			)
		})
	}
}

// statusRecorder captures the status for the log line. WriteHeader may not be
// called at all, so the zero value must be 200.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}
