package reporting

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/loloDawit/go-admin/services/orders/internal/outbox"
)

func counterValue(t *testing.T, rm *metricdata.ResourceMetrics, name, result string) int64 {
	t.Helper()
	for _, scope := range rm.ScopeMetrics {
		for _, m := range scope.Metrics {
			if m.Name != name {
				continue
			}
			sum, ok := m.Data.(metricdata.Sum[int64])
			if !ok {
				t.Fatalf("%s is not an int64 sum", name)
			}
			for _, dp := range sum.DataPoints {
				if v, found := dp.Attributes.Value("result"); found && v.AsString() == result {
					return dp.Value
				}
			}
		}
	}
	return 0
}

func applyOnce(t *testing.T, applied bool) metricdata.ResourceMetrics {
	t.Helper()
	reader := sdkmetric.NewManualReader()
	otel.SetMeterProvider(sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)))

	metrics, err := NewMetrics()
	if err != nil {
		t.Fatalf("NewMetrics: %v", err)
	}
	p := NewProjector(applierFunc(func(context.Context, outbox.Envelope) (bool, error) {
		return applied, nil
	}), nil, slog.New(slog.DiscardHandler), metrics)

	env := outbox.Envelope{ID: "e1", Type: outbox.TypeOrderCreated, Version: 1, Payload: json.RawMessage(`{}`)}
	if err := p.apply(context.Background(), env); err != nil {
		t.Fatalf("apply: %v", err)
	}

	var collected metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &collected); err != nil {
		t.Fatalf("Collect: %v", err)
	}
	return collected
}

// A redelivery must be counted as a duplicate, not silently as an application:
// the projection looks identical either way and only the counter separates them.
func TestDuplicateIsCountedSeparately(t *testing.T) {
	collected := applyOnce(t, false)

	if got := counterValue(t, &collected, "orders_projection_applied_total", ResultDuplicate); got != 1 {
		t.Fatalf("duplicate count = %d, want 1", got)
	}
	if got := counterValue(t, &collected, "orders_projection_applied_total", ResultApplied); got != 0 {
		t.Fatalf("applied count = %d, want 0", got)
	}
}

func TestAppliedIsCountedSeparately(t *testing.T) {
	collected := applyOnce(t, true)

	if got := counterValue(t, &collected, "orders_projection_applied_total", ResultApplied); got != 1 {
		t.Fatalf("applied count = %d, want 1", got)
	}
	if got := counterValue(t, &collected, "orders_projection_applied_total", ResultDuplicate); got != 0 {
		t.Fatalf("duplicate count = %d, want 0", got)
	}
}
