// Package readiness is the non-temporary home for the /readyz route. Spec §4
// deletes services/*/internal/platformcheck in M2 ("readiness checking
// survives; the route and the package do not"): if the handler stayed inside
// platformcheck, deleting that package would silently remove /readyz while
// the Dockerfile HEALTHCHECK (which only probes /healthz) kept passing.
package readiness

import "context"

// Probe reports whether this process can currently serve traffic. M1 wires
// platformcheck.Service.Probe in; M2 supplies whatever probe its real
// capabilities need without touching this package.
type Probe func(ctx context.Context) error
