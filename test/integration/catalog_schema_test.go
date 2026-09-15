//go:build integration

package integration_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
)

func catalogConn(t *testing.T) *pgx.Conn {
	t.Helper()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn(t, "catalog_user", "dev_only_catalog", "catalog_db"))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { conn.Close(context.Background()) })
	return conn
}

func insertProduct(t *testing.T, conn *pgx.Conn, sku, title, description string) int64 {
	t.Helper()
	var id int64
	err := conn.QueryRow(context.Background(), `
		INSERT INTO products (sku, title, description, price_minor, currency, status)
		VALUES ($1, $2, $3, 1000, 'GBP', 'active') RETURNING id`, sku, title, description).Scan(&id)
	if err != nil {
		t.Fatalf("insert %s: %v", sku, err)
	}
	t.Cleanup(func() {
		_, _ = conn.Exec(context.Background(), `DELETE FROM products WHERE id = $1`, id)
	})
	return id
}

// The generated column weights title 'A' and description 'B'. Search ranking is
// built on that ordering; if it does not hold, neither does the feature.
func TestSearchRanksATitleMatchAboveADescriptionMatch(t *testing.T) {
	conn := catalogConn(t)
	insertProduct(t, conn, "rank-desc", "Walnut desk", "a sturdy oak surface")
	insertProduct(t, conn, "rank-title", "Oak stool", "walnut legs")

	rows, err := conn.Query(context.Background(), `
		SELECT sku FROM products
		WHERE search @@ websearch_to_tsquery('english', $1)
		  AND sku LIKE 'rank-%'
		ORDER BY ts_rank(search, websearch_to_tsquery('english', $1)) DESC`, "oak")
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	defer rows.Close()

	var order []string
	for rows.Next() {
		var sku string
		if err := rows.Scan(&sku); err != nil {
			t.Fatalf("scan: %v", err)
		}
		order = append(order, sku)
	}
	if len(order) != 2 {
		t.Fatalf("want both products matched, got %v", order)
	}
	if order[0] != "rank-title" {
		t.Fatalf("title match must rank first, got %v", order)
	}
}

// price_minor is BIGINT: a value beyond float64's exact range must survive
// unchanged, which a float column would silently round.
func TestPriceSurvivesBeyondFloat64Precision(t *testing.T) {
	conn := catalogConn(t)
	const exact int64 = 9007199254740993

	var id int64
	err := conn.QueryRow(context.Background(), `
		INSERT INTO products (sku, title, price_minor, currency)
		VALUES ('precision', 'Precision', $1, 'GBP') RETURNING id`, exact).Scan(&id)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	t.Cleanup(func() { _, _ = conn.Exec(context.Background(), `DELETE FROM products WHERE id = $1`, id) })

	var got int64
	if err := conn.QueryRow(context.Background(), `SELECT price_minor FROM products WHERE id = $1`, id).Scan(&got); err != nil {
		t.Fatalf("select: %v", err)
	}
	if got != exact {
		t.Fatalf("price: want %d, got %d", exact, got)
	}
}

func TestSkuIsCaseInsensitivelyUnique(t *testing.T) {
	conn := catalogConn(t)
	insertProduct(t, conn, "Case-1", "First", "")

	_, err := conn.Exec(context.Background(), `
		INSERT INTO products (sku, title, price_minor, currency)
		VALUES ('case-1', 'Second', 1000, 'GBP')`)
	if err == nil {
		t.Fatal("a sku differing only in case was accepted")
	}
}

func TestANegativePriceIsRefused(t *testing.T) {
	conn := catalogConn(t)
	_, err := conn.Exec(context.Background(), `
		INSERT INTO products (sku, title, price_minor, currency)
		VALUES ('negative', 'Negative', -1, 'GBP')`)
	if err == nil {
		t.Fatal("a negative price was accepted")
	}
}
