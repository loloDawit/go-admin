package outbox

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/loloDawit/go-admin/services/orders/internal/errs"
)

type DepthStore interface {
	UnpublishedDepth(ctx context.Context) (int64, error)
}

type Metrics struct {
	meter  metric.Meter
	depth  metric.Int64ObservableGauge
	lag    metric.Float64Histogram
	errors metric.Int64Counter
}

func NewMetrics() (*Metrics, error) {
	meter := otel.Meter("orders/outbox")

	depth, err := meter.Int64ObservableGauge("orders_outbox_unpublished",
		metric.WithDescription("outbox rows that have not been published"))
	if err != nil {
		return nil, errs.Wrap(errs.OpRegisterMetric, err)
	}
	lag, err := meter.Float64Histogram("orders_outbox_publish_lag_seconds",
		metric.WithDescription("seconds between an outbox row being written and published"))
	if err != nil {
		return nil, errs.Wrap(errs.OpRegisterMetric, err)
	}
	failures, err := meter.Int64Counter("orders_outbox_publish_errors_total",
		metric.WithDescription("failed publish attempts"))
	if err != nil {
		return nil, errs.Wrap(errs.OpRegisterMetric, err)
	}
	return &Metrics{meter: meter, depth: depth, lag: lag, errors: failures}, nil
}

// An observable gauge is the right instrument because it is pulled: the
// callback asks the database at collection time, so the value cannot go stale
// or reset when the worker restarts.
func (m *Metrics) WatchDepth(store DepthStore) error {
	_, err := m.meter.RegisterCallback(func(ctx context.Context, o metric.Observer) error {
		depth, err := store.UnpublishedDepth(ctx)
		if err != nil {
			return err
		}
		o.ObserveInt64(m.depth, depth)
		return nil
	}, m.depth)
	if err != nil {
		return errs.Wrap(errs.OpRegisterMetric, err)
	}
	return nil
}

func (m *Metrics) ObserveLag(ctx context.Context, d time.Duration) {
	m.lag.Record(ctx, d.Seconds())
}

func (m *Metrics) CountError(ctx context.Context, class string) {
	m.errors.Add(ctx, 1, metric.WithAttributes(attribute.String("class", class)))
}
