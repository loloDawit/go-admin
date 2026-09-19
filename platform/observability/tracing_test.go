package observability

import (
	"context"
	"slices"
	"testing"

	"go.opentelemetry.io/otel"
)

// An unconfigured endpoint must leave a working, non-recording tracer rather
// than an error: a service with no collector still has to serve requests.
func TestNoEndpointYieldsNoOpProvider(t *testing.T) {
	shutdown, err := NewTracerProvider(context.Background(), "test", "")
	if err != nil {
		t.Fatalf("NewTracerProvider: %v", err)
	}
	t.Cleanup(func() { _ = shutdown(context.Background()) })

	_, span := otel.Tracer("test").Start(context.Background(), "op")
	defer span.End()

	if span.IsRecording() {
		t.Fatal("a no-op provider produced a recording span")
	}
}

// The propagator is installed even without an endpoint: a service that does not
// export traces must still forward the context of one that does.
func TestPropagatorIsInstalled(t *testing.T) {
	if _, err := NewTracerProvider(context.Background(), "test", ""); err != nil {
		t.Fatalf("NewTracerProvider: %v", err)
	}
	if fields := otel.GetTextMapPropagator().Fields(); !slices.Contains(fields, "traceparent") {
		t.Fatalf("propagator fields = %v, want traceparent", fields)
	}
}
