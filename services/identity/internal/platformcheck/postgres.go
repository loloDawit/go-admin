package platformcheck

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// schema_migrations is created by golang-migrate. Its absence means migrations
// have never run against this database.
func (r *PostgresRepository) SchemaState(ctx context.Context) (SchemaState, error) {
	var state SchemaState

	err := r.pool.QueryRow(ctx, `SELECT version, dirty FROM schema_migrations LIMIT 1`).
		Scan(&state.Version, &state.Dirty)

	if errors.Is(err, pgx.ErrNoRows) {
		return SchemaState{}, nil
	}
	if err != nil {
		return SchemaState{}, err
	}
	return state, nil
}
