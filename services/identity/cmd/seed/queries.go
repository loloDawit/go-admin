package main

// ON CONFLICT DO NOTHING is what makes a re-run safe: an existing owner keeps
// the password they have rather than silently having it reset.
const insertOwnerStmt = `
INSERT INTO staff (email, first_name, last_name, password_hash, role_id, is_active)
SELECT $1, $2, $3, $4, r.id, true FROM roles r WHERE r.name = $5
ON CONFLICT (email) DO NOTHING`

const countAdminRoleStmt = `SELECT count(*) FROM roles WHERE name = $1`
