package customer

// customerColumns is the column list every query below returns, in this
// order; postgres.go's scan calls depend on it.
const customerColumns = `id, email, name, created_at, updated_at`

const createCustomerStmt = `INSERT INTO customers (email, name)
VALUES ($1, $2)
RETURNING ` + customerColumns

const getCustomerByIDQuery = `SELECT ` + customerColumns + ` FROM customers WHERE id = $1`

// getCustomerByEmailQuery relies on email being citext: the comparison folds
// case without a lower() index or a lower() call here.
const getCustomerByEmailQuery = `SELECT ` + customerColumns + ` FROM customers WHERE email = $1`

// customerFilterClause: a NULL $1 means no search filter.
const customerFilterClause = `($1::text IS NULL OR name ILIKE $1 ESCAPE '\' OR email ILIKE $1 ESCAPE '\')`

// listCustomersQueryPrefix and listCustomersQuerySuffix bracket the sort
// column and direction, both resolved from a fixed allowlist in postgres.go,
// never from caller input: a column name cannot be a bind parameter. They are
// concatenated rather than passed through fmt.Sprintf so the '%' wildcards in
// customerFilterClause never collide with a format verb.
const listCustomersQueryPrefix = `SELECT ` + customerColumns + ` FROM customers
WHERE ` + customerFilterClause + `
ORDER BY `

const listCustomersQuerySuffix = `
LIMIT $2 OFFSET $3`

const countCustomersQuery = `SELECT COUNT(*) FROM customers WHERE ` + customerFilterClause

// customerLifetimeValueQuery counts money actually taken: pending is not yet
// value, and cancelled or refunded money went back. Grouping by currency makes
// a customer with orders in two of them two rows, which the caller refuses
// rather than summing into a number that means nothing.
const customerLifetimeValueQuery = `SELECT currency, COALESCE(SUM(total_minor), 0) FROM orders
WHERE customer_id = $1 AND status IN ('paid', 'packed', 'shipped', 'delivered')
GROUP BY currency`

const orderSummaryColumns = `id, number, status, total_minor, currency, placed_at`

// customerOrdersQuery is scoped by customer_id = $1: every row of every
// status this customer has ever placed, not just the revenue-counted ones
// lifetime value uses.
const customerOrdersQuery = `SELECT ` + orderSummaryColumns + ` FROM orders
WHERE customer_id = $1
ORDER BY placed_at DESC
LIMIT $2 OFFSET $3`

const customerOrdersCountQuery = `SELECT COUNT(*) FROM orders WHERE customer_id = $1`
