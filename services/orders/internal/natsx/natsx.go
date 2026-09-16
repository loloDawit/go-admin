// Package natsx is the one place that knows this service's JetStream shape:
// the stream, the consumer, and the settings the ordering and dedup guarantees
// depend on.
package natsx

import (
	"context"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/loloDawit/go-admin/services/orders/internal/errs"
)

const (
	StreamName    = "ORDERS"
	SubjectPrefix = "orders."
	ConsumerName  = "orders-reporting"
)

// MsgIDHeader carries the envelope id, which is how the stream discards a
// publisher retry that follows a lost acknowledgement.
const MsgIDHeader = "Nats-Msg-Id"

func Connect(_ context.Context, url string) (*nats.Conn, jetstream.JetStream, error) {
	conn, err := nats.Connect(url, nats.RetryOnFailedConnect(true), nats.MaxReconnects(-1))
	if err != nil {
		return nil, nil, errs.Wrap(errs.OpConnectBroker, err)
	}
	js, err := jetstream.New(conn)
	if err != nil {
		conn.Close()
		return nil, nil, errs.Wrap(errs.OpConnectBroker, err)
	}
	return conn, js, nil
}

func EnsureStream(ctx context.Context, js jetstream.JetStream, duplicates time.Duration) (jetstream.Stream, error) {
	stream, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:       StreamName,
		Subjects:   []string{SubjectPrefix + ">"},
		Retention:  jetstream.LimitsPolicy,
		Duplicates: duplicates,
	})
	if err != nil {
		return nil, errs.Wrap(errs.OpEnsureStream, err)
	}
	return stream, nil
}

// MaxAckPending 1 is not a throughput setting: with one publisher sending in
// outbox-id order it is what keeps paid from being applied after refunded.
func EnsureConsumer(ctx context.Context, stream jetstream.Stream) (jetstream.Consumer, error) {
	consumer, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:       ConsumerName,
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxAckPending: 1,
	})
	if err != nil {
		return nil, errs.Wrap(errs.OpEnsureConsumer, err)
	}
	return consumer, nil
}
