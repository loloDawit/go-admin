// Package platformcheck is the M1 walking skeleton and is TEMPORARY. It proves
// the gateway -> service -> service-owned database path works. It is removed
// when this service gains its real capabilities in M2. Do not build on it.
package platformcheck

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
