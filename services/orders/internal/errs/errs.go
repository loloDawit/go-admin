// Package errs is the orders service's one error registry: every sentinel
// and operation string a wrap names lives here. httperr maps the ones it
// recognizes to a client-facing status; the rest are startup-only and never
// reach httperr, since they can only occur before the server starts serving
// requests.
package errs

import (
	"errors"
	"fmt"
)

// Sentinels. Where httperr maps one to a status code, nothing else decides one.
var (
	// httperr maps all three to 503: each means not ready, never a request fault.
	ErrNoMigrations        = errors.New("no migrations applied")
	ErrDirtySchema         = errors.New("schema is in a dirty state")
	ErrDatabaseUnavailable = errors.New("database is unavailable")

	// ErrCatalogUnavailable: Catalog could not be reached, or answered with a 5xx.
	ErrCatalogUnavailable = errors.New("catalog is unavailable")
	// ErrCatalogTimeout: the call to Catalog exceeded its configured deadline. Distinct from ErrCatalogUnavailable so the two can be mapped to different statuses.
	ErrCatalogTimeout = errors.New("catalog request timed out")
	// ErrCatalogRejected: Catalog answered with a 4xx, or a body this client could not parse.
	ErrCatalogRejected = errors.New("catalog rejected the request")
)

// Operation strings for Wrap, named here so a log line and its test expectation cannot drift independently.
const (
	OpResolveProducts = "resolve products from catalog"
)

// Wrap names the step that failed, so a log line reads "resolve products from catalog: connection refused", not a bare driver message.
func Wrap(op string, err error) error {
	return fmt.Errorf("%s: %w", op, err)
}
