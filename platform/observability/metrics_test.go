package observability

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
)

// A no-op meter that errors, or panics on Add, would take down every call site
// that records a metric in a service with no collector.
func TestNoEndpointYieldsAUsableNoOpMeter(t *testing.T) {
	shutdown, err := NewMeterProvider(context.Background(), "test", "")
	if err != nil {
		t.Fatalf("NewMeterProvider: %v", err)
	}
	t.Cleanup(func() { _ = shutdown(context.Background()) })

	counter, err := otel.Meter("test").Int64Counter("things")
	if err != nil {
		t.Fatalf("Int64Counter: %v", err)
	}
	counter.Add(context.Background(), 1)
}
