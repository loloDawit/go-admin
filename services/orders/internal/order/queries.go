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

const insertOrderEventStmt = `INSERT INTO order_events (order_id, from_status, to_status, actor_id, reason)
VALUES ($1, $2, $3, $4, $5)`

const listOrderEventsQuery = `SELECT id, from_status, to_status, actor_id, reason, at
FROM order_events WHERE order_id = $1 ORDER BY at, id`

const insertOutboxStmt = `INSERT INTO outbox (event_id, subject, payload)
VALUES ($1, $2, $3)`

const getOrderByIDQuery = `SELECT ` + orderColumns + ` FROM orders WHERE id = $1`

const listOrderItemsQuery = `SELECT id, product_id, title_snapshot, unit_price_minor, currency, quantity, line_total_minor
FROM order_items WHERE order_id = $1 ORDER BY id`

// getOrderStatusForUpdateQuery locks the row so the status it reports cannot
// change under a concurrent transition before updateOrderStatusStmt writes
// the new one, both within the same transaction.
const getOrderStatusForUpdateQuery = `SELECT status FROM orders WHERE id = $1 FOR UPDATE`

// updateOrderStatusStmt is a compare-and-set: it writes only if status still
// matches $2, which is what makes the event this accompanies trustworthy.
const updateOrderStatusStmt = `UPDATE orders SET status = $3, updated_at = now()
WHERE id = $1 AND status = $2
RETURNING ` + orderColumns

// orderFilterClause: a NULL argument means no filter on that field. The
// columns are qualified because the listing query joins customers.
const orderFilterClause = `($1::order_status IS NULL OR o.status = $1) AND ($2::bigint IS NULL OR o.customer_id = $2)`

// listOrderColumns is orderColumns qualified, plus the customer's name. The
// join is inner because orders.customer_id is NOT NULL and references
// customers(id), so it cannot drop a row.
const listOrderColumns = `o.id, o.number, o.customer_id, o.status, o.total_minor, o.currency, o.placed_at, o.updated_at, c.name`

// listOrdersQueryTemplate takes the sort column and direction, both resolved
// from a fixed allowlist in postgres.go, never from caller input: a column
// name cannot be a bind parameter.
const listOrdersQueryTemplate = `SELECT ` + listOrderColumns + ` FROM orders o
JOIN customers c ON c.id = o.customer_id
WHERE ` + orderFilterClause + `
ORDER BY %s %s
LIMIT $3 OFFSET $4`

const listOrdersCountQuery = `SELECT COUNT(*) FROM orders o
JOIN customers c ON c.id = o.customer_id
WHERE ` + orderFilterClause
