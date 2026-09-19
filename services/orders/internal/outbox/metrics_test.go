package outbox_test

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/loloDawit/go-admin/services/orders/internal/outbox"
)

type stubDepth struct{ depth int64 }

func (s *stubDepth) UnpublishedDepth(context.Context) (int64, error) { return s.depth, nil }

func gaugeValue(t *testing.T, rm *metricdata.ResourceMetrics, name string) int64 {
	t.Helper()
	for _, scope := range rm.ScopeMetrics {
		for _, m := range scope.Metrics {
			if m.Name != name {
				continue
			}
			gauge, ok := m.Data.(metricdata.Gauge[int64])
			if !ok || len(gauge.DataPoints) == 0 {
				t.Fatalf("%s is not an int64 gauge with data", name)
			}
			return gauge.DataPoints[0].Value
		}
	}
	t.Fatalf("metric %s was not exported", name)
	return 0
}

// The gauge is observed from the database, so a worker that restarts with 40
// unpublished rows reports 40 rather than 0.
func TestDepthIsReadFromTheStore(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	otel.SetMeterProvider(sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)))

	m, err := outbox.NewMetrics()
	if err != nil {
		t.Fatalf("NewMetrics: %v", err)
	}
	if err := m.WatchDepth(&stubDepth{depth: 40}); err != nil {
		t.Fatalf("WatchDepth: %v", err)
	}

	var collected metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &collected); err != nil {
		t.Fatalf("Collect: %v", err)
	}

	if got := gaugeValue(t, &collected, "orders_outbox_unpublished"); got != 40 {
		t.Fatalf("orders_outbox_unpublished = %d, want 40", got)
	}
}
