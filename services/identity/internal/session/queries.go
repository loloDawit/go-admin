package session

// authSelectColumns is the column list every query below scans in this exact
// order (id, email, password_hash, is_active, must_change_password,
// permissions) — postgres.go's Scan calls depend on that order.
const authSelectColumns = `SELECT s.id, s.email, s.password_hash, s.is_active, s.must_change_password,
       COALESCE(array_agg(p.name) FILTER (WHERE p.name IS NOT NULL), '{}')
`

// staffPermissionsJoin joins a staff row to its role's current permissions.
// The LEFT JOINs (not JOIN) and the FILTER/COALESCE pair below matter
// together: LEFT JOIN keeps a staff row whose role holds zero permissions,
// FILTER drops the null row that produces from array_agg, and COALESCE turns
// array_agg's resulting NULL into '{}' — verified against this stack with a
// role holding zero permissions: pgx scans the column into a non-nil, empty
// []string, which json.Marshal renders as [], not null.
const staffPermissionsJoin = `JOIN roles r ON r.id = s.role_id
LEFT JOIN role_permissions rp ON rp.role_id = r.id
LEFT JOIN permissions p ON p.id = rp.permission_id
`

// staffGroupBy collapses the one-to-many join against
// role_permissions/permissions into one row per staff member.
const staffGroupBy = `GROUP BY s.id`

// authByEmailQuery and authByIDQuery share every clause but the WHERE, so
// they compose from the same FROM staff/join/group-by body; only their
// starting table differs from authByTokenHashQuery, which reaches staff
// through a session row rather than starting from it, so that one is kept
// as its own constant rather than forced into the same composition.
const authByEmailQuery = authSelectColumns + `FROM staff s
` + staffPermissionsJoin + `WHERE s.email = $1
` + staffGroupBy

const authByIDQuery = authSelectColumns + `FROM staff s
` + staffPermissionsJoin + `WHERE s.id = $1
` + staffGroupBy

// authByTokenHashQuery's WHERE clause excludes a revoked or expired session,
// and a deactivated staff member, at the query itself — the one place a
// future caller cannot bypass the check by skipping a Go-side recheck.
// Service.Validate still reads the scanned is_active column as a second,
// defence-in-depth check against the same condition.
const authByTokenHashQuery = authSelectColumns + `FROM sessions sess
JOIN staff s ON s.id = sess.staff_id
` + staffPermissionsJoin + `WHERE sess.token_hash = $1 AND sess.revoked_at IS NULL AND sess.expires_at > now() AND s.is_active
` + staffGroupBy

const createSessionStmt = `INSERT INTO sessions (token_hash, staff_id, expires_at) VALUES ($1, $2, $3)`

// revokeSessionStmt affecting zero rows (already revoked, or never existed)
// is not an error: Service.Revoke is meant to be idempotent.
const revokeSessionStmt = `UPDATE sessions SET revoked_at = now() WHERE token_hash = $1 AND revoked_at IS NULL`
