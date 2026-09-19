package outbox

import (
	"context"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/loloDawit/go-admin/services/orders/internal/errs"
	"github.com/loloDawit/go-admin/services/orders/internal/natsx"
)

type PublisherStore interface {
	Unpublished(ctx context.Context, limit int) ([]Stored, error)
	MarkPublished(ctx context.Context, id int64) error
}

type Publisher struct {
	store    PublisherStore
	js       jetstream.JetStream
	batch    int
	interval time.Duration
	logger   *slog.Logger
	metrics  *Metrics
}

func NewPublisher(store PublisherStore, js jetstream.JetStream, batch int, interval time.Duration, logger *slog.Logger, metrics *Metrics) *Publisher {
	return &Publisher{store: store, js: js, batch: batch, interval: interval, logger: logger, metrics: metrics}
}

var tracer = otel.Tracer("orders/outbox")

func (p *Publisher) Run(ctx context.Context) error {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := p.drain(ctx); err != nil && ctx.Err() == nil {
				p.logger.ErrorContext(ctx, "outbox drain", slog.String("error", err.Error()))
			}
		}
	}
}

// A failed publish stops the batch rather than skipping the row: the next row's
// event must not overtake it, because the projection applies paid and refunded
// in the order it receives them.
func (p *Publisher) drain(ctx context.Context) error {
	rows, err := p.store.Unpublished(ctx, p.batch)
	if err != nil {
		return err
	}

	for _, row := range rows {
		if err := p.publish(ctx, row); err != nil {
			return err
		}
	}
	return nil
}

func (p *Publisher) publish(ctx context.Context, row Stored) error {
	ctx, span := tracer.Start(ctx, "nats.publish", trace.WithSpanKind(trace.SpanKindProducer))
	defer span.End()
	span.SetAttributes(attribute.String("messaging.destination.name", row.Subject))

	msg := &nats.Msg{
		Subject: row.Subject,
		Data:    row.Payload,
		Header:  nats.Header{natsx.MsgIDHeader: []string{row.EventID}},
	}
	if _, err := p.js.PublishMsg(ctx, msg); err != nil {
		p.metrics.CountError(ctx, "publish")
		span.RecordError(err)
		return errs.Wrap(errs.OpPublishEvent, err)
	}
	if err := p.store.MarkPublished(ctx, row.ID); err != nil {
		p.metrics.CountError(ctx, "mark_published")
		span.RecordError(err)
		return err
	}
	p.metrics.ObserveLag(ctx, time.Since(row.CreatedAt))
	return nil
}
