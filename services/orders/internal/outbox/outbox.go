// Package outbox is transport, not domain state: a row here is a serialised
// envelope waiting to be published and then purged. order_events is the durable
// record the UI reads; the two are deliberately separate tables.
package outbox

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

const (
	TypeOrderCreated       = "order.created"
	TypeOrderStatusChanged = "order.status_changed"
)

// Version is this service's envelope version. A consumer that meets a higher
// one acknowledges and logs rather than retrying forever.
const Version = 1

// Envelope is interpretable from itself: the money travels with the event so a
// later price correction cannot change what was already recognised.
type Envelope struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Version     int       `json:"version"`
	OccurredAt  time.Time `json:"occurredAt"`
	AggregateID string    `json:"aggregateId"`
	ActorID     string    `json:"actorId"`
	// Traceparent is captured when the row is written, because the span that
	// wrote it has ended by the time the publisher runs.
	Traceparent string          `json:"traceparent,omitempty"`
	Payload     json.RawMessage `json:"payload"`
}

// Record is what the writing transaction inserts.
type Record struct {
	EventID string
	Subject string
	Payload []byte
}

// Stored is an unpublished row as the publisher reads it back.
type Stored struct {
	ID      int64
	EventID string
	Subject string
	Payload []byte
}

// Subject is derived from the type so the two cannot drift: order.created
// publishes to orders.created.
func Subject(eventType string) string {
	return "orders." + eventType[len("order."):]
}

// The id is the dedup key on both sides — the stream's Nats-Msg-Id and the
// consumer's processed_events primary key — so it is minted once, here, and
// never derived from anything that could repeat.
func newEventID() string { return uuid.NewString() }

func New(ctx context.Context, eventType, aggregateID, actorID string, occurredAt time.Time, payload any) (Record, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return Record{}, err
	}
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)

	id := newEventID()
	envelope, err := json.Marshal(Envelope{
		ID:          id,
		Type:        eventType,
		Version:     Version,
		OccurredAt:  occurredAt,
		AggregateID: aggregateID,
		ActorID:     actorID,
		Traceparent: carrier["traceparent"],
		Payload:     body,
	})
	if err != nil {
		return Record{}, err
	}
	return Record{EventID: id, Subject: Subject(eventType), Payload: envelope}, nil
}
