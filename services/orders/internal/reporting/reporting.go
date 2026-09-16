// Package reporting maintains the revenue history §15.7 requires and today's
// API cannot produce. Status counts and recent orders are plain queries: only
// revenue over time is otherwise unproducible, and only it is projected.
package reporting

import (
	"encoding/json"
	"time"

	"github.com/loloDawit/go-admin/services/orders/internal/errs"
	"github.com/loloDawit/go-admin/services/orders/internal/outbox"
)

// Handles reports whether this build implements the envelope's version.
func Handles(env outbox.Envelope) bool {
	return env.Version == outbox.Version
}

type Created struct {
	TotalMinor int64
	Currency   string
}

type StatusChange struct {
	From       string
	To         string
	TotalMinor int64
	Currency   string
}

func ParseCreated(env outbox.Envelope) (Created, error) {
	var raw struct {
		TotalMinor int64  `json:"totalMinor"`
		Currency   string `json:"currency"`
	}
	if err := json.Unmarshal(env.Payload, &raw); err != nil {
		return Created{}, errs.Wrap(errs.OpProjectEvent, err)
	}
	return Created{TotalMinor: raw.TotalMinor, Currency: raw.Currency}, nil
}

func ParseStatusChange(env outbox.Envelope) (StatusChange, error) {
	var raw struct {
		From       string `json:"from"`
		To         string `json:"to"`
		TotalMinor int64  `json:"totalMinor"`
		Currency   string `json:"currency"`
	}
	if err := json.Unmarshal(env.Payload, &raw); err != nil {
		return StatusChange{}, errs.Wrap(errs.OpProjectEvent, err)
	}
	return StatusChange(raw), nil
}

// Day is the projection's key alongside currency: revenue is never summed
// across currencies, the rule LifetimeValue already enforces by refusing.
func Day(at time.Time) time.Time {
	return at.UTC().Truncate(24 * time.Hour)
}
