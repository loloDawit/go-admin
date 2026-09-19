// Package ratelimit refuses a burst at the gateway, before it reaches an
// upstream.
package ratelimit

import (
	"net"
	"net/http"
	"sync"

	"golang.org/x/time/rate"

	"github.com/loloDawit/go-admin/services/gateway/internal/auth"
	"github.com/loloDawit/go-admin/services/gateway/internal/httperr"
)

type Config struct {
	Rate       float64
	Burst      int
	LoginRate  float64
	LoginBurst int
}

type buckets struct {
	mu    sync.Mutex
	limit rate.Limit
	burst int
	byKey map[string]*rate.Limiter
}

func newBuckets(perSecond float64, burst int) *buckets {
	return &buckets{limit: rate.Limit(perSecond), burst: burst, byKey: map[string]*rate.Limiter{}}
}

func (b *buckets) allow(key string) bool {
	b.mu.Lock()
	limiter, ok := b.byKey[key]
	if !ok {
		limiter = rate.NewLimiter(b.limit, b.burst)
		b.byKey[key] = limiter
	}
	b.mu.Unlock()
	return limiter.Allow()
}

// Middleware keys on the principal, not the IP: every browser tab in one shop
// shares one NAT address, so an IP limit either throttles the whole staff at
// once or is set so high it limits nothing. It must run after the principal is
// resolved.
//
// A request with no principal passes through; LoginMiddleware guards the one
// unauthenticated path that is worth brute-forcing.
func Middleware(cfg Config) func(http.Handler) http.Handler {
	b := newBuckets(cfg.Rate, cfg.Burst)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			staffID := auth.StaffIDFromContext(r.Context())
			if staffID == "" || b.allow(staffID) {
				next.ServeHTTP(w, r)
				return
			}
			httperr.WriteRateLimited(w)
		})
	}
}

// LoginMiddleware is per-IP and deliberately so: login is the one
// unauthenticated path, and there is no principal to key on before it succeeds.
func LoginMiddleware(cfg Config) func(http.Handler) http.Handler {
	b := newBuckets(cfg.LoginRate, cfg.LoginBurst)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if b.allow(clientIP(r)) {
				next.ServeHTTP(w, r)
				return
			}
			httperr.WriteRateLimited(w)
		})
	}
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
