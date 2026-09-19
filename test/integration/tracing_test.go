//go:build integration

package integration_test

import (
	"encoding/json"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

func jaegerURL() string {
	if url := os.Getenv("JAEGER_URL"); url != "" {
		return url
	}
	return "http://localhost:16686"
}

type jaegerTrace struct {
	TraceID string `json:"traceID"`
	Spans   []struct {
		OperationName string `json:"operationName"`
		ProcessID     string `json:"processID"`
		StartTime     int64  `json:"startTime"`
	} `json:"spans"`
	Processes map[string]struct {
		ServiceName string `json:"serviceName"`
	} `json:"processes"`
}

// since filters on the query rather than the default lookback window: a stale
// trace from a previous run would let this pass against a broken propagator,
// which is exactly what it did before the parameter was added.
func jaegerTraces(t *testing.T, service string, since time.Time) []jaegerTrace {
	t.Helper()
	start := strconv.FormatInt(since.UnixMicro(), 10)
	resp, err := http.Get(jaegerURL() + "/api/traces?service=" + service + "&limit=50&start=" + start)
	if err != nil {
		t.Fatalf("query jaeger: %v", err)
	}
	defer resp.Body.Close()

	var out struct {
		Data []jaegerTrace `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode jaeger response: %v", err)
	}
	return out.Data
}

func (tr jaegerTrace) services() map[string]bool {
	out := map[string]bool{}
	for _, span := range tr.Spans {
		out[tr.Processes[span.ProcessID].ServiceName] = true
	}
	return out
}

// One request must produce one trace containing every service that handled it.
// Each service starting its own trace is what a broken propagator looks like,
// and it is indistinguishable from working unless the services are counted
// within a single trace.
func TestOneRequestIsOneTraceAcrossServices(t *testing.T) {
	since := time.Now()
	c := loggedInClient(t)
	if status, env := apiCall(t, c, http.MethodGet, "/api/v1/orders", nil, nil); status != http.StatusOK {
		t.Fatalf("list orders: want 200, got %d (%s)", status, env.Code)
	}

	deadline := time.Now().Add(30 * time.Second)
	for {
		for _, tr := range jaegerTraces(t, "gateway", since) {
			if svcs := tr.services(); svcs["gateway"] && svcs["orders"] {
				return
			}
		}
		if time.Now().After(deadline) {
			t.Fatal("no single trace contains both gateway and orders spans")
		}
		time.Sleep(500 * time.Millisecond)
	}
}

// A path segment that is all digits, or a UUID, is an id rather than a pattern.
// "v1" is neither, which is why this looks at whole segments.
func concreteSegment(operationName string) string {
	for _, segment := range strings.Split(operationName, "/") {
		if idPattern.MatchString(segment) {
			return segment
		}
	}
	return ""
}

var idPattern = regexp.MustCompile(`^([0-9]+|[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})$`)

// The span name is the route pattern. A concrete id here is one operation name
// per order in the trace index, which is the same cardinality bomb the route
// label avoids.
func TestSpanNamesAreRoutePatterns(t *testing.T) {
	since := time.Now()
	c := loggedInClient(t)
	if status, env := apiCall(t, c, http.MethodGet, "/api/v1/orders", nil, nil); status != http.StatusOK {
		t.Fatalf("list orders: want 200, got %d (%s)", status, env.Code)
	}

	deadline := time.Now().Add(30 * time.Second)
	for {
		traces := jaegerTraces(t, "orders", since)
		for _, tr := range traces {
			for _, span := range tr.Spans {
				if !strings.Contains(span.OperationName, "/api/") {
					continue
				}
				if segment := concreteSegment(span.OperationName); segment != "" {
					t.Fatalf("operation name %q carries a concrete id: %q", span.OperationName, segment)
				}
			}
		}
		if len(traces) > 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("no orders traces reached jaeger")
		}
		time.Sleep(500 * time.Millisecond)
	}
}
