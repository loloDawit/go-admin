package platformcheck

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// undefinedTable is Postgres SQLSTATE 42P01, returned when schema_migrations
// itself does not exist yet, as distinct from the table existing with zero
// rows (pgx.ErrNoRows).
const undefinedTable = "42P01"

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// schema_migrations is created by golang-migrate. Its absence (SQLSTATE
// 42P01) means migrations have never run against this database; a present but
// empty table (pgx.ErrNoRows) means the same thing. Both map to the zero
// SchemaState, which Service.Check turns into ErrNoMigrations.
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
