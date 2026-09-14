package role

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/loloDawit/go-admin/services/identity/internal/permission"
)

// execer is what *pgxpool.Pool and pgx.Tx both satisfy.
type execer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// RunInTx gives fn a repository bound to one transaction. Every method on
// that repository must read through r.q, never r.pool: a method that reaches
// past q runs on its own connection, outside fn's transaction.
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
func (r *PostgresRepository) LockActiveStaffWithEditStaffOutsideRole(ctx context.Context, roleID int64) (int, error) {
	if _, err := r.q.Exec(ctx, lockAdminGuardStmt); err != nil {
		return 0, err
	}
	rows, err := r.q.Query(ctx, lockActiveStaffWithEditStaffQuery, string(permission.EditStaff))
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	others := 0
	for rows.Next() {
		var otherRoleID int64
		if err := rows.Scan(&otherRoleID); err != nil {
			return 0, err
		}
		if otherRoleID != roleID {
			others++
		}
	}
	return others, rows.Err()
}

// insertRolePermissions asserts what it wrote: a name matching no row inserts nothing and raises no SQL error on its own.
func insertRolePermissions(ctx context.Context, e execer, roleID int64, names []string) error {
	unique := uniqueStrings(names)
	if len(unique) == 0 {
		return nil
	}
	tag, err := e.Exec(ctx, insertRolePermissionsStmt, roleID, unique)
	if err != nil {
		return err
	}
	if int(tag.RowsAffected()) != len(unique) {
		return ErrPermissionRowMismatch
	}
	return nil
}

func uniqueStrings(in []string) []string {
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

const uniqueViolationCode = "23505"
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

// queryRower is what *pgxpool.Pool and pgx.Tx both satisfy, so
// getRoleWithPermissions can run inside or outside a transaction.
type queryRower interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func getRoleWithPermissions(ctx context.Context, q queryRower, id int64) (Role, error) {
	var rl Role
	err := q.QueryRow(ctx, roleWithPermissionsQuery, id).Scan(&rl.ID, &rl.Name, &rl.Permissions)
	if isNoRows(err) {
		return Role{}, ErrNotFound
	}
	if err != nil {
		return Role{}, err
	}
	return rl, nil
}

func (r *PostgresRepository) Create(ctx context.Context, in CreateRole) (Role, error) {
	tx := r.q

	var id int64
	if err := tx.QueryRow(ctx, createRoleStmt, in.Name).Scan(&id); err != nil {
		if isUniqueViolation(err) {
			return Role{}, ErrNameTaken
		}
		return Role{}, err
	}
	if err := insertRolePermissions(ctx, tx, id, in.Permissions); err != nil {
		return Role{}, err
	}

	created, err := getRoleWithPermissions(ctx, tx, id)
	if err != nil {
		return Role{}, err
	}
	return created, nil
}

func (r *PostgresRepository) Update(ctx context.Context, id int64, in UpdateRole) (Role, error) {
	tx := r.q

	var updatedID int64
	var err error
	err = tx.QueryRow(ctx, updateRoleNameStmt, id, in.Name).Scan(&updatedID)
	if isNoRows(err) {
		return Role{}, ErrNotFound
	}
	if isUniqueViolation(err) {
		return Role{}, ErrNameTaken
	}
	if err != nil {
		return Role{}, err
	}

	if in.Permissions != nil {
		if _, err := tx.Exec(ctx, deleteRolePermissionsStmt, id); err != nil {
			return Role{}, err
		}
		if err := insertRolePermissions(ctx, tx, id, *in.Permissions); err != nil {
			return Role{}, err
		}
	}

	updated, err := getRoleWithPermissions(ctx, tx, id)
	if err != nil {
		return Role{}, err
	}
	return updated, nil
}

func (r *PostgresRepository) Delete(ctx context.Context, id int64) error {
	tag, err := r.q.Exec(ctx, deleteRoleStmt, id)
	if isForeignKeyViolation(err) {
		return ErrInUse
	}
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id int64) (Role, error) {
	return getRoleWithPermissions(ctx, r.q, id)
}

func (r *PostgresRepository) List(ctx context.Context) ([]Role, error) {
	rows, err := r.q.Query(ctx, listRolesWithPermissionsQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := []Role{}
	for rows.Next() {
		var rl Role
		if err := rows.Scan(&rl.ID, &rl.Name, &rl.Permissions); err != nil {
			return nil, err
		}
		list = append(list, rl)
	}
	return list, rows.Err()
}

func (r *PostgresRepository) HasEditStaffPermission(ctx context.Context, roleID int64) (bool, error) {
	var has bool
	err := r.q.QueryRow(ctx, hasEditStaffPermissionQuery, roleID, string(permission.EditStaff)).Scan(&has)
	return has, err
}

func isNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode
}

func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == foreignKeyViolationCode
}
