package product

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// uniqueViolationCode is Postgres's SQLSTATE for a unique-constraint conflict.
const uniqueViolationCode = "23505"

// sortColumns is the only place a caller's sort string reaches a column
// name: anything absent here is refused rather than interpolated.
var sortColumns = map[string]string{
	"title":      "title",
	"price":      "price_minor",
	"created_at": "created_at",
	"sku":        "sku",
}

const defaultSort = "-created_at"

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

func (r *PostgresRepository) Create(ctx context.Context, in CreateProduct) (Product, error) {
	var p Product
	row := r.q.QueryRow(ctx, createProductStmt, in.SKU, in.Title, in.Description, in.PriceMinor, in.Currency)
	err := scanProduct(row, &p)
	if isUniqueViolation(err) {
		return Product{}, ErrSkuTaken
	}
	if err != nil {
		return Product{}, err
	}
	return p, nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id int64) (Product, error) {
	var p Product
	row := r.q.QueryRow(ctx, getProductByIDQuery, id)
	err := scanProduct(row, &p)
	if isNoRows(err) {
		return Product{}, ErrProductNotFound
	}
	if err != nil {
		return Product{}, err
	}
	return p, nil
}

func (r *PostgresRepository) Update(ctx context.Context, id int64, in UpdateProduct) (Product, error) {
	var p Product
	row := r.q.QueryRow(ctx, updateProductStmt, id, in.Title, in.Description, in.PriceMinor, in.Currency)
	err := scanProduct(row, &p)
	if isNoRows(err) {
		return r.notFoundOrArchived(ctx, id)
	}
	if err != nil {
		return Product{}, err
	}
	return p, nil
}

func (r *PostgresRepository) Archive(ctx context.Context, id int64) (Product, error) {
	var p Product
	row := r.q.QueryRow(ctx, archiveProductStmt, id)
	err := scanProduct(row, &p)
	if isNoRows(err) {
		return r.notFoundOrArchived(ctx, id)
	}
	if err != nil {
		return Product{}, err
	}
	return p, nil
}

// notFoundOrArchived runs after an id-scoped UPDATE returned no rows, to
// tell a missing id apart from one that exists but is already archived.
func (r *PostgresRepository) notFoundOrArchived(ctx context.Context, id int64) (Product, error) {
	existing, err := r.GetByID(ctx, id)
	if err != nil {
		return Product{}, err
	}
	if existing.Status == StatusArchived {
		return Product{}, ErrProductArchived
	}
	// The row exists and is not archived, yet the update matched nothing: it
	// changed state between the two statements above.
	return Product{}, fmt.Errorf("product %d changed concurrently", id)
}

func (r *PostgresRepository) List(ctx context.Context, q ListQuery) ([]Product, int, error) {
	col, desc, err := resolveSort(q.Sort)
	if err != nil {
		return nil, 0, err
	}
	direction := "ASC"
	if desc {
		direction = "DESC"
	}
	status := statusParam(q.Status)

	stmt := fmt.Sprintf(listProductsQueryTemplate, 1, 1, col, direction)
	rows, err := r.q.Query(ctx, stmt, status, status, q.PageSize, offset(q.Page, q.PageSize))
	if err != nil {
		return nil, 0, err
	}
	items, err := scanProducts(rows)
	if err != nil {
		return nil, 0, err
	}

	countStmt := fmt.Sprintf(listProductsCountQuery, 1, 1)
	var total int
	if err := r.q.QueryRow(ctx, countStmt, status, status).Scan(&total); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *PostgresRepository) Search(ctx context.Context, q SearchQuery) ([]Product, int, error) {
	status := statusParam(q.Status)

	stmt := fmt.Sprintf(searchProductsQueryTemplate, 2, 2)
	rows, err := r.q.Query(ctx, stmt, q.Text, status, q.PageSize, offset(q.Page, q.PageSize))
	if err != nil {
		return nil, 0, err
	}
	items, err := scanProducts(rows)
	if err != nil {
		return nil, 0, err
	}

	countStmt := fmt.Sprintf(searchProductsCountQuery, 2, 2)
	var total int
	if err := r.q.QueryRow(ctx, countStmt, q.Text, status).Scan(&total); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *PostgresRepository) ResolveByIDs(ctx context.Context, ids []int64) ([]Product, error) {
	rows, err := r.q.Query(ctx, resolveProductsQuery, ids)
	if err != nil {
		return nil, err
	}
	return scanProducts(rows)
}

// resolveSort maps a caller's sort string to a fixed column via sortColumns;
// anything absent from the map is refused, never interpolated.
func resolveSort(sort string) (string, bool, error) {
	if sort == "" {
		sort = defaultSort
	}
	desc := strings.HasPrefix(sort, "-")
	key := strings.TrimPrefix(sort, "-")

	col, ok := sortColumns[key]
	if !ok {
		return "", false, ErrInvalidSort
	}
	return col, desc, nil
}

func statusParam(s *Status) *string {
	if s == nil {
		return nil
	}
	v := string(*s)
	return &v
}

func offset(page, pageSize int) int {
	return (page - 1) * pageSize
}

// row is what pgx.Row and pgxpool.Row both satisfy.
type row interface {
	Scan(dest ...any) error
}

func scanProduct(r row, p *Product) error {
	return r.Scan(&p.ID, &p.SKU, &p.Title, &p.Description, &p.PriceMinor, &p.Currency, &p.Status, &p.CreatedAt, &p.UpdatedAt)
}

func scanProducts(rows pgx.Rows) ([]Product, error) {
	defer rows.Close()

	items := []Product{}
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ID, &p.SKU, &p.Title, &p.Description, &p.PriceMinor, &p.Currency, &p.Status, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, p)
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
