package reporting

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/nats-io/nats.go/jetstream"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/loloDawit/go-admin/services/orders/internal/errs"
	"github.com/loloDawit/go-admin/services/orders/internal/outbox"
)

type Applier interface {
	Apply(ctx context.Context, env outbox.Envelope) error
}

type Projector struct {
	repo     Applier
	consumer jetstream.Consumer
	logger   *slog.Logger
}

func NewProjector(repo Applier, consumer jetstream.Consumer, logger *slog.Logger) *Projector {
	return &Projector{repo: repo, consumer: consumer, logger: logger}
}

func (p *Projector) Run(ctx context.Context) error {
	consumed, err := p.consumer.Consume(func(msg jetstream.Msg) {
		p.handle(ctx, msg)
	})
	if err != nil {
		return errs.Wrap(errs.OpProjectEvent, err)
	}
	defer consumed.Stop()

	<-ctx.Done()
	return nil
}

// A message that cannot be decoded, or names a version this build does not
// implement, is acknowledged: a redelivery cannot make either true. The log is
// what says the projection is knowingly behind.
func (p *Projector) handle(ctx context.Context, msg jetstream.Msg) {
	var env outbox.Envelope
	if err := json.Unmarshal(msg.Data(), &env); err != nil {
		p.logger.ErrorContext(ctx, "undecodable event", slog.String("error", err.Error()))
		_ = msg.Ack()
		return
	}

	if !Handles(env) {
		p.logger.WarnContext(ctx, "unimplemented event version",
			slog.String("event_id", env.ID),
			slog.String("type", env.Type),
			slog.Int("version", env.Version),
		)
		_ = msg.Ack()
		return
	}

	if err := p.apply(ctx, env); err != nil {
		p.logger.ErrorContext(ctx, "project event",
			slog.String("event_id", env.ID),
			slog.String("error", err.Error()),
		)
		_ = msg.Nak()
		return
	}
	_ = msg.Ack()
}

var tracer = otel.Tracer("orders/reporting")

// The producing span ended when the request returned, so its context is a link
// rather than a parent: claiming it as a parent would misstate when the work
// happened.
func (p *Projector) apply(ctx context.Context, env outbox.Envelope) error {
	opts := []trace.SpanStartOption{trace.WithSpanKind(trace.SpanKindConsumer)}
	if env.Traceparent != "" {
		producing := otel.GetTextMapPropagator().Extract(context.Background(),
			propagation.MapCarrier{"traceparent": env.Traceparent})
		opts = append(opts, trace.WithLinks(trace.LinkFromContext(producing)))
	}

	ctx, span := tracer.Start(ctx, "projection.apply", opts...)
	defer span.End()
	span.SetAttributes(
		attribute.String("event.id", env.ID),
		attribute.String("event.type", env.Type),
	)

	if err := p.repo.Apply(ctx, env); err != nil {
		span.RecordError(err)
		return err
	}
	return nil
}
