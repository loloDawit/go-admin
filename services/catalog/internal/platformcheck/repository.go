package platformcheck

import "context"

type Repository interface {
	SchemaState(ctx context.Context) (SchemaState, error)
}
