package outbox

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/loloDawit/go-admin/services/orders/internal/errs"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Unpublished(ctx context.Context, limit int) ([]Stored, error) {
	rows, err := r.pool.Query(ctx, unpublishedQuery, limit)
	if err != nil {
		return nil, errs.Wrap(errs.OpReadOutbox, err)
	}
	defer rows.Close()

	out := []Stored{}
	for rows.Next() {
		var s Stored
		if err := rows.Scan(&s.ID, &s.EventID, &s.Subject, &s.Payload); err != nil {
			return nil, errs.Wrap(errs.OpReadOutbox, err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) MarkPublished(ctx context.Context, id int64) error {
	if _, err := r.pool.Exec(ctx, markPublishedStmt, id); err != nil {
		return errs.Wrap(errs.OpMarkPublished, err)
	}
	return nil
}
