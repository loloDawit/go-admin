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

// customerLifetimeValueQuery counts money actually taken: pending is not yet
// value, and cancelled or refunded money went back. Grouping by currency makes
// a customer with orders in two of them two rows, which the caller refuses
// rather than summing into a number that means nothing.
const customerLifetimeValueQuery = `SELECT currency, COALESCE(SUM(total_minor), 0) FROM orders
WHERE customer_id = $1 AND status IN ('paid', 'packed', 'shipped', 'delivered')
GROUP BY currency`
