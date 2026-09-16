//go:build integration

package integration_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// The header is NATS's own dedup key, not this system's internal detail: these
// tests are an outside client and speak the wire protocol, which is also why
// services/orders/internal is correctly out of reach here.
const msgIDHeader = "Nats-Msg-Id"

func composeService(t *testing.T, action, service string) {
	t.Helper()
	cmd := exec.Command("docker", "compose", "-f", "../../deploy/compose/docker-compose.yml", action, service)
	cmd.Env = append(cmd.Environ(), "COMPOSE_PROJECT_NAME="+composeProject(t))
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("docker compose %s %s: %v\n%s", action, service, err, out)
	}
}

// The project name is per checkout, so the tests must target the same one the
// Makefile booted rather than compose's directory default.
func composeProject(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("sh", "-c",
		`cd ../.. && printf '%s-%s' "$(basename "$PWD")" "$(printf '%s' "$PWD" | shasum | cut -c1-6)"`).Output()
	if err != nil {
		t.Fatalf("compose project name: %v", err)
	}
	return string(out)
}

func waitFor(t *testing.T, limit time.Duration, what string, done func() bool) {
	t.Helper()
	deadline := time.Now().Add(limit)
	for time.Now().Before(deadline) {
		if done() {
			return
		}
		time.Sleep(250 * time.Millisecond)
	}
	t.Fatalf("%s did not happen within %s", what, limit)
}

// placeOrder drives the real path: an active product, a customer, and an order
// through the gateway. Inserting rows directly would skip the transaction whose
// behaviour these tests are about.
func placeOrder(t *testing.T) int64 {
	t.Helper()
	c := loggedInClient(t)
	stamp := time.Now().UnixNano()

	created := createProduct(t, c, uniqueSKU(t, "obx"), "Outbox product", "for the outbox tests", 4000)
	if status, env := apiCall(t, c, http.MethodPost, "/api/v1/products/"+created.ID+"/activate", nil, nil); status != http.StatusOK {
		t.Fatalf("activate: want 200, got %d (%s)", status, env.Code)
	}

	var customer struct {
		ID string `json:"id"`
	}
	if status, env := apiCall(t, c, http.MethodPost, "/api/v1/customers", map[string]string{
		"email": fmt.Sprintf("outbox-%d@example.com", stamp),
		"name":  fmt.Sprintf("Outbox %d", stamp),
	}, &customer); status != http.StatusCreated {
		t.Fatalf("create customer: want 201, got %d (%s)", status, env.Code)
	}

	var order struct {
		ID string `json:"id"`
	}
	if status, env := apiCall(t, c, http.MethodPost, "/api/v1/orders", map[string]any{
		"customerId": customer.ID,
		"items":      []map[string]any{{"productId": created.ID, "quantity": 1}},
	}, &order); status != http.StatusCreated {
		t.Fatalf("create order: want 201, got %d (%s)", status, env.Code)
	}

	id, err := strconv.ParseInt(order.ID, 10, 64)
	if err != nil {
		t.Fatalf("order id %q: %v", order.ID, err)
	}
	return id
}

func advanceOrder(t *testing.T, id int64, status string) {
	t.Helper()
	c := loggedInClient(t)
	if got, env := apiCall(t, c, http.MethodPost,
		fmt.Sprintf("/api/v1/orders/%d/status", id), map[string]string{"status": status}, nil); got != http.StatusOK {
		t.Fatalf("set status %s: want 200, got %d (%s)", status, got, env.Code)
	}
}

func refundOrder(t *testing.T, id int64) {
	t.Helper()
	c := loggedInClient(t)
	if got, env := apiCall(t, c, http.MethodPost,
		fmt.Sprintf("/api/v1/orders/%d/refund", id), map[string]string{"reason": "integration test"}, nil); got != http.StatusOK {
		t.Fatalf("refund: want 200, got %d (%s)", got, env.Code)
	}
}

// outboxPublishedAt returns nil while the row is still waiting, which is what
// the crash and outage tests assert on.
func outboxPublishedAt(t *testing.T, conn *pgx.Conn, orderID int64) *time.Time {
	t.Helper()
	var at *time.Time
	err := conn.QueryRow(context.Background(), `
		SELECT published_at FROM outbox
		WHERE payload->>'aggregateId' = $1 AND payload->>'type' = 'order.created'`,
		strconv.FormatInt(orderID, 10)).Scan(&at)
	if errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("no outbox row for order %d — it was never written", orderID)
	}
	if err != nil {
		t.Fatalf("outbox lookup: %v", err)
	}
	return at
}

// The polling helpers take a connection rather than opening one: waitFor polls
// four times a second, and a fresh connection per poll exhausts Postgres's slots.
func scalar(t *testing.T, conn *pgx.Conn, query string, args ...any) int64 {
	t.Helper()
	var v int64
	if err := conn.QueryRow(context.Background(), query, args...).Scan(&v); err != nil {
		t.Fatalf("query: %v", err)
	}
	return v
}

func recognisedToday(t *testing.T, conn *pgx.Conn) int64 {
	return scalar(t, conn, `SELECT COALESCE(SUM(recognised_minor), 0) FROM revenue_by_day WHERE day = CURRENT_DATE`)
}

func netToday(t *testing.T, conn *pgx.Conn) int64 {
	return scalar(t, conn, `SELECT COALESCE(SUM(recognised_minor - refunded_minor), 0) FROM revenue_by_day WHERE day = CURRENT_DATE`)
}

