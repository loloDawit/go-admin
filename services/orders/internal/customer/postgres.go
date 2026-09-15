package customer

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// uniqueViolationCode is Postgres's SQLSTATE for a unique-constraint conflict.
const uniqueViolationCode = "23505"

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
	rows, err := r.q.Query(ctx, listCustomersQuery, q.PageSize, offset(q.Page, q.PageSize))
	if err != nil {
		return nil, 0, err
	}
	items, err := scanCustomers(rows)
	if err != nil {
		return nil, 0, err
	}

	var total int
	if err := r.q.QueryRow(ctx, countCustomersQuery).Scan(&total); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *PostgresRepository) LifetimeValueMinor(ctx context.Context, id int64) (int64, error) {
	var total int64
	if err := r.q.QueryRow(ctx, customerLifetimeValueQuery, id).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
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
