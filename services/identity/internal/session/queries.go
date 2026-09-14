package session

// authSelectColumns' column order (id, email, password_hash, is_active,
// must_change_password, permissions) must match postgres.go's Scan calls.
const authSelectColumns = `SELECT s.id, s.email, s.password_hash, s.is_active, s.must_change_password,
       COALESCE(array_agg(p.name) FILTER (WHERE p.name IS NOT NULL), '{}')
`

// LEFT JOIN keeps a staff row whose role holds zero permissions; FILTER+COALESCE turn array_agg's NULL into '{}', not a JSON null.
const staffPermissionsJoin = `JOIN roles r ON r.id = s.role_id
LEFT JOIN role_permissions rp ON rp.role_id = r.id
LEFT JOIN permissions p ON p.id = rp.permission_id
`

// staffGroupBy collapses the one-to-many join against
// role_permissions/permissions into one row per staff member.
const staffGroupBy = `GROUP BY s.id`

const authByEmailQuery = authSelectColumns + `FROM staff s
` + staffPermissionsJoin + `WHERE s.email = $1
` + staffGroupBy

const authByIDQuery = authSelectColumns + `FROM staff s
` + staffPermissionsJoin + `WHERE s.id = $1
` + staffGroupBy

// The WHERE clause excludes a revoked/expired session or deactivated staff
// member here, not just in Service.Validate, so a caller cannot bypass it.
const authByTokenHashQuery = authSelectColumns + `FROM sessions sess
JOIN staff s ON s.id = sess.staff_id
` + staffPermissionsJoin + `WHERE sess.token_hash = $1 AND sess.revoked_at IS NULL AND sess.expires_at > now() AND s.is_active
` + staffGroupBy

const createSessionStmt = `INSERT INTO sessions (token_hash, staff_id, expires_at) VALUES ($1, $2, $3)`

// revokeSessionStmt affecting zero rows (already revoked, or never existed)
// is not an error: Service.Revoke is meant to be idempotent.
const revokeSessionStmt = `UPDATE sessions SET revoked_at = now() WHERE token_hash = $1 AND revoked_at IS NULL`
