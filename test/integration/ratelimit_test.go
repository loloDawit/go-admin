//go:build integration

package integration_test

import (
	"net/http"
	"testing"
	"time"
)

// A burst from one member of staff is refused at the gateway. Asserting the 429
// alone would pass against a limiter that counts after proxying, so this also
// checks the refusal carries Retry-After and that the limit recovers, which a
// limiter that had simply broken would not do.
func TestABurstIsRefusedAndThenRecovers(t *testing.T) {
	c := loggedInClient(t)

	var refused, served int
	var retryAfter string
	for range 400 {
		resp, err := c.Get(gatewayURL() + "/api/v1/orders")
		if err != nil {
			t.Fatalf("request: %v", err)
		}
		resp.Body.Close()
		switch resp.StatusCode {
		case http.StatusTooManyRequests:
			refused++
			if retryAfter == "" {
				retryAfter = resp.Header.Get("Retry-After")
			}
		case http.StatusOK:
			served++
		default:
			t.Fatalf("unexpected status %d", resp.StatusCode)
		}
	}

	if refused == 0 {
		t.Fatalf("400 requests were all served; the limiter is not refusing anything")
	}
	if served == 0 {
		t.Fatal("no request was served at all; the limiter is refusing everything")
	}
	if retryAfter == "" {
		t.Fatal("a 429 carried no Retry-After header")
	}

	// The bucket refills, so a rate limit is a delay and not a lockout.
	time.Sleep(2 * time.Second)
	resp, err := c.Get(gatewayURL() + "/api/v1/orders")
	if err != nil {
		t.Fatalf("request after the bucket refilled: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("after refill: want 200, got %d", resp.StatusCode)
	}
}
