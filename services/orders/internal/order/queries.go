package order

// nextOrderNumberQuery takes the sequence and the database clock in one
// round trip, so the two can never be read from different transactions.
const nextOrderNumberQuery = `SELECT nextval('order_number_seq'), now()`

// orderColumns is the column list every order-header query below returns, in
// this order; postgres.go's scan calls depend on it.
const orderColumns = `id, number, customer_id, status, total_minor, currency, placed_at, updated_at`

const insertOrderStmt = `INSERT INTO orders (number, customer_id, total_minor, currency, placed_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING ` + orderColumns

const insertOrderItemStmt = `INSERT INTO order_items (order_id, product_id, title_snapshot, unit_price_minor, currency, quantity, line_total_minor)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id`

const insertOrderEventStmt = `INSERT INTO order_events (order_id, from_status, to_status, actor_id)
VALUES ($1, $2, $3, $4)`

const getOrderByIDQuery = `SELECT ` + orderColumns + ` FROM orders WHERE id = $1`

const listOrderItemsQuery = `SELECT id, product_id, title_snapshot, unit_price_minor, currency, quantity, line_total_minor
FROM order_items WHERE order_id = $1 ORDER BY id`
