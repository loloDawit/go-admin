package outbox

import (
	"context"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

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
}

func NewPublisher(store PublisherStore, js jetstream.JetStream, batch int, interval time.Duration, logger *slog.Logger) *Publisher {
	return &Publisher{store: store, js: js, batch: batch, interval: interval, logger: logger}
}

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
		msg := &nats.Msg{
			Subject: row.Subject,
			Data:    row.Payload,
			Header:  nats.Header{natsx.MsgIDHeader: []string{row.EventID}},
		}
		if _, err := p.js.PublishMsg(ctx, msg); err != nil {
			return errs.Wrap(errs.OpPublishEvent, err)
		}
		if err := p.store.MarkPublished(ctx, row.ID); err != nil {
			return err
		}
	}
	return nil
}
