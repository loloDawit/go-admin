//go:build integration

package integration_test

import (
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"
)

func prometheusURL() string {
	if url := os.Getenv("PROMETHEUS_URL"); url != "" {
		return url
	}
	return "http://localhost:9090"
}

func prometheusGet(t *testing.T, path string, out any) {
	t.Helper()
	resp, err := http.Get(prometheusURL() + path)
	if err != nil {
		t.Fatalf("query prometheus: %v", err)
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		t.Fatalf("decode prometheus response: %v", err)
	}
}

// One time series per order id would take the Prometheus instance down at a few
// thousand orders, which is why the label is the route pattern.
func TestRouteLabelIsAPattern(t *testing.T) {
	c := loggedInClient(t)
	if status, env := apiCall(t, c, http.MethodGet, "/api/v1/orders", nil, nil); status != http.StatusOK {
		t.Fatalf("list orders: want 200, got %d (%s)", status, env.Code)
	}

	deadline := time.Now().Add(60 * time.Second)
	for {
		var out struct {
			Data []string `json:"data"`
		}
		prometheusGet(t, "/api/v1/label/http_route/values", &out)

		if len(out.Data) > 0 {
			for _, value := range out.Data {
				if segment := concreteSegment(value); segment != "" {
					t.Fatalf("http_route %q carries a concrete id: %q", value, segment)
				}
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("no http_route label values reached prometheus")
		}
		time.Sleep(time.Second)
	}
}

// A rule file that does not parse is silently absent rather than a boot
// failure, so the only way to know the alerts exist is to ask for them.
func TestAlertRulesAreLoaded(t *testing.T) {
	var out struct {
		Data struct {
			Groups []struct {
				Rules []struct {
					Name string `json:"name"`
				} `json:"rules"`
			} `json:"groups"`
		} `json:"data"`
	}
	prometheusGet(t, "/api/v1/rules", &out)

	loaded := map[string]bool{}
	for _, group := range out.Data.Groups {
		for _, rule := range group.Rules {
			loaded[rule.Name] = true
		}
	}
	for _, want := range []string{"OutboxBacklogGrowing", "HighErrorRate", "ProjectionStalled"} {
		if !loaded[want] {
			t.Fatalf("alert %s is not loaded; rules present: %v", want, loaded)
		}
	}
}

// The outbox metrics are what answer "was the operation committed?" and "was
// the event eventually published?", so their absence is a silent hole in the
// only signal there is.
//
// It places its own order rather than relying on another test having placed
// one: the publish-lag histogram does not exist until something is published,
// and on a cold boot this ran before anything had been.
func TestOutboxMetricsAreExported(t *testing.T) {
	placeOrder(t)

	want := []string{
		"orders_outbox_unpublished",
		"orders_outbox_publish_lag_seconds_bucket",
		"orders_projection_applied_total",
		"http_server_request_duration_seconds_bucket",
		"db_client_operation_duration_seconds_bucket",
	}

	// The export interval is 10s and the scrape 5s, so the first sample of a
	// metric can be twenty seconds behind the event that produced it.
	deadline := time.Now().Add(90 * time.Second)
	var missing []string
	for {
		var out struct {
			Data []string `json:"data"`
		}
		prometheusGet(t, "/api/v1/label/__name__/values", &out)

		exported := map[string]bool{}
		for _, name := range out.Data {
			exported[name] = true
		}
		missing = missing[:0]
		for _, name := range want {
			if !exported[name] {
				missing = append(missing, name)
			}
		}
		if len(missing) == 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("metrics never reached prometheus: %v", missing)
		}
		time.Sleep(2 * time.Second)
	}
}
