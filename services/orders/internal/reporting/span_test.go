package reporting

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/loloDawit/go-admin/services/orders/internal/outbox"
)

type applierFunc func(context.Context, outbox.Envelope) (bool, error)

func (f applierFunc) Apply(ctx context.Context, env outbox.Envelope) (bool, error) {
	return f(ctx, env)
}

func recordedApply(t *testing.T, env outbox.Envelope) tracetest.SpanStub {
	t.Helper()
	return recordedApplyWith(t, env, func(context.Context, outbox.Envelope) (bool, error) { return true, nil })
}

func recordedApplyWith(t *testing.T, env outbox.Envelope, apply applierFunc) tracetest.SpanStub {
	t.Helper()
	otel.SetTextMapPropagator(propagation.TraceContext{})
	recorder := tracetest.NewSpanRecorder()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder)))
	tracer = otel.Tracer("orders/reporting")

	metrics, err := NewMetrics()
	if err != nil {
		t.Fatalf("NewMetrics: %v", err)
	}
	p := NewProjector(apply, nil, slog.New(slog.DiscardHandler), metrics)
	if err := p.apply(context.Background(), env); err != nil {
		t.Fatalf("apply: %v", err)
	}

	spans := tracetest.SpanStubsFromReadOnlySpans(recorder.Ended())
	if len(spans) != 1 {
		t.Fatalf("recorded %d spans, want 1", len(spans))
	}
	return spans[0]
}

// A link, not a parent: the producing span ended when the request returned, and
// claiming it as a parent would misstate when the work happened.
func TestApplyLinksToTheProducingTrace(t *testing.T) {
	span := recordedApply(t, outbox.Envelope{
		ID: "e1", Type: outbox.TypeOrderCreated, Version: 1,
		Traceparent: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
		Payload:     json.RawMessage(`{}`),
	})

	if len(span.Links) != 1 {
		t.Fatalf("span has %d links, want 1", len(span.Links))
	}
	if got := span.Links[0].SpanContext.TraceID().String(); got != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Fatalf("linked trace = %s", got)
	}
	if span.Parent.IsValid() {
		t.Fatal("the producing trace was attached as a parent, not a link")
	}
}

// An envelope written before this field existed has no traceparent and must
// still be projected, unlinked rather than refused.
func TestApplyWithoutATraceparentIsUnlinked(t *testing.T) {
	span := recordedApply(t, outbox.Envelope{
		ID: "e2", Type: outbox.TypeOrderCreated, Version: 1, Payload: json.RawMessage(`{}`),
	})

	if len(span.Links) != 0 {
		t.Fatalf("span has %d links, want none", len(span.Links))
	}
}
