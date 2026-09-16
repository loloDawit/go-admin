package reporting

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/loloDawit/go-admin/services/orders/internal/errs"
	"github.com/loloDawit/go-admin/services/orders/internal/outbox"
)

// uniqueViolationCode is Postgres's SQLSTATE for a duplicate key.
const uniqueViolationCode = "23505"

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// Apply records the event as processed in the same transaction that changes the
// projection. Revenue is not naturally idempotent — applying it twice doubles
// it — so a redelivery must conflict here and roll the whole thing back.
func (r *PostgresRepository) Apply(ctx context.Context, env outbox.Envelope) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return errs.Wrap(errs.OpProjectEvent, err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, insertProcessedStmt, env.ID); err != nil {
		if isUniqueViolation(err) {
			return nil
		}
		return errs.Wrap(errs.OpProjectEvent, err)
	}

	if err := project(ctx, tx, env); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func project(ctx context.Context, tx pgx.Tx, env outbox.Envelope) error {
	day := Day(env.OccurredAt)

	switch env.Type {
	case outbox.TypeOrderCreated:
		created, err := ParseCreated(env)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, upsertPlacedStmt, day, created.Currency)
		return err

	case outbox.TypeOrderStatusChanged:
		change, err := ParseStatusChange(env)
		if err != nil {
			return err
		}
		orderID, err := strconv.ParseInt(env.AggregateID, 10, 64)
		if err != nil {
			return errs.Wrap(errs.OpProjectEvent, err)
		}
		switch change.To {
		case "paid":
			if _, err := tx.Exec(ctx, upsertRecognisedStmt, day, change.Currency, change.TotalMinor); err != nil {
				return err
			}
			_, err := tx.Exec(ctx, insertRecognisedOrderStmt, orderID, day, change.Currency, change.TotalMinor)
			return err
		case "refunded":
			return reverse(ctx, tx, orderID)
		default:
			return nil
		}

	default:
		return nil
	}
}

// reverse credits the day the revenue was recognised on, not the day of the
// refund. An order with nothing recognised is not an error: it was never paid,
// or its events arrived out of order, and either way there is nothing to undo.
func reverse(ctx context.Context, tx pgx.Tx, orderID int64) error {
	var day time.Time
	var currency string
	var amount int64
	err := tx.QueryRow(ctx, takeRecognisedOrderStmt, orderID).Scan(&day, &currency, &amount)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return errs.Wrap(errs.OpProjectEvent, err)
	}
	_, err = tx.Exec(ctx, upsertRefundedStmt, day, currency, amount)
	return err
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode
}
