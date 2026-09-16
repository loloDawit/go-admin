package outbox_test

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	natstest "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/loloDawit/go-admin/services/orders/internal/natsx"
	"github.com/loloDawit/go-admin/services/orders/internal/outbox"
)

func runJetStream(t *testing.T) *server.Server {
	t.Helper()
	opts := natstest.DefaultTestOptions
	opts.Port = -1
	opts.JetStream = true
	opts.StoreDir = t.TempDir()
	s := natstest.RunServer(&opts)
	t.Cleanup(s.Shutdown)
	return s
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

type fakeStore struct {
	mu        sync.Mutex
	rows      []outbox.Stored
	published []int64
}

func (f *fakeStore) Unpublished(context.Context, int) ([]outbox.Stored, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]outbox.Stored, 0, len(f.rows))
	for _, r := range f.rows {
		if !contains(f.published, r.ID) {
			out = append(out, r)
		}
	}
	return out, nil
}

func (f *fakeStore) MarkPublished(_ context.Context, id int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.published = append(f.published, id)
	return nil
}

func (f *fakeStore) publishedIDs() []int64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]int64(nil), f.published...)
}

func contains(ids []int64, id int64) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}

func TestPublisherSendsInIDOrderAndMarksPublished(t *testing.T) {
	s := runJetStream(t)
	conn, js, err := natsx.Connect(t.Context(), s.ClientURL())
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Cleanup(conn.Close)
	stream, err := natsx.EnsureStream(t.Context(), js, time.Minute)
	if err != nil {
		t.Fatalf("EnsureStream: %v", err)
	}
	consumer, err := natsx.EnsureConsumer(t.Context(), stream)
	if err != nil {
		t.Fatalf("EnsureConsumer: %v", err)
	}

	store := &fakeStore{rows: []outbox.Stored{
		{ID: 1, EventID: "e1", Subject: "orders.created", Payload: []byte(`{"id":"e1"}`)},
		{ID: 2, EventID: "e2", Subject: "orders.status_changed", Payload: []byte(`{"id":"e2"}`)},
		{ID: 3, EventID: "e3", Subject: "orders.status_changed", Payload: []byte(`{"id":"e3"}`)},
	}}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	pub := outbox.NewPublisher(store, js, 10, 10*time.Millisecond, discardLogger())
	go func() { _ = pub.Run(ctx) }()

	var got []string
	for range 3 {
		batch, err := consumer.Fetch(1, jetstream.FetchMaxWait(5*time.Second))
		if err != nil {
			t.Fatalf("Fetch: %v", err)
		}
		for msg := range batch.Messages() {
			got = append(got, msg.Headers().Get(natsx.MsgIDHeader))
			_ = msg.Ack()
		}
	}

	want := []string{"e1", "e2", "e3"}
	if len(got) != 3 || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Fatalf("delivered %v, want %v", got, want)
	}
}

// published_at is written only after NATS acknowledges. Marking first would lose
// any event whose publish failed, which is the failure the outbox exists to
// prevent.
func TestPublisherDoesNotMarkWhatItCouldNotSend(t *testing.T) {
	s := runJetStream(t)
	conn, js, err := natsx.Connect(t.Context(), s.ClientURL())
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Cleanup(conn.Close)
	if _, err := natsx.EnsureStream(t.Context(), js, time.Minute); err != nil {
		t.Fatalf("EnsureStream: %v", err)
	}

	// A subject no stream matches is rejected by the server, so the publish
	// fails without needing the broker to be down.
	store := &fakeStore{rows: []outbox.Stored{
		{ID: 1, EventID: "e1", Subject: "elsewhere.created", Payload: []byte(`{"id":"e1"}`)},
	}}

	ctx, cancel := context.WithTimeout(t.Context(), 400*time.Millisecond)
	defer cancel()
	pub := outbox.NewPublisher(store, js, 10, 10*time.Millisecond, discardLogger())
	_ = pub.Run(ctx)

	if ids := store.publishedIDs(); len(ids) != 0 {
		t.Fatalf("marked %v published despite the publish failing", ids)
	}
}
