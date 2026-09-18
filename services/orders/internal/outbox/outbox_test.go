package outbox_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/loloDawit/go-admin/services/orders/internal/outbox"
)

// The span that wrote the row has ended by the time the publisher runs, so the
// context has to travel inside the envelope or the link is unreconstructable.
func TestNewCapturesTheTraceContext(t *testing.T) {
	otel.SetTextMapPropagator(propagation.TraceContext{})

	traceID, err := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	if err != nil {
		t.Fatalf("trace id: %v", err)
	}
	spanID, err := trace.SpanIDFromHex("00f067aa0ba902b7")
	if err != nil {
		t.Fatalf("span id: %v", err)
	}
	ctx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: traceID, SpanID: spanID, TraceFlags: trace.FlagsSampled,
	}))

	rec, err := outbox.New(ctx, outbox.TypeOrderCreated, "34", "3", time.Now(), map[string]any{"totalMinor": 100})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	var env outbox.Envelope
	if err := json.Unmarshal(rec.Payload, &env); err != nil {
		t.Fatalf("envelope: %v", err)
	}
	if !strings.Contains(env.Traceparent, "4bf92f3577b34da6a3ce929d0e0e4736") {
		t.Fatalf("traceparent = %q", env.Traceparent)
	}
}

// An event written outside a span has no trace context, and must serialise
// without an empty field that a consumer would try to parse.
func TestNewWithoutASpanOmitsTheTraceparent(t *testing.T) {
	otel.SetTextMapPropagator(propagation.TraceContext{})

	rec, err := outbox.New(context.Background(), outbox.TypeOrderCreated, "34", "3", time.Now(), map[string]any{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if strings.Contains(string(rec.Payload), "traceparent") {
		t.Fatalf("payload carries an empty traceparent: %s", rec.Payload)
	}
}
