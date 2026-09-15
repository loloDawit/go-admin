//go:build integration

package integration_test

import (
	"context"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

func ordersConn(t *testing.T) *pgx.Conn {
	t.Helper()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn(t, "orders_user", "dev_only_orders", "orders_db"))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { conn.Close(context.Background()) })
	return conn
}

func insertCustomer(t *testing.T, conn *pgx.Conn, email, name string) int64 {
	t.Helper()
	var id int64
	err := conn.QueryRow(context.Background(), `
		INSERT INTO customers (email, name) VALUES ($1, $2) RETURNING id`, email, name).Scan(&id)
	if err != nil {
		t.Fatalf("insert customer %s: %v", email, err)
	}
	t.Cleanup(func() {
		_, _ = conn.Exec(context.Background(), `DELETE FROM customers WHERE id = $1`, id)
	})
	return id
}

func insertOrder(t *testing.T, conn *pgx.Conn, customerID int64, number string) int64 {
	t.Helper()
	var id int64
	err := conn.QueryRow(context.Background(), `
		INSERT INTO orders (number, customer_id, total_minor, currency)
		VALUES ($1, $2, 1000, 'GBP') RETURNING id`, number, customerID).Scan(&id)
	if err != nil {
		t.Fatalf("insert order %s: %v", number, err)
	}
	t.Cleanup(func() {
		_, _ = conn.Exec(context.Background(), `DELETE FROM orders WHERE id = $1`, id)
	})
	return id
}

var orderNumberPattern = regexp.MustCompile(`^ORD-\d{4}-(\d{6})$`)

// order_number_seq has no annual reset, so two consecutive nextval calls
// must be adjacent integers, and the zero-padded text must sort increasing.
func TestOrderNumbersAreSequentialAndReadable(t *testing.T) {
	conn := ordersConn(t)

	var first, second string
	format := `SELECT 'ORD-' || to_char(now(), 'YYYY') || '-' || lpad(nextval('order_number_seq')::text, 6, '0')`
	if err := conn.QueryRow(context.Background(), format).Scan(&first); err != nil {
		t.Fatalf("first nextval: %v", err)
	}
	if err := conn.QueryRow(context.Background(), format).Scan(&second); err != nil {
		t.Fatalf("second nextval: %v", err)
	}

	firstMatch := orderNumberPattern.FindStringSubmatch(first)
	secondMatch := orderNumberPattern.FindStringSubmatch(second)
	if firstMatch == nil || secondMatch == nil {
		t.Fatalf("want ORD-<year>-<6 digits>, got %q then %q", first, second)
	}

	firstSeq, err := strconv.Atoi(firstMatch[1])
	if err != nil {
		t.Fatalf("parse first sequence: %v", err)
	}
	secondSeq, err := strconv.Atoi(secondMatch[1])
	if err != nil {
		t.Fatalf("parse second sequence: %v", err)
	}
	if secondSeq != firstSeq+1 {
		t.Fatalf("want consecutive nextval, got %d then %d", firstSeq, secondSeq)
	}
	if !(strings.Compare(second, first) > 0) {
		t.Fatalf("order numbers must sort increasing as text: %s then %s", first, second)
	}
}

// ON DELETE CASCADE must remove items and events with their order.
func TestDeletingAnOrderCascadesToItemsAndEvents(t *testing.T) {
	conn := ordersConn(t)
	customerID := insertCustomer(t, conn, "cascade@example.com", "Cascade Customer")
	orderID := insertOrder(t, conn, customerID, "ORD-CASCADE-TEST")

	_, err := conn.Exec(context.Background(), `
		INSERT INTO order_items (order_id, product_id, title_snapshot, unit_price_minor, currency, quantity, line_total_minor)
		VALUES ($1, 42, 'Snapshot Title', 500, 'GBP', 2, 1000)`, orderID)
	if err != nil {
		t.Fatalf("insert item: %v", err)
	}
	_, err = conn.Exec(context.Background(), `
		INSERT INTO order_events (order_id, to_status, actor_id)
		VALUES ($1, 'pending', 'system')`, orderID)
	if err != nil {
		t.Fatalf("insert event: %v", err)
	}

	if _, err := conn.Exec(context.Background(), `DELETE FROM orders WHERE id = $1`, orderID); err != nil {
		t.Fatalf("delete order: %v", err)
	}

	var itemCount, eventCount int
	if err := conn.QueryRow(context.Background(), `SELECT count(*) FROM order_items WHERE order_id = $1`, orderID).Scan(&itemCount); err != nil {
		t.Fatalf("count items: %v", err)
	}
	if err := conn.QueryRow(context.Background(), `SELECT count(*) FROM order_events WHERE order_id = $1`, orderID).Scan(&eventCount); err != nil {
		t.Fatalf("count events: %v", err)
	}
	if itemCount != 0 || eventCount != 0 {
		t.Fatalf("cascade left residue: %d items, %d events", itemCount, eventCount)
	}
}

func TestCustomerEmailIsCaseInsensitivelyUnique(t *testing.T) {
	conn := ordersConn(t)
	insertCustomer(t, conn, "Case-Dup@example.com", "First")

	_, err := conn.Exec(context.Background(), `
		INSERT INTO customers (email, name) VALUES ('case-dup@example.com', 'Second')`)
	if err == nil {
		t.Fatal("an email differing only in case was accepted")
	}
}

func TestOrderNumberIsUnique(t *testing.T) {
	conn := ordersConn(t)
	customerID := insertCustomer(t, conn, "dup-number@example.com", "Dup Number")
	insertOrder(t, conn, customerID, "ORD-DUP-0001")

	_, err := conn.Exec(context.Background(), `
		INSERT INTO orders (number, customer_id, total_minor, currency)
		VALUES ('ORD-DUP-0001', $1, 1000, 'GBP')`, customerID)
	if err == nil {
		t.Fatal("a duplicate order number was accepted")
	}
}

func TestNegativeOrderTotalIsRefused(t *testing.T) {
	conn := ordersConn(t)
	customerID := insertCustomer(t, conn, "negative-total@example.com", "Negative Total")

	_, err := conn.Exec(context.Background(), `
		INSERT INTO orders (number, customer_id, total_minor, currency)
		VALUES ('ORD-NEGATIVE-0001', $1, -1, 'GBP')`, customerID)
	if err == nil {
		t.Fatal("a negative total_minor was accepted")
	}
}

// order_items.product_id carries no foreign key by design: it names a row in
// Catalog's own database. An out-of-range id here must still be accepted.
func TestOrderItemProductIDHasNoForeignKey(t *testing.T) {
	conn := ordersConn(t)
	customerID := insertCustomer(t, conn, "no-fk@example.com", "No FK")
	orderID := insertOrder(t, conn, customerID, "ORD-NOFK-0001")

	_, err := conn.Exec(context.Background(), `
		INSERT INTO order_items (order_id, product_id, title_snapshot, unit_price_minor, currency, quantity, line_total_minor)
		VALUES ($1, 999999999, 'Nonexistent Product', 100, 'GBP', 1, 100)`, orderID)
	if err != nil {
		t.Fatalf("a product_id with no matching row elsewhere must still be accepted: %v", err)
	}
}
