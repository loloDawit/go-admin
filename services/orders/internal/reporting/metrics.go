package reporting

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/loloDawit/go-admin/services/orders/internal/errs"
)

const (
	ResultApplied        = "applied"
	ResultDuplicate      = "duplicate"
	ResultUnknownVersion = "unknown_version"
)

type Metrics struct {
	applied metric.Int64Counter
}

func NewMetrics() (*Metrics, error) {
	applied, err := otel.Meter("orders/reporting").Int64Counter(
		"orders_projection_applied_total",
		metric.WithDescription("events the projection processed, by outcome"),
	)
	if err != nil {
		return nil, errs.Wrap(errs.OpRegisterMetric, err)
	}
	return &Metrics{applied: applied}, nil
}

// Counting duplicates is how the consumer-side dedup layer becomes visible
// rather than merely correct: the projection looks identical either way.
func (m *Metrics) Applied(ctx context.Context, eventType, result string) {
	m.applied.Add(ctx, 1, metric.WithAttributes(
		attribute.String("type", eventType),
		attribute.String("result", result),
	))
}
