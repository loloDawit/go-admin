// Package readiness owns /readyz separately from whatever supplies the probe, so retiring a schema-check package cannot silently take the route with it.
package readiness

import "context"

// Probe reports whether this process can currently serve traffic.
type Probe func(ctx context.Context) error
