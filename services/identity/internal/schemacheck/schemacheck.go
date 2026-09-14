// Package schemacheck reports whether this service's schema is migrated and
// clean. It supplies the readiness probe; the /_platform route it once served
// was retired when identity gained real routes.
package schemacheck

import "github.com/loloDawit/go-admin/services/identity/internal/errs"

type SchemaState struct {
	Version int
	Dirty   bool
}

// These re-export the identity service's registered sentinels (see
// internal/errs) under this package's existing names.
var (
	ErrNoMigrations        = errs.ErrNoMigrations
	ErrDirtySchema         = errs.ErrDirtySchema
	ErrDatabaseUnavailable = errs.ErrDatabaseUnavailable
)
