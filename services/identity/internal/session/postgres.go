package session

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) AuthByEmail(ctx context.Context, email string) (StaffAuth, error) {
	var auth StaffAuth
	row := r.pool.QueryRow(ctx, authByEmailQuery, email)
	err := row.Scan(&auth.ID, &auth.Email, &auth.PasswordHash, &auth.IsActive, &auth.MustChangePassword, &auth.Permissions)
	if isNoRows(err) {
		return StaffAuth{}, ErrNotFound
	}
	if err != nil {
		return StaffAuth{}, err
	}
	return auth, nil
}

func (r *PostgresRepository) AuthByID(ctx context.Context, id int64) (StaffAuth, error) {
	var auth StaffAuth
	row := r.pool.QueryRow(ctx, authByIDQuery, id)
	err := row.Scan(&auth.ID, &auth.Email, &auth.PasswordHash, &auth.IsActive, &auth.MustChangePassword, &auth.Permissions)
	if isNoRows(err) {
		return StaffAuth{}, ErrNotFound
	}
	if err != nil {
		return StaffAuth{}, err
	}
	return auth, nil
}

func (r *PostgresRepository) AuthByTokenHash(ctx context.Context, tokenHash []byte) (StaffAuth, error) {
	var auth StaffAuth
	row := r.pool.QueryRow(ctx, authByTokenHashQuery, tokenHash)
	err := row.Scan(&auth.ID, &auth.Email, &auth.PasswordHash, &auth.IsActive, &auth.MustChangePassword, &auth.Permissions)
	if isNoRows(err) {
		return StaffAuth{}, ErrNotFound
	}
	if err != nil {
		return StaffAuth{}, err
	}
	return auth, nil
}

func (r *PostgresRepository) CreateSession(ctx context.Context, staffID int64, tokenHash []byte, expiresAt time.Time) error {
	_, err := r.pool.Exec(ctx, createSessionStmt, tokenHash, staffID, expiresAt)
	return err
}

func (r *PostgresRepository) RevokeSession(ctx context.Context, tokenHash []byte) error {
	_, err := r.pool.Exec(ctx, revokeSessionStmt, tokenHash)
	return err
}

func isNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
