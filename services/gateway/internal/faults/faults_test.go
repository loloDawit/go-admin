package faults_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/loloDawit/go-admin/services/gateway/internal/faults"
)

// Configuration fails fast: an injector that can be switched on outside the dev
// profile is a vulnerability, not a test tool.
func TestFaultsOutsideDevAreFatal(t *testing.T) {
	if err := faults.Validate(faults.Config{LatencyMS: 100}, "production"); err == nil {
		t.Fatal("injected latency was accepted in production")
	}
	if err := faults.Validate(faults.Config{ErrorRate: 0.1}, "production"); err == nil {
		t.Fatal("an error rate was accepted in production")
	}
	if err := faults.Validate(faults.Config{}, "production"); err != nil {
		t.Fatalf("an unconfigured injector was rejected in production: %v", err)
	}
	if err := faults.Validate(faults.Config{LatencyMS: 100}, "dev"); err != nil {
		t.Fatalf("injected latency was rejected in dev: %v", err)
	}
}

func TestAnImpossibleErrorRateIsRejected(t *testing.T) {
	if err := faults.Validate(faults.Config{ErrorRate: 1.5}, "dev"); err == nil {
		t.Fatal("an error rate above 1 was accepted")
	}
}

// At rate 1 the upstream must never be reached: an injector that answers 503
// after proxying has changed the response but not the load.
func TestAFullErrorRateNeverReachesTheUpstream(t *testing.T) {
	var reached int
	upstream := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		reached++
		w.WriteHeader(http.StatusOK)
	})
	handler := faults.Middleware(faults.Config{ErrorRate: 1})(upstream)

	for range 20 {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil))
		if rec.Code != http.StatusBadGateway {
			t.Fatalf("status = %d, want 502", rec.Code)
		}
	}
	if reached != 0 {
		t.Fatalf("the upstream was reached %d times", reached)
	}
}

func TestLatencyIsInjected(t *testing.T) {
	upstream := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	handler := faults.Middleware(faults.Config{LatencyMS: 120})(upstream)

	start := time.Now()
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	if elapsed := time.Since(start); elapsed < 100*time.Millisecond {
		t.Fatalf("elapsed = %s, want at least the injected latency", elapsed)
	}
}

// An unconfigured injector must be transparent, not merely harmless.
func TestAnUnconfiguredInjectorIsTransparent(t *testing.T) {
	var reached int
	upstream := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		reached++
		w.WriteHeader(http.StatusOK)
	})
	handler := faults.Middleware(faults.Config{})(upstream)

	start := time.Now()
	for range 5 {
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	}
	if reached != 5 {
		t.Fatalf("reached = %d, want 5", reached)
	}
	if elapsed := time.Since(start); elapsed > 50*time.Millisecond {
		t.Fatalf("elapsed = %s; an injector that is off must cost nothing", elapsed)
	}
}
