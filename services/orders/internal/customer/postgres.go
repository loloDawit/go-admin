package customer

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// uniqueViolationCode is Postgres's SQLSTATE for a unique-constraint conflict.
const uniqueViolationCode = "23505"

// customerSortColumns is the only place a caller's sort string reaches a
// column name: anything absent here is refused rather than interpolated.
var customerSortColumns = map[string]string{
	"name":       "name",
	"email":      "email",
	"created_at": "created_at",
}

const defaultCustomerSort = "-created_at"

// querier is what *pgxpool.Pool and pgx.Tx both satisfy.
type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

type PostgresRepository struct {
	q querier
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{q: pool}
}

func (r *PostgresRepository) Create(ctx context.Context, in CreateCustomer) (Customer, error) {
	var c Customer
	row := r.q.QueryRow(ctx, createCustomerStmt, in.Email, in.Name)
	err := scanCustomer(row, &c)
	if isUniqueViolation(err) {
		return Customer{}, ErrCustomerEmailTaken
	}
	if err != nil {
		return Customer{}, err
	}
	return c, nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id int64) (Customer, error) {
	var c Customer
	row := r.q.QueryRow(ctx, getCustomerByIDQuery, id)
	err := scanCustomer(row, &c)
	if isNoRows(err) {
		return Customer{}, ErrCustomerNotFound
	}
	if err != nil {
		return Customer{}, err
	}
	return c, nil
}

func (r *PostgresRepository) GetByEmail(ctx context.Context, email string) (Customer, error) {
	var c Customer
	row := r.q.QueryRow(ctx, getCustomerByEmailQuery, email)
	err := scanCustomer(row, &c)
	if isNoRows(err) {
		return Customer{}, ErrCustomerNotFound
	}
	if err != nil {
		return Customer{}, err
	}
	return c, nil
}

func (r *PostgresRepository) List(ctx context.Context, q ListQuery) ([]Customer, int, error) {
	col, desc, err := resolveCustomerSort(q.Sort)
	if err != nil {
		return nil, 0, err
	}
	direction := "ASC"
	if desc {
		direction = "DESC"
	}
	search := searchParam(q.Q)

	stmt := listCustomersQueryPrefix + col + " " + direction + listCustomersQuerySuffix
	rows, err := r.q.Query(ctx, stmt, search, q.PageSize, offset(q.Page, q.PageSize))
	if err != nil {
		return nil, 0, err
	}
	items, err := scanCustomers(rows)
	if err != nil {
		return nil, 0, err
	}

	var total int
	if err := r.q.QueryRow(ctx, countCustomersQuery, search).Scan(&total); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// resolveCustomerSort maps a caller's sort string to a fixed column via
// customerSortColumns; anything absent from the map is refused, never
// interpolated.
func resolveCustomerSort(sort string) (string, bool, error) {
	if sort == "" {
		sort = defaultCustomerSort
	}
	desc := strings.HasPrefix(sort, "-")
	key := strings.TrimPrefix(sort, "-")

	col, ok := customerSortColumns[key]
	if !ok {
		return "", false, ErrInvalidSort
	}
	return col, desc, nil
}

// searchParam is nil for an empty query, which customerFilterClause reads as no filter.
func searchParam(q string) *string {
	if q == "" {
		return nil
	}
	return &q
}

func (r *PostgresRepository) LifetimeValueMinor(ctx context.Context, id int64) (int64, string, error) {
	rows, err := r.q.Query(ctx, customerLifetimeValueQuery, id)
	if err != nil {
		return 0, "", err
	}
	defer rows.Close()

	var total int64
	var currency string
	seen := 0
	for rows.Next() {
		if err := rows.Scan(&currency, &total); err != nil {
			return 0, "", err
		}
		seen++
	}
	if err := rows.Err(); err != nil {
		return 0, "", err
	}
	if seen > 1 {
		return 0, "", ErrMixedCurrencyHistory
	}
	return total, currency, nil
}

func (r *PostgresRepository) OrderHistory(ctx context.Context, customerID int64, q OrderHistoryQuery) ([]OrderSummary, int, error) {
	rows, err := r.q.Query(ctx, customerOrdersQuery, customerID, q.PageSize, offset(q.Page, q.PageSize))
	if err != nil {
		return nil, 0, err
	}
	items, err := scanOrderSummaries(rows)
	if err != nil {
		return nil, 0, err
	}

	var total int
	if err := r.q.QueryRow(ctx, customerOrdersCountQuery, customerID).Scan(&total); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func scanOrderSummaries(rows pgx.Rows) ([]OrderSummary, error) {
	defer rows.Close()

	items := []OrderSummary{}
	for rows.Next() {
		var o OrderSummary
		if err := rows.Scan(&o.ID, &o.Number, &o.Status, &o.TotalMinor, &o.Currency, &o.PlacedAt); err != nil {
			return nil, err
		}
		items = append(items, o)
	}
	return items, rows.Err()
}

func offset(page, pageSize int) int {
	return (page - 1) * pageSize
}

// row is what pgx.Row and pgxpool.Row both satisfy.
type row interface {
	Scan(dest ...any) error
}

func scanCustomer(r row, c *Customer) error {
	return r.Scan(&c.ID, &c.Email, &c.Name, &c.CreatedAt, &c.UpdatedAt)
}

func scanCustomers(rows pgx.Rows) ([]Customer, error) {
	defer rows.Close()

	items := []Customer{}
	for rows.Next() {
		var c Customer
		if err := rows.Scan(&c.ID, &c.Email, &c.Name, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	return items, rows.Err()
}

func isNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode
}
