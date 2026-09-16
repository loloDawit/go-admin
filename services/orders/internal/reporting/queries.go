package reporting

const insertProcessedStmt = `INSERT INTO processed_events (event_id) VALUES ($1)`

const upsertPlacedStmt = `INSERT INTO revenue_by_day (day, currency, placed_count)
VALUES ($1, $2, 1)
ON CONFLICT (day, currency) DO UPDATE SET placed_count = revenue_by_day.placed_count + 1`

const upsertRecognisedStmt = `INSERT INTO revenue_by_day (day, currency, paid_count, recognised_minor)
VALUES ($1, $2, 1, $3)
ON CONFLICT (day, currency) DO UPDATE
SET paid_count = revenue_by_day.paid_count + 1,
    recognised_minor = revenue_by_day.recognised_minor + EXCLUDED.recognised_minor`

const upsertRefundedStmt = `INSERT INTO revenue_by_day (day, currency, refunded_minor)
VALUES ($1, $2, $3)
ON CONFLICT (day, currency) DO UPDATE
SET refunded_minor = revenue_by_day.refunded_minor + EXCLUDED.refunded_minor`

const insertRecognisedOrderStmt = `INSERT INTO recognised_orders (order_id, day, currency, amount_minor)
VALUES ($1, $2, $3, $4)
ON CONFLICT (order_id) DO NOTHING`

const takeRecognisedOrderStmt = `DELETE FROM recognised_orders WHERE order_id = $1
RETURNING day, currency, amount_minor`
