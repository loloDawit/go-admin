package staff

// staffSelectColumns is the column list every query below returns, in this order; postgres.go's Scan calls depend on it.
const staffSelectColumns = `id, email, first_name, last_name, role_id, is_active, must_change_password`

// createStaffStmt always sets must_change_password true: a newly created
// account must change its one-time password before anything else.
const createStaffStmt = `INSERT INTO staff (email, first_name, last_name, password_hash, role_id, is_active, must_change_password)
VALUES ($1, $2, $3, $4, $5, true, true)
RETURNING ` + staffSelectColumns

const getStaffByIDQuery = `SELECT ` + staffSelectColumns + ` FROM staff WHERE id = $1`

// staffFilterClause: a NULL $1 means no search filter.
const staffFilterClause = `($1::text IS NULL OR first_name ILIKE $1 ESCAPE '\' OR last_name ILIKE $1 ESCAPE '\' OR email ILIKE $1 ESCAPE '\')`

// listStaffQueryPrefix and listStaffQuerySuffix bracket the sort column and
// direction, both resolved from a fixed allowlist in postgres.go, never from
// caller input: a column name cannot be a bind parameter. They are
// concatenated rather than passed through fmt.Sprintf so the '%' wildcards in
// staffFilterClause never collide with a format verb.
const listStaffQueryPrefix = `SELECT ` + staffSelectColumns + ` FROM staff
WHERE ` + staffFilterClause + `
ORDER BY `

const listStaffQuerySuffix = `
LIMIT $2 OFFSET $3`

const countStaffQuery = `SELECT COUNT(*) FROM staff WHERE ` + staffFilterClause

// updateStaffStmt's COALESCE pair on each column is what makes the update
// partial: a nil parameter leaves that column exactly as it was.
const updateStaffStmt = `UPDATE staff SET
    first_name = COALESCE($2, first_name),
    last_name  = COALESCE($3, last_name),
    email      = COALESCE($4, email),
    role_id    = COALESCE($5, role_id),
    is_active  = COALESCE($6, is_active),
    updated_at = now()
WHERE id = $1
RETURNING ` + staffSelectColumns

const passwordHashByIDQuery = `SELECT password_hash FROM staff WHERE id = $1`

// setPasswordHashStmt clears must_change_password on every password change, forced or voluntary.
const setPasswordHashStmt = `UPDATE staff SET password_hash = $2, must_change_password = false, updated_at = now() WHERE id = $1`

// hasEditStaffPermissionQuery checks the permission a role holds, not the role's name.
const hasEditStaffPermissionQuery = `SELECT EXISTS(
    SELECT 1 FROM role_permissions rp
    JOIN permissions p ON p.id = rp.permission_id
    WHERE rp.role_id = $1 AND p.name = $2
)`

// FOR UPDATE OF s locks the candidate rows for the transaction's duration, so
// two concurrent demotions cannot each read that another admin remains.
const lockActiveEditStaffQuery = `SELECT s.id FROM staff s
JOIN role_permissions rp ON rp.role_id = s.role_id
JOIN permissions p ON p.id = rp.permission_id
WHERE p.name = $1 AND s.is_active
FOR UPDATE OF s`

const revokeSessionsForStaffStmt = `UPDATE sessions SET revoked_at = now() WHERE staff_id = $1 AND revoked_at IS NULL`

// Both admin guards take this lock before counting. Row locks are not enough:
// the staff guard writes staff rows while the role guard writes
// role_permissions, so neither blocks the other and two concurrent demotions
// can each see an admin the other is removing.
const lockAdminGuardStmt = `SELECT pg_advisory_xact_lock(4_812_001)`
