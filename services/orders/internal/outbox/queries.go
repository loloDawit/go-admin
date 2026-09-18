package outbox

// Ordered by id because that is the order the rows were committed in, and the
// projection's correctness depends on paid reaching it before refunded.
const unpublishedQuery = `SELECT id, event_id, subject, payload, created_at
FROM outbox WHERE published_at IS NULL ORDER BY id LIMIT $1`

const markPublishedStmt = `UPDATE outbox SET published_at = now() WHERE id = $1`

// Read from the table rather than counted in memory: an in-memory counter is
// zero after a restart, which is exactly when it is consulted.
const unpublishedDepthQuery = `SELECT count(*) FROM outbox WHERE published_at IS NULL`
