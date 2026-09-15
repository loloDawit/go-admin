// Package schemacheck reports whether this service's schema is migrated and clean; it supplies the readiness probe.
package schemacheck

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
