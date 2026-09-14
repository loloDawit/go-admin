// Package platformcheck is temporary scaffolding proving the gateway -> service -> database path; do not build on it.
package platformcheck

import "github.com/loloDawit/go-admin/services/catalog/internal/errs"

type SchemaState struct {
	Version int
	Dirty   bool
}

// Aliased from errs, never redeclared: a second errors.New with identical text would be a distinct value that errors.Is stops matching.
var (
	ErrNoMigrations        = errs.ErrNoMigrations
	ErrDirtySchema         = errs.ErrDirtySchema
	ErrDatabaseUnavailable = errs.ErrDatabaseUnavailable
)