func processedFor(t *testing.T, conn *pgx.Conn, orderID int64) int64 {
	return scalar(t, conn, `
		SELECT COUNT(*) FROM processed_events p
		JOIN outbox o ON o.event_id = p.event_id
		WHERE o.payload->>'aggregateId' = $1`, strconv.FormatInt(orderID, 10))
}

// republish sends the stored envelope again under a FRESH Nats-Msg-Id, so the
// stream accepts it and the consumer sees the same envelope id twice. That is a
// redelivery after the duplicate window has expired, and processed_events is
// then the only thing standing between it and a doubled figure. Reusing the
// original header would have the stream discard the message and prove nothing
// about the consumer at all.
func republish(t *testing.T, conn *pgx.Conn, orderID int64) {
	t.Helper()
	var eventID, subject string
	var payload []byte
	err := conn.QueryRow(context.Background(), `
		SELECT event_id, subject, payload FROM outbox
		WHERE payload->>'aggregateId' = $1 ORDER BY id DESC LIMIT 1`,
		strconv.FormatInt(orderID, 10)).Scan(&eventID, &subject, &payload)
	if err != nil {
		t.Fatalf("read outbox: %v", err)
	}

	natsConn, err := nats.Connect(hostURL("NATS_URL", "nats://127.0.0.1:4222"))
	if err != nil {
		t.Fatalf("connect nats: %v", err)
	}
	defer natsConn.Close()

	js, err := jetstream.New(natsConn)
	if err != nil {
		t.Fatalf("jetstream: %v", err)
	}

	if _, err := js.PublishMsg(context.Background(), &nats.Msg{
		Subject: subject,
		Data:    payload,
		Header:  nats.Header{msgIDHeader: []string{eventID + "-redelivered"}},
	}); err != nil {
		t.Fatalf("republish: %v", err)
	}
}

// The row is committed with the order, so no window exists in which the order
// exists and the event does not. Stopping the worker first is what proves it:
// nothing can have published before the crash being simulated.
func TestCrashAfterCommitDoesNotLoseTheEvent(t *testing.T) {
	conn := ordersConn(t)
	composeService(t, "stop", "orders-worker")
	t.Cleanup(func() { composeService(t, "start", "orders-worker") })

	orderID := placeOrder(t)
	if published := outboxPublishedAt(t, conn, orderID); published != nil {
		t.Fatal("an event was published while the worker was stopped")
	}

	composeService(t, "start", "orders-worker")
	waitFor(t, 30*time.Second, "the event to publish", func() bool {
		return outboxPublishedAt(t, conn, orderID) != nil
	})
	waitFor(t, 30*time.Second, "the projection to record it", func() bool {
		return processedFor(t, conn, orderID) > 0
	})
}

// Revenue is not naturally idempotent: applying one event twice doubles it.
func TestDuplicateEventDoesNotDuplicateTheSideEffect(t *testing.T) {
	conn := ordersConn(t)
	before := recognisedToday(t, conn)

	orderID := placeOrder(t)
	advanceOrder(t, orderID, "paid")
	// Wait on the revenue itself, not on processed_events: a break that removes
	// the dedup row would otherwise stall this wait instead of proving the
	// double-apply it exists to catch.
	waitFor(t, 30*time.Second, "revenue to be recognised", func() bool {
		return recognisedToday(t, conn) > before
	})

	recognised := recognisedToday(t, conn)
	republish(t, conn, orderID)
	time.Sleep(5 * time.Second)

	if after := recognisedToday(t, conn); after != recognised {
		t.Fatalf("recognised revenue moved from %d to %d on a redelivery", recognised, after)
	}
}

// A broker outage must not reach the request path: the order commits, the event
// waits, and the backlog drains when the broker returns.
func TestBrokerOutageDoesNotRollBackACommittedOrder(t *testing.T) {
	conn := ordersConn(t)
	composeService(t, "stop", "nats")
	t.Cleanup(func() { composeService(t, "start", "nats") })

	orderID := placeOrder(t)
	if published := outboxPublishedAt(t, conn, orderID); published != nil {
		t.Fatal("an event was published while the broker was down")
	}

	composeService(t, "start", "nats")
	waitFor(t, 60*time.Second, "the backlog to drain", func() bool {
		return outboxPublishedAt(t, conn, orderID) != nil
	})
}

func TestPublisherResumesAfterRestart(t *testing.T) {
	conn := ordersConn(t)
	composeService(t, "stop", "orders-worker")
	t.Cleanup(func() { composeService(t, "start", "orders-worker") })

	const n = 3
	ids := make([]int64, 0, n)
	for range n {
		ids = append(ids, placeOrder(t))
	}

	composeService(t, "start", "orders-worker")
	for _, id := range ids {
		waitFor(t, 60*time.Second, fmt.Sprintf("order %d to publish", id), func() bool {
			return outboxPublishedAt(t, conn, id) != nil
		})
	}

	for _, id := range ids {
		if got := processedFor(t, conn, id); got != 1 {
			t.Fatalf("order %d produced %d processed events, want exactly 1", id, got)
		}
	}
}

// paid must not be applied after refunded. With one publisher and
// MaxAckPending 1 it cannot be, and this is what would notice if either
// precondition were removed.
func TestRefundedOrderDoesNotLeaveRevenueRecognised(t *testing.T) {
	conn := ordersConn(t)
	before := netToday(t, conn)

	orderID := placeOrder(t)
	advanceOrder(t, orderID, "paid")
	waitFor(t, 30*time.Second, "revenue to be recognised", func() bool {
		return netToday(t, conn) > before
	})

	refundOrder(t, orderID)
	waitFor(t, 30*time.Second, "revenue to be reversed", func() bool {
		return netToday(t, conn) == before
	})
}
