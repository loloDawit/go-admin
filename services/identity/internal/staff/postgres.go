package staff

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/loloDawit/go-admin/services/identity/internal/permission"
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
	pool *pgxpool.Pool
	q    querier
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool, q: pool}
}

// RunInTx gives fn a repository bound to one transaction, so a guard's read
// and the write it protects cannot be interleaved with another caller's.
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

// Postgres refuses FOR UPDATE alongside an aggregate, so the rows are locked and counted here instead of in SQL.
func (r *PostgresRepository) LockActiveEditStaffExcluding(ctx context.Context, excludeID int64) (int, error) {
	rows, err := r.q.Query(ctx, lockActiveEditStaffQuery, string(permission.EditStaff))
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	others := 0
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return 0, err
		}
		if id != excludeID {
			others++
		}
	}
	return others, rows.Err()
}

func (r *PostgresRepository) RevokeSessions(ctx context.Context, staffID int64) error {
	_, err := r.q.Exec(ctx, revokeSessionsForStaffStmt, staffID)
	return err
}

func (r *PostgresRepository) Create(ctx context.Context, in CreateStaff, passwordHash string) (Staff, error) {
	var st Staff
	row := r.q.QueryRow(ctx, createStaffStmt, in.Email, in.FirstName, in.LastName, passwordHash, in.RoleID)
	err := scanStaff(row, &st)
	if isUniqueViolation(err) {
		return Staff{}, ErrEmailTaken
	}
	if err != nil {
		return Staff{}, err
	}
	return st, nil
}

func (r *PostgresRepository) Update(ctx context.Context, id int64, in UpdateStaff) (Staff, error) {
	var st Staff
	row := r.q.QueryRow(ctx, updateStaffStmt, id, in.FirstName, in.LastName, in.Email, in.RoleID, in.IsActive)
	err := scanStaff(row, &st)
	if isUniqueViolation(err) {
		return Staff{}, ErrEmailTaken
	}
	if isNoRows(err) {
		return Staff{}, ErrNotFound
	}
	if err != nil {
		return Staff{}, err
	}
	return st, nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id int64) (Staff, error) {
	var st Staff
	row := r.q.QueryRow(ctx, getStaffByIDQuery, id)
	err := scanStaff(row, &st)
	if isNoRows(err) {
		return Staff{}, ErrNotFound
	}
	if err != nil {
		return Staff{}, err
	}
	return st, nil
}

func (r *PostgresRepository) List(ctx context.Context) ([]Staff, error) {
	rows, err := r.q.Query(ctx, listStaffQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := []Staff{}
	for rows.Next() {
		var st Staff
		if err := rows.Scan(&st.ID, &st.Email, &st.FirstName, &st.LastName, &st.RoleID, &st.IsActive, &st.MustChangePassword); err != nil {
			return nil, err
		}
		list = append(list, st)
	}
	return list, rows.Err()
}

func (r *PostgresRepository) PasswordHash(ctx context.Context, id int64) (string, error) {
	var hash string
	err := r.q.QueryRow(ctx, passwordHashByIDQuery, id).Scan(&hash)
	if isNoRows(err) {
		return "", ErrNotFound
	}
	return hash, err
}

func (r *PostgresRepository) SetPasswordHash(ctx context.Context, id int64, hash string) error {
	_, err := r.q.Exec(ctx, setPasswordHashStmt, id, hash)
	return err
}

func (r *PostgresRepository) HasEditStaffPermission(ctx context.Context, roleID int64) (bool, error) {
	var has bool
	err := r.q.QueryRow(ctx, hasEditStaffPermissionQuery, roleID, string(permission.EditStaff)).Scan(&has)
	return has, err
}

func (r *PostgresRepository) CountOtherActiveStaffWithEditStaff(ctx context.Context, excludeID int64) (int, error) {
	var count int
	err := r.q.QueryRow(ctx, countOtherActiveStaffWithEditStaffQuery, excludeID, string(permission.EditStaff)).Scan(&count)
	return count, err
}

// row is what pgx.Row and pgxpool.Row both satisfy; scanStaff is shared by
// Create and Update, whose statements return the same column list.
type row interface {
	Scan(dest ...any) error
}

func scanStaff(r row, st *Staff) error {
	return r.Scan(&st.ID, &st.Email, &st.FirstName, &st.LastName, &st.RoleID, &st.IsActive, &st.MustChangePassword)
}

func isNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode
}
