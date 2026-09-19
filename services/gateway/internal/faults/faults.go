// Package faults injects latency and errors at the gateway so a load test can
// observe degradation rather than a hard outage.
package faults

import (
	"math/rand/v2"
	"net/http"
	"time"

	"github.com/loloDawit/go-admin/services/gateway/internal/errs"
	"github.com/loloDawit/go-admin/services/gateway/internal/httperr"
)

// DevProfile is the only profile in which injection is permitted.
const DevProfile = "dev"

type Config struct {
	LatencyMS int
	ErrorRate float64
}

func (c Config) enabled() bool {
	return c.LatencyMS > 0 || c.ErrorRate > 0
}

// An injector reachable in production by an environment variable is a
// vulnerability, not a test tool, so this is fatal at startup rather than
// ignored at request time.
func Validate(cfg Config, profile string) error {
	if cfg.enabled() && profile != DevProfile {
		return errs.ErrFaultsNotPermitted
	}
	if cfg.ErrorRate < 0 || cfg.ErrorRate > 1 {
		return errs.ErrInvalidErrorRate
	}
	return nil
}

// Middleware returns next unchanged when nothing is configured, so an injector
// that is off costs nothing on the request path.
func Middleware(cfg Config) func(http.Handler) http.Handler {
	if !cfg.enabled() {
		return func(next http.Handler) http.Handler { return next }
	}
	latency := time.Duration(cfg.LatencyMS) * time.Millisecond
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if latency > 0 {
				select {
				case <-time.After(latency):
				case <-r.Context().Done():
					return
				}
			}
			if cfg.ErrorRate > 0 && rand.Float64() < cfg.ErrorRate {
				httperr.WriteUpstreamUnavailable(w)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
