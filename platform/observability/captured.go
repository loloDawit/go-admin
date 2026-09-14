package observability

import (
	"context"
	"log/slog"
	"sync"
)

// Captured records log records in memory so tests can assert structured
// attributes rather than parsing rendered output.
type Captured struct {
	shared *capturedState
	attrs  []slog.Attr
	groups []string
}

// capturedState is shared by a Captured and every handler derived from it via
// WithAttrs/WithGroup, so records logged through any of them land in one list.
type capturedState struct {
	mu      sync.Mutex
	records []slog.Record
}

func NewCaptured() (*slog.Logger, *Captured) {
	c := &Captured{shared: &capturedState{}}
	return slog.New(c), c
}

func (c *Captured) Records() []slog.Record {
	c.shared.mu.Lock()
	defer c.shared.mu.Unlock()
	return append([]slog.Record(nil), c.shared.records...)
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
	recordAttrs := make([]slog.Attr, 0, r.NumAttrs())
	r.Attrs(func(a slog.Attr) bool {
		recordAttrs = append(recordAttrs, a)
		return true
	})

	merged := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)
	merged.AddAttrs(c.attrs...)
	merged.AddAttrs(c.nest(recordAttrs)...)

	c.shared.mu.Lock()
	defer c.shared.mu.Unlock()
	c.shared.records = append(c.shared.records, merged)
	return nil
}

// WithAttrs mirrors the real JSONHandler: accumulated attrs carry forward to every record logged through the returned handler.
func (c *Captured) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return c
	}
	next := c.clone()
	next.attrs = append(next.attrs, c.nest(attrs)...)
	return next
}

func (c *Captured) WithGroup(name string) slog.Handler {
	if name == "" {
		return c
	}
	next := c.clone()
	next.groups = append(next.groups, name)
	return next
}

func (c *Captured) clone() *Captured {
	return &Captured{
		shared: c.shared,
		attrs:  append([]slog.Attr(nil), c.attrs...),
		groups: append([]string(nil), c.groups...),
	}
}

// nest wraps attrs under every currently open group, innermost group last.
func (c *Captured) nest(attrs []slog.Attr) []slog.Attr {
	wrapped := attrs
	for i := len(c.groups) - 1; i >= 0; i-- {
		wrapped = []slog.Attr{slog.Attr{Key: c.groups[i], Value: slog.GroupValue(wrapped...)}}
	}
	return wrapped
}
