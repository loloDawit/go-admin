package platformcheck

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// undefinedTable is Postgres SQLSTATE 42P01: schema_migrations itself does not exist yet, distinct from an empty table (pgx.ErrNoRows).
const undefinedTable = "42P01"

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// A missing table (42P01) and a present-but-empty one (pgx.ErrNoRows) both mean migrations have never run; both map to the zero SchemaState.
func (r *PostgresRepository) SchemaState(ctx context.Context) (SchemaState, error) {
	var state SchemaState

	err := r.pool.QueryRow(ctx, `SELECT version, dirty FROM schema_migrations LIMIT 1`).
		Scan(&state.Version, &state.Dirty)

	if errors.Is(err, pgx.ErrNoRows) || isUndefinedTable(err) {
		return SchemaState{}, nil
	}
	if err != nil {
		return SchemaState{}, err
	}
	return state, nil
}

func isUndefinedTable(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == undefinedTable
}
