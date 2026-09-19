package role

const createRoleStmt = `INSERT INTO roles (name) VALUES ($1) RETURNING id`

const updateRoleNameStmt = `UPDATE roles SET name = COALESCE($2, name), updated_at = now() WHERE id = $1 RETURNING id`

const deleteRoleStmt = `DELETE FROM roles WHERE id = $1`

const deleteRolePermissionsStmt = `DELETE FROM role_permissions WHERE role_id = $1`

// insertRolePermissionsStmt: the caller compares RowsAffected to len(names), since a name absent from permissions inserts nothing and raises no error.
const insertRolePermissionsStmt = `INSERT INTO role_permissions (role_id, permission_id)
SELECT $1, id FROM permissions WHERE name = ANY($2::text[])`

// roleWithPermissionsQuery's FILTER keeps a role with no permissions at an empty array rather than the LEFT JOIN's one all-NULL row.
const roleWithPermissionsQuery = `
SELECT r.id, r.name, COALESCE(array_agg(p.name ORDER BY p.name) FILTER (WHERE p.name IS NOT NULL), '{}')
FROM roles r
LEFT JOIN role_permissions rp ON rp.role_id = r.id
LEFT JOIN permissions p ON p.id = rp.permission_id
WHERE r.id = $1
GROUP BY r.id, r.name`

// roleFilterClause: a NULL $1 means no search filter.
const roleFilterClause = `($1::text IS NULL OR r.name ILIKE '%' || $1 || '%')`

// The member count is a scalar subquery rather than another LEFT JOIN: joining
// staff as well would multiply the permission rows before the aggregate. It
// is aliased member_count so ORDER BY can reference it directly rather than
// repeating the subquery.
//
// listRolesWithPermissionsQueryPrefix and listRolesWithPermissionsQuerySuffix
// bracket the sort column and direction, both resolved from a fixed allowlist
// in postgres.go, never from caller input. They are concatenated rather than
// passed through fmt.Sprintf so the '%' wildcard in roleFilterClause never
// collides with a format verb.
const listRolesWithPermissionsQueryPrefix = `
SELECT r.id, r.name,
       COALESCE(array_agg(p.name ORDER BY p.name) FILTER (WHERE p.name IS NOT NULL), '{}'),
       (SELECT COUNT(*) FROM staff s WHERE s.role_id = r.id) AS member_count
FROM roles r
LEFT JOIN role_permissions rp ON rp.role_id = r.id
LEFT JOIN permissions p ON p.id = rp.permission_id
WHERE ` + roleFilterClause + `
GROUP BY r.id, r.name
ORDER BY `

const listRolesWithPermissionsQuerySuffix = `
LIMIT $2 OFFSET $3`

const countRolesQuery = `SELECT COUNT(*) FROM roles r WHERE ` + roleFilterClause

const hasEditStaffPermissionQuery = `SELECT EXISTS(
    SELECT 1 FROM role_permissions rp
    JOIN permissions p ON p.id = rp.permission_id
    WHERE rp.role_id = $1 AND p.name = $2
)`

// FOR UPDATE OF s locks the candidate staff rows for the transaction, so a
// concurrent demotion cannot slip between this count and the write it guards.
const lockActiveStaffWithEditStaffQuery = `SELECT s.role_id FROM staff s
JOIN role_permissions rp ON rp.role_id = s.role_id
JOIN permissions p ON p.id = rp.permission_id
WHERE p.name = $1 AND s.is_active
FOR UPDATE OF s`

// The same advisory key the staff guard takes: a role demotion and a staff
// demotion must serialize, and they write different tables, so row locks alone
// leave each invisible to the other.
const lockAdminGuardStmt = `SELECT pg_advisory_xact_lock(4_812_001)`
