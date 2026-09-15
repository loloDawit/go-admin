package order

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/loloDawit/go-admin/services/orders/internal/errs"
)

// foreignKeyViolationCode is Postgres's SQLSTATE for a reference to a row that does not exist.
const foreignKeyViolationCode = "23503"

// querier is what *pgxpool.Pool and pgx.Tx both satisfy.
type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

type PostgresRepository struct {
	pool *pgxpool.Pool
	q    querier
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool, q: pool}
}

// RunInTx gives fn a repository bound to one transaction, so the sequence
// read and every insert it drives commit or roll back together.
func (r *PostgresRepository) RunInTx(ctx context.Context, fn func(Repository) error) error {
	if r.pool == nil {
		return fn(r)
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := fn(&PostgresRepository{q: tx}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PostgresRepository) NextOrderNumber(ctx context.Context) (int64, time.Time, error) {
	var seq int64
	var at time.Time
	err := r.q.QueryRow(ctx, nextOrderNumberQuery).Scan(&seq, &at)
	return seq, at, err
}

func (r *PostgresRepository) InsertOrder(ctx context.Context, in NewOrder) (Order, error) {
	var o Order
	var status string
	row := r.q.QueryRow(ctx, insertOrderStmt, in.Number, in.CustomerID, in.TotalMinor, in.Currency, in.PlacedAt)
	err := row.Scan(&o.ID, &o.Number, &o.CustomerID, &status, &o.TotalMinor, &o.Currency, &o.PlacedAt, &o.UpdatedAt)
	if isForeignKeyViolation(err) {
		return Order{}, errs.ErrCustomerNotFound
	}
	if err != nil {
		return Order{}, err
	}
	o.Status = Status(status)
	return o, nil
}

func (r *PostgresRepository) InsertItems(ctx context.Context, orderID int64, items []Item) ([]Item, error) {
	inserted := make([]Item, len(items))
	for i, it := range items {
		row := r.q.QueryRow(ctx, insertOrderItemStmt,
			orderID, it.ProductID, it.TitleSnapshot, it.UnitPriceMinor, it.Currency, it.Quantity, it.LineTotalMinor)
		if err := row.Scan(&it.ID); err != nil {
			return nil, err
		}
		inserted[i] = it
	}
	return inserted, nil
}

func (r *PostgresRepository) InsertEvent(ctx context.Context, in NewEvent) error {
	var fromStatus *string
	if in.FromStatus != nil {
		s := string(*in.FromStatus)
		fromStatus = &s
	}
	_, err := r.q.Exec(ctx, insertOrderEventStmt, in.OrderID, fromStatus, string(in.ToStatus), in.ActorID)
	return err
}

func (r *PostgresRepository) GetByID(ctx context.Context, id int64) (Order, error) {
	var o Order
	var status string
	err := r.q.QueryRow(ctx, getOrderByIDQuery, id).
		Scan(&o.ID, &o.Number, &o.CustomerID, &status, &o.TotalMinor, &o.Currency, &o.PlacedAt, &o.UpdatedAt)
	if isNoRows(err) {
		return Order{}, ErrOrderNotFound
	}
	if err != nil {
		return Order{}, err
	}
	o.Status = Status(status)

	items, err := r.itemsForOrder(ctx, id)
	if err != nil {
		return Order{}, err
	}
	o.Items = items
	return o, nil
}

func (r *PostgresRepository) itemsForOrder(ctx context.Context, orderID int64) ([]Item, error) {
	rows, err := r.q.Query(ctx, listOrderItemsQuery, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []Item{}
	for rows.Next() {
		var it Item
		if err := rows.Scan(&it.ID, &it.ProductID, &it.TitleSnapshot, &it.UnitPriceMinor, &it.Currency, &it.Quantity, &it.LineTotalMinor); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

func isNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == foreignKeyViolationCode
}
