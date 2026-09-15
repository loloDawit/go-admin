package schemacheck

import "context"

type Repository interface {
	SchemaState(ctx context.Context) (SchemaState, error)
}
