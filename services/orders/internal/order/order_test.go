package order

import "testing"

// The discriminator for point 4 of the brief: SQL lpad truncates past six
// digits (1,000,000 -> "100000", colliding with order 100,000); this must not.
func TestFormatOrderNumberDoesNotTruncatePastSixDigits(t *testing.T) {
	got := formatOrderNumber(2026, 1000000)
	want := "ORD-2026-1000000"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestFormatOrderNumberPadsToSixDigits(t *testing.T) {
	got := formatOrderNumber(2026, 42)
	want := "ORD-2026-000042"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
