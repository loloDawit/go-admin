//go:build integration

package integration_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
)

func setOrderStatus(t *testing.T, conn *pgx.Conn, id int64, status string) {
	t.Helper()
	_, err := conn.Exec(context.Background(), `UPDATE orders SET status = $2 WHERE id = $1`, id, status)
	if err != nil {
		t.Fatalf("set order status: %v", err)
	}
}

// customerByEmail mirrors customer/queries.go's getCustomerByEmailQuery: this
// package cannot import orders' internal package, so the SQL is retyped,
// same as the other tests in this file.
func customerByEmail(t *testing.T, conn *pgx.Conn, email string) (int64, bool) {
	t.Helper()
	var id int64
	err := conn.QueryRow(context.Background(), `SELECT id FROM customers WHERE email = $1`, email).Scan(&id)
	if err == pgx.ErrNoRows {
		return 0, false
	}
	if err != nil {
		t.Fatalf("lookup by email: %v", err)
	}
	return id, true
}

// customerLifetimeValueMinor mirrors customer/queries.go's customerLifetimeValueQuery.
func customerLifetimeValueMinor(t *testing.T, conn *pgx.Conn, customerID int64) int64 {
	t.Helper()
	var total int64
	err := conn.QueryRow(context.Background(), `
		SELECT COALESCE(SUM(total_minor), 0) FROM orders
		WHERE customer_id = $1 AND status NOT IN ('cancelled', 'refunded')`, customerID).Scan(&total)
	if err != nil {
		t.Fatalf("lifetime value: %v", err)
	}
	return total
}

// The reason citext is there: a lookup must find a customer regardless of
// the case the caller typed the email in.
func TestLookupByEmailIsCaseInsensitive(t *testing.T) {
	conn := ordersConn(t)
	insertCustomer(t, conn, "Mixed-Lookup@Example.com", "Mixed Lookup")

	id, found := customerByEmail(t, conn, "mixed-lookup@example.com")
	if !found {
		t.Fatal("want a match on differing case")
	}
	if id == 0 {
		t.Fatal("want a non-zero id")
	}
}

// §15.4 requires lifetime value; cancelled and refunded orders must not
// count, and a second customer's orders must not bleed into the total.
func TestLifetimeValueSumsOnlyThatCustomersRevenueOrders(t *testing.T) {
	conn := ordersConn(t)
	customerA := insertCustomer(t, conn, "ltv-a@example.com", "Customer A")
	customerB := insertCustomer(t, conn, "ltv-b@example.com", "Customer B")

	paid := insertOrder(t, conn, customerA, "ORD-LTV-PAID")
	setOrderStatus(t, conn, paid, "paid")

	refunded := insertOrder(t, conn, customerA, "ORD-LTV-REFUNDED")
	setOrderStatus(t, conn, refunded, "refunded")

	otherPaid := insertOrder(t, conn, customerB, "ORD-LTV-OTHER")
	setOrderStatus(t, conn, otherPaid, "paid")

	got := customerLifetimeValueMinor(t, conn, customerA)
	if got != 1000 {
		t.Fatalf("want 1000 (the one paid order, refund and other customer excluded), got %d", got)
	}
}
