package order

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/loloDawit/go-admin/services/orders/internal/errs"
)

// orderSortColumns is the only place a caller's sort string reaches a
// column name: anything absent here is refused rather than interpolated.
var orderSortColumns = map[string]string{
	"placed_at":   "placed_at",
	"total_minor": "total_minor",
	"status":      "status",
	"number":      "number",
}

const defaultOrderSort = "-placed_at"

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
	_, err := r.q.Exec(ctx, insertOrderEventStmt, in.OrderID, fromStatus, string(in.ToStatus), in.ActorID, in.Reason)
	return err
}

// GetStatusForUpdate locks the row: no other transaction can write a new
// status until this one commits or rolls back.
func (r *PostgresRepository) GetStatusForUpdate(ctx context.Context, id int64) (Status, error) {
	var status string
	err := r.q.QueryRow(ctx, getOrderStatusForUpdateQuery, id).Scan(&status)
	if isNoRows(err) {
		return "", ErrOrderNotFound
	}
	if err != nil {
		return "", err
	}
	return Status(status), nil
}

// UpdateStatus's WHERE clause repeats the check the caller already made
// under GetStatusForUpdate's lock: belt and braces, not the only guard.
func (r *PostgresRepository) UpdateStatus(ctx context.Context, id int64, from, to Status) (Order, error) {
	var o Order
	var status string
	row := r.q.QueryRow(ctx, updateOrderStatusStmt, id, string(from), string(to))
	err := row.Scan(&o.ID, &o.Number, &o.CustomerID, &status, &o.TotalMinor, &o.Currency, &o.PlacedAt, &o.UpdatedAt)
	if isNoRows(err) {
		return Order{}, ErrOrderNotFound
	}
	if err != nil {
		return Order{}, err
	}
	o.Status = Status(status)
	return o, nil
}

func (r *PostgresRepository) ListOrders(ctx context.Context, q ListQuery) ([]Order, int, error) {
	col, desc, err := resolveOrderSort(q.Sort)
	if err != nil {
		return nil, 0, err
	}
	direction := "ASC"
	if desc {
		direction = "DESC"
	}
	status := orderStatusParam(q.Status)

	stmt := fmt.Sprintf(listOrdersQueryTemplate, col, direction)
	rows, err := r.q.Query(ctx, stmt, status, q.CustomerID, q.PageSize, offset(q.Page, q.PageSize))
	if err != nil {
		return nil, 0, err
	}
	items, err := scanOrders(rows)
	if err != nil {
		return nil, 0, err
	}

	var total int
	if err := r.q.QueryRow(ctx, listOrdersCountQuery, status, q.CustomerID).Scan(&total); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// resolveOrderSort maps a caller's sort string to a fixed column via
// orderSortColumns; anything absent from the map is refused, never
// interpolated.
func resolveOrderSort(sort string) (string, bool, error) {
	if sort == "" {
		sort = defaultOrderSort
	}
	desc := strings.HasPrefix(sort, "-")
	key := strings.TrimPrefix(sort, "-")

	col, ok := orderSortColumns[key]
	if !ok {
		return "", false, ErrInvalidSort
	}
	return col, desc, nil
}

func orderStatusParam(s *Status) *string {
	if s == nil {
		return nil
	}
	v := string(*s)
	return &v
}

func offset(page, pageSize int) int {
	return (page - 1) * pageSize
}

func scanOrders(rows pgx.Rows) ([]Order, error) {
	defer rows.Close()

	items := []Order{}
	for rows.Next() {
		var o Order
		var status string
		if err := rows.Scan(&o.ID, &o.Number, &o.CustomerID, &status, &o.TotalMinor, &o.Currency, &o.PlacedAt, &o.UpdatedAt); err != nil {
			return nil, err
		}
		o.Status = Status(status)
		items = append(items, o)
	}
	return items, rows.Err()
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

func (r *PostgresRepository) ListEvents(ctx context.Context, orderID int64) ([]Event, error) {
	rows, err := r.q.Query(ctx, listOrderEventsQuery, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := []Event{}
	for rows.Next() {
		var e Event
		var from *string
		var to string
		if err := rows.Scan(&e.ID, &from, &to, &e.ActorID, &e.Reason, &e.At); err != nil {
			return nil, err
		}
		if from != nil {
			s := Status(*from)
			e.FromStatus = &s
		}
		e.ToStatus = Status(to)
		events = append(events, e)
	}
	return events, rows.Err()
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
