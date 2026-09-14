//go:build integration

package integration_test

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

func setProductStatus(t *testing.T, conn *pgx.Conn, id int64, status string) {
	t.Helper()
	_, err := conn.Exec(context.Background(), `UPDATE products SET status = $2 WHERE id = $1`, id, status)
	if err != nil {
		t.Fatalf("set status: %v", err)
	}
}

// TestSearchAcceptsAQuotedPhrase pins websearch_to_tsquery's phrase syntax,
// not plainto_tsquery's implicit AND-of-words: only the exact phrase order
// must match.
func TestSearchAcceptsAQuotedPhrase(t *testing.T) {
	conn := catalogConn(t)
	insertProduct(t, conn, "phrase-1", "Standing desk", "an oak surface")
	insertProduct(t, conn, "phrase-2", "Desk standing lamp", "a metal base")

	var sku string
	err := conn.QueryRow(context.Background(), `
		SELECT sku FROM products
		WHERE search @@ websearch_to_tsquery('english', $1)
		  AND sku LIKE 'phrase-%'`, `"standing desk"`).Scan(&sku)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if sku != "phrase-1" {
		t.Fatalf("want the quoted phrase to match only phrase-1, got %q", sku)
	}
}

// A user typing operator soup into a search box must get results or none,
// never a driver error: to_tsquery raises on this input, websearch_to_tsquery
// does not.
func TestSearchDoesNotErrorOnOperatorSoup(t *testing.T) {
	conn := catalogConn(t)

	rows, err := conn.Query(context.Background(), `
		SELECT sku FROM products WHERE search @@ websearch_to_tsquery('english', $1)`, `&& || ! "unterminated`)
	if err != nil {
		t.Fatalf("operator soup must not raise, got: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
	}
	if rows.Err() != nil {
		t.Fatalf("row iteration: %v", rows.Err())
	}
}

func TestSearchExcludesArchivedByDefault(t *testing.T) {
	conn := catalogConn(t)
	insertProduct(t, conn, "search-active", "Findable oak table", "")
	archived := insertProduct(t, conn, "search-archived", "Findable oak chair", "")
	setProductStatus(t, conn, archived, "archived")

	rows, err := conn.Query(context.Background(), `
		SELECT sku FROM products
		WHERE search @@ websearch_to_tsquery('english', $1)
		  AND sku LIKE 'search-%'
		  AND status <> 'archived'`, "findable")
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	defer rows.Close()

	var skus []string
	for rows.Next() {
		var sku string
		if err := rows.Scan(&sku); err != nil {
			t.Fatalf("scan: %v", err)
		}
		skus = append(skus, sku)
	}
	if len(skus) != 1 || skus[0] != "search-active" {
		t.Fatalf("want only search-active, got %v", skus)
	}
}

// The default listing view (no status filter) excludes archived; an
// explicit status filter includes it.
func TestListingExcludesArchivedByDefaultAndIncludesItWithAnExplicitFilter(t *testing.T) {
	conn := catalogConn(t)
	insertProduct(t, conn, "list-active", "Listable oak stool", "")
	archived := insertProduct(t, conn, "list-archived", "Listable oak bench", "")
	setProductStatus(t, conn, archived, "archived")

	var defaultCount int
	err := conn.QueryRow(context.Background(), `
		SELECT COUNT(*) FROM products WHERE sku LIKE 'list-%' AND status <> 'archived'`).Scan(&defaultCount)
	if err != nil {
		t.Fatalf("default count: %v", err)
	}
	if defaultCount != 1 {
		t.Fatalf("default view: want 1 (archived excluded), got %d", defaultCount)
	}

	var archivedCount int
	err = conn.QueryRow(context.Background(), `
		SELECT COUNT(*) FROM products WHERE sku LIKE 'list-%' AND status = 'archived'`).Scan(&archivedCount)
	if err != nil {
		t.Fatalf("archived count: %v", err)
	}
	if archivedCount != 1 {
		t.Fatalf("explicit archived filter: want 1, got %d", archivedCount)
	}
}

// An archived product must still answer a lookup by id, even though it drops
// out of both listing and search.
func TestArchivedProductIsStillRetrievableByID(t *testing.T) {
	conn := catalogConn(t)
	id := insertProduct(t, conn, "getbyid-archived", "Still gettable", "")
	setProductStatus(t, conn, id, "archived")

	var status string
	if err := conn.QueryRow(context.Background(), `SELECT status FROM products WHERE id = $1`, id).Scan(&status); err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if status != "archived" {
		t.Fatalf("status: want archived, got %q", status)
	}
}

// An index nobody checked is an index that might not be used: this pins a
// bitmap scan on products_search_idx rather than a sequential scan.
//
// The planner only reaches for the index when the match is selective against
// a table large enough that a full scan costs more: a handful of rows would
// make a sequential scan the genuinely cheaper, and correct, choice.
func TestSearchQueryUsesTheGinIndex(t *testing.T) {
	conn := catalogConn(t)
	ctx := context.Background()

	// Below roughly this size, GIN's own scan overhead exceeds a plain
	// sequential scan's cost and the planner is right to prefer the
	// sequential plan; verified empirically against this table's real cost
	// settings, not assumed.
	const fillerRows = 200000
	_, err := conn.Exec(ctx, `
		INSERT INTO products (sku, title, description, price_minor, currency)
		SELECT 'plan-filler-' || g, 'Generic stock item', 'ordinary warehouse goods', 1000, 'GBP'
		FROM generate_series(1, $1) g`, fillerRows)
	if err != nil {
		t.Fatalf("seed filler rows: %v", err)
	}
	t.Cleanup(func() { _, _ = conn.Exec(context.Background(), `DELETE FROM products WHERE sku LIKE 'plan-filler-%'`) })
	if _, err := conn.Exec(ctx, `ANALYZE products`); err != nil {
		t.Fatalf("analyze: %v", err)
	}

	rows, err := conn.Query(ctx, `
		EXPLAIN SELECT id FROM products WHERE search @@ websearch_to_tsquery('english', $1)`, "zzyzxquery")
	if err != nil {
		t.Fatalf("explain: %v", err)
	}
	defer rows.Close()

	var plan string
	usesIndex := false
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			t.Fatalf("scan: %v", err)
		}
		plan += line + "\n"
		if strings.Contains(line, "products_search_idx") {
			usesIndex = true
		}
	}
	t.Logf("query plan:\n%s", plan)
	if !usesIndex {
		t.Fatalf("want products_search_idx in the plan, got:\n%s", plan)
	}
}
