package image

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

func (r *PostgresRepository) Create(ctx context.Context, img Image) (Image, error) {
	var out Image
	row := r.q.QueryRow(ctx, insertImageStmt, img.ProductID, img.ObjectKey, img.Alt)
	err := scanImage(row, &out)
	if isUniqueViolation(err) {
		return Image{}, errors.New("object key already in use")
	}
	if err != nil {
		return Image{}, err
	}
	return out, nil
}

func (r *PostgresRepository) Get(ctx context.Context, productID, imageID int64) (Image, error) {
	var out Image
	row := r.q.QueryRow(ctx, getImageQuery, productID, imageID)
	err := scanImage(row, &out)
	if isNoRows(err) {
		return Image{}, ErrImageNotFound
	}
	if err != nil {
		return Image{}, err
	}
	return out, nil
}

func (r *PostgresRepository) Delete(ctx context.Context, productID, imageID int64) error {
	tag, err := r.q.Exec(ctx, deleteImageStmt, productID, imageID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrImageNotFound
	}
	return nil
}

func (r *PostgresRepository) ListByProduct(ctx context.Context, productID int64) ([]Image, error) {
	rows, err := r.q.Query(ctx, listImagesByProductQuery, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []Image{}
	for rows.Next() {
		var img Image
		if err := rows.Scan(&img.ID, &img.ProductID, &img.ObjectKey, &img.Alt, &img.Position, &img.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, img)
	}
	return items, rows.Err()
}

// row is what pgx.Row and pgxpool.Row both satisfy.
type row interface {
	Scan(dest ...any) error
}

func scanImage(r row, img *Image) error {
	return r.Scan(&img.ID, &img.ProductID, &img.ObjectKey, &img.Alt, &img.Position, &img.CreatedAt)
}

func isNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode
}
