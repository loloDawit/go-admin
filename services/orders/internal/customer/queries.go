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

const listCustomersQuery = `SELECT ` + customerColumns + ` FROM customers
ORDER BY created_at DESC
LIMIT $1 OFFSET $2`

const countCustomersQuery = `SELECT COUNT(*) FROM customers`

// customerLifetimeValueQuery excludes cancelled and refunded orders: money
// returned to the customer is not part of their lifetime value.
const customerLifetimeValueQuery = `SELECT COALESCE(SUM(total_minor), 0) FROM orders
WHERE customer_id = $1 AND status NOT IN ('cancelled', 'refunded')`
