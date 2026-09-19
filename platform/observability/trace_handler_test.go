package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"go.opentelemetry.io/otel/trace"
)

func decode(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	var record map[string]any
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatalf("log record is not JSON: %v", err)
	}
	return record
}

func sampledContext(t *testing.T) context.Context {
	t.Helper()
	traceID, err := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	if err != nil {
		t.Fatalf("trace id: %v", err)
	}
	spanID, err := trace.SpanIDFromHex("00f067aa0ba902b7")
	if err != nil {
		t.Fatalf("span id: %v", err)
	}
	return trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: traceID, SpanID: spanID, TraceFlags: trace.FlagsSampled,
	}))
}

func TestLogInsideSpanCarriesTraceID(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger("test", &buf)

	logger.InfoContext(sampledContext(t), "hello")

	record := decode(t, &buf)
	if record["trace_id"] != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Fatalf("trace_id = %v", record["trace_id"])
	}
	if record["span_id"] != "00f067aa0ba902b7" {
		t.Fatalf("span_id = %v", record["span_id"])
	}
}

// A log line outside a span must still be a log line: the handler cannot
// require a span to exist.
func TestLogOutsideSpanHasNoTraceID(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger("test", &buf)

	logger.InfoContext(context.Background(), "hello")

	record := decode(t, &buf)
	if _, ok := record["trace_id"]; ok {
		t.Fatal("trace_id present outside a span")
	}
	if record["msg"] != "hello" {
		t.Fatalf("msg = %v", record["msg"])
	}
}
