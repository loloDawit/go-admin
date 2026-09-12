package observability

import (
	"context"
	"log/slog"
	"sync"
)

// Captured records log records in memory so tests can assert structured
// attributes rather than parsing rendered output.
type Captured struct {
	mu      sync.Mutex
	records []slog.Record
}

func NewCaptured() (*slog.Logger, *Captured) {
	c := &Captured{}
	return slog.New(c), c
}

func (c *Captured) Records() []slog.Record {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]slog.Record(nil), c.records...)
}

func (c *Captured) Attr(i int, key string) (slog.Value, bool) {
	records := c.Records()
	if i >= len(records) {
		return slog.Value{}, false
	}
	var (
		found slog.Value
		ok    bool
	)
	records[i].Attrs(func(a slog.Attr) bool {
		if a.Key == key {
			found, ok = a.Value, true
			return false
		}
		return true
	})
	return found, ok
}

func (c *Captured) Enabled(context.Context, slog.Level) bool { return true }

func (c *Captured) Handle(_ context.Context, r slog.Record) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.records = append(c.records, r.Clone())
	return nil
}

func (c *Captured) WithAttrs([]slog.Attr) slog.Handler { return c }
func (c *Captured) WithGroup(string) slog.Handler      { return c }
