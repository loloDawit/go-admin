package reporting_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/loloDawit/go-admin/services/orders/internal/outbox"
	"github.com/loloDawit/go-admin/services/orders/internal/reporting"
)

func envelope(t *testing.T, id, eventType, aggregateID string, version int, at time.Time, payload any) outbox.Envelope {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return outbox.Envelope{
		ID: id, Type: eventType, Version: version,
		OccurredAt: at, AggregateID: aggregateID, ActorID: "3",
		Payload: body,
	}
}

// A version this build does not implement cannot be made implementable by a
// redelivery, so it is acknowledged and logged rather than retried forever.
func TestUnknownVersionIsNotHandled(t *testing.T) {
	day := time.Date(2026, 9, 16, 10, 0, 0, 0, time.UTC)
	payload := map[string]any{"totalMinor": 100, "currency": "USD"}

	ahead := envelope(t, "future-1", outbox.TypeOrderCreated, "1", outbox.Version+1, day, payload)
	if reporting.Handles(ahead) {
		t.Fatal("an envelope one version ahead was reported as handled")
	}

	current := envelope(t, "now-1", outbox.TypeOrderCreated, "1", outbox.Version, day, payload)
	if !reporting.Handles(current) {
		t.Fatal("an envelope at the current version was not reported as handled")
	}
}

func TestStatusChangeCarriesTheMoneyItRecognises(t *testing.T) {
	day := time.Date(2026, 9, 16, 10, 0, 0, 0, time.UTC)
	env := envelope(t, "sc-1", outbox.TypeOrderStatusChanged, "1", outbox.Version, day,
		map[string]any{"from": "paid", "to": "refunded", "totalMinor": 500, "currency": "USD"})

	change, err := reporting.ParseStatusChange(env)
	if err != nil {
		t.Fatalf("ParseStatusChange: %v", err)
	}
	if change.From != "paid" || change.To != "refunded" || change.TotalMinor != 500 || change.Currency != "USD" {
		t.Fatalf("change = %+v", change)
	}
}

// UTC truncation is what keeps a day key stable regardless of the server's zone.
func TestDayIsTheUTCDate(t *testing.T) {
	late := time.Date(2026, 9, 16, 23, 30, 0, 0, time.FixedZone("ahead", 3*60*60))
	if got := reporting.Day(late); got.Format(time.DateOnly) != "2026-09-16" {
		t.Fatalf("Day = %s, want 2026-09-16", got.Format(time.DateOnly))
	}
}
