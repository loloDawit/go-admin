package ratelimit_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/loloDawit/go-admin/services/gateway/internal/auth"
	"github.com/loloDawit/go-admin/services/gateway/internal/ratelimit"
)

func requestAs(staffID string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)
	return req.WithContext(auth.WithStaffID(context.Background(), staffID))
}

// Asserting the 429 alone would pass against a limiter that counts after
// proxying. What matters is that a refused request never reaches the upstream.
func TestABurstNeverReachesTheUpstream(t *testing.T) {
	var reached int
	upstream := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		reached++
		w.WriteHeader(http.StatusOK)
	})
	handler := ratelimit.Middleware(ratelimit.Config{Rate: 1, Burst: 2})(upstream)

	var refused int
	for range 10 {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, requestAs("7"))
		if rec.Code == http.StatusTooManyRequests {
			refused++
			if rec.Header().Get("Retry-After") == "" {
				t.Fatal("a 429 with no Retry-After")
			}
		}
	}

	if refused == 0 {
		t.Fatal("ten requests at burst 2 were all allowed")
	}
	if reached+refused != 10 {
		t.Fatalf("reached=%d refused=%d, want them to sum to 10", reached, refused)
	}
	if reached > 3 {
		t.Fatalf("reached=%d, want at most burst plus one refill", reached)
	}
}

// Two people sharing one NAT address must not throttle each other, which is the
// whole reason this keys on the principal.
func TestTwoPrincipalsHaveSeparateBuckets(t *testing.T) {
	upstream := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	handler := ratelimit.Middleware(ratelimit.Config{Rate: 1, Burst: 1})(upstream)

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, requestAs("7"))
	exhausted := httptest.NewRecorder()
	handler.ServeHTTP(exhausted, requestAs("7"))
	if exhausted.Code != http.StatusTooManyRequests {
		t.Fatalf("the first principal's second request got %d, want 429", exhausted.Code)
	}

	second := httptest.NewRecorder()
	handler.ServeHTTP(second, requestAs("8"))
	if second.Code != http.StatusOK {
		t.Fatalf("the second principal got %d; the buckets are shared", second.Code)
	}
}

// An unauthenticated request has no principal to key on; LoginMiddleware, not
// this one, is what guards those.
func TestAnUnauthenticatedRequestPassesThrough(t *testing.T) {
	var reached int
	upstream := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		reached++
		w.WriteHeader(http.StatusOK)
	})
	handler := ratelimit.Middleware(ratelimit.Config{Rate: 1, Burst: 1})(upstream)

	for range 5 {
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/login", nil))
	}
	if reached != 5 {
		t.Fatalf("reached=%d, want all five", reached)
	}
}

func TestLoginIsLimitedPerIP(t *testing.T) {
	var reached int
	upstream := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		reached++
		w.WriteHeader(http.StatusOK)
	})
	handler := ratelimit.LoginMiddleware(ratelimit.Config{LoginRate: 0.2, LoginBurst: 2})(upstream)

	var refused int
	for range 6 {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/login", nil)
		req.RemoteAddr = "203.0.113.7:54321"
		handler.ServeHTTP(rec, req)
		if rec.Code == http.StatusTooManyRequests {
			refused++
		}
	}
	if refused == 0 {
		t.Fatal("six login attempts at burst 2 were all allowed")
	}
	if reached > 3 {
		t.Fatalf("reached=%d, want at most burst plus one refill", reached)
	}
}
