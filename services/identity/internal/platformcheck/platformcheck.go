// Package platformcheck is the M1 walking skeleton and is TEMPORARY. It proves
// the gateway -> service -> service-owned database path works. It is removed
// when this service gains its real capabilities in M2. Do not build on it.
package platformcheck

import "errors"

type SchemaState struct {
	Version int
	Dirty   bool
}

var (
	ErrNoMigrations = errors.New("no migrations applied")
	ErrDirtySchema  = errors.New("schema is in a dirty state")
)
