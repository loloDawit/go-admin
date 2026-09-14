// Package platformcheck is temporary scaffolding proving the gateway -> service -> database path; do not build on it.
package platformcheck

import "errors"

type SchemaState struct {
	Version int
	Dirty   bool
}

var (
	ErrNoMigrations        = errors.New("no migrations applied")
	ErrDirtySchema         = errors.New("schema is in a dirty state")
	ErrDatabaseUnavailable = errors.New("database is unavailable")
)
