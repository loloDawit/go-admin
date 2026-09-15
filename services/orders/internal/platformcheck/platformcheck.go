// Package platformcheck is temporary scaffolding proving the gateway -> service -> database path; do not build on it.
package platformcheck

import "github.com/loloDawit/go-admin/services/orders/internal/errs"

type SchemaState struct {
	Version int
	Dirty   bool
}

// Aliased to internal/errs, the registry: a second errors.New with identical
// text would be a distinct value, and errors.Is would silently stop matching.
var (
	ErrNoMigrations        = errs.ErrNoMigrations
	ErrDirtySchema         = errs.ErrDirtySchema
	ErrDatabaseUnavailable = errs.ErrDatabaseUnavailable
)
