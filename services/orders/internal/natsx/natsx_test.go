package natsx_test

import (
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	natstest "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"

	"github.com/loloDawit/go-admin/services/orders/internal/natsx"
)

// Port -1 asks the OS for a free one, so parallel packages do not collide.
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

func TestEnsureStreamAndConsumerAreIdempotent(t *testing.T) {
	s := runJetStream(t)

	conn, js, err := natsx.Connect(t.Context(), s.ClientURL())
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Cleanup(conn.Close)

	for i := range 2 {
		stream, err := natsx.EnsureStream(t.Context(), js, time.Minute)
		if err != nil {
			t.Fatalf("EnsureStream call %d: %v", i, err)
		}
		if _, err := natsx.EnsureConsumer(t.Context(), stream); err != nil {
			t.Fatalf("EnsureConsumer call %d: %v", i, err)
		}
	}

	stream, err := js.Stream(t.Context(), natsx.StreamName)
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	info, err := stream.Info(t.Context())
	if err != nil {
		t.Fatalf("Info: %v", err)
	}
	if info.Config.Duplicates != time.Minute {
		t.Fatalf("duplicate window = %v, want 1m", info.Config.Duplicates)
	}
}

// The duplicate window is what discards a publisher retry after a lost ack, so
// a stream created without it would drop the first dedup layer silently.
func TestDuplicateMessageIDIsDiscarded(t *testing.T) {
	s := runJetStream(t)

	conn, js, err := natsx.Connect(t.Context(), s.ClientURL())
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Cleanup(conn.Close)

	if _, err := natsx.EnsureStream(t.Context(), js, time.Minute); err != nil {
		t.Fatalf("EnsureStream: %v", err)
	}

	msg := &nats.Msg{
		Subject: natsx.SubjectPrefix + "created",
		Data:    []byte(`{"id":"dup-1"}`),
		Header:  nats.Header{natsx.MsgIDHeader: []string{"dup-1"}},
	}
	if _, err := js.PublishMsg(t.Context(), msg); err != nil {
		t.Fatalf("first publish: %v", err)
	}
	ack, err := js.PublishMsg(t.Context(), msg)
	if err != nil {
		t.Fatalf("second publish: %v", err)
	}
	if !ack.Duplicate {
		t.Fatal("second publish of the same Nats-Msg-Id was not reported as a duplicate")
	}
}
