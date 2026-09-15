package product

// productColumns is the column list every query below returns, in this
// order; postgres.go's scan calls depend on it.
const productColumns = `id, sku, title, description, price_minor, currency, status, created_at, updated_at`

const createProductStmt = `INSERT INTO products (sku, title, description, price_minor, currency)
VALUES ($1, $2, $3, $4, $5)
RETURNING ` + productColumns

const getProductByIDQuery = `SELECT ` + productColumns + ` FROM products WHERE id = $1`

// updateProductStmt's status <> 'archived' guard is what makes an edit to an
// archived product return no rows rather than silently succeeding.
const updateProductStmt = `UPDATE products SET
    title       = COALESCE($2, title),
    description = COALESCE($3, description),
    price_minor = COALESCE($4, price_minor),
    currency    = COALESCE($5, currency),
    updated_at  = now()
WHERE id = $1 AND status <> 'archived'
RETURNING ` + productColumns

// A product is created draft and cannot be ordered until it is active, so
// without this there is no route from creation to a sellable product.
const activateProductStmt = `UPDATE products SET status = 'active', updated_at = now()
WHERE id = $1 AND status <> 'active'
RETURNING ` + productColumns

const archiveProductStmt = `UPDATE products SET status = 'archived', updated_at = now()
WHERE id = $1 AND status <> 'archived'
RETURNING ` + productColumns

// statusFilterClause: a NULL $N means the default view (archived excluded);
// a non-NULL value filters to exactly that status, archived included.
// The indexed verb fills both positions from one argument: numbering them
// separately let a caller pass an argument the query never referenced, which
// Postgres rejects as an undetermined parameter type.
const statusFilterClause = `(($%[1]d::product_status IS NULL AND status <> 'archived') OR status = $%[1]d)`

// listProductsQueryTemplate takes the sort column and direction, both
// resolved from a fixed allowlist in postgres.go, never from caller input.
const listProductsQueryTemplate = `SELECT ` + productColumns + ` FROM products
WHERE ` + statusFilterClause + `
ORDER BY %[2]s %[3]s
LIMIT $2 OFFSET $3`

const listProductsCountQuery = `SELECT COUNT(*) FROM products WHERE ` + statusFilterClause

// searchProductsQueryTemplate ranks by ts_rank over the same tsquery the
// WHERE clause matched against; websearch_to_tsquery accepts quoted phrases
// and operator soup without raising.
const searchProductsQueryTemplate = `SELECT ` + productColumns + ` FROM products
WHERE search @@ websearch_to_tsquery('english', $1)
  AND ` + statusFilterClause + `
ORDER BY ts_rank(search, websearch_to_tsquery('english', $1)) DESC
LIMIT $3 OFFSET $4`

const searchProductsCountQuery = `SELECT COUNT(*) FROM products
WHERE search @@ websearch_to_tsquery('english', $1)
  AND ` + statusFilterClause

// resolveProductsQuery deliberately carries no status filter: an order must
// still resolve a product archived after the order was placed. Ordering by
// array_position matches the caller's id order in the one query, with no
// reordering in Go; a duplicate id in $1 collapses to the single matching row.
const resolveProductsQuery = `SELECT ` + productColumns + ` FROM products
WHERE id = ANY($1::bigint[])
ORDER BY array_position($1::bigint[], id)`
