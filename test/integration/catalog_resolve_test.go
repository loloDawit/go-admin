//go:build integration

package integration_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
)

// resolveProductIDs mirrors product/queries.go's resolveProductsQuery: this
// package cannot import catalog's internal package, so the SQL is retyped,
// same as the other tests in this file.
func resolveProductIDs(t *testing.T, conn *pgx.Conn, ids []int64) []int64 {
	t.Helper()
	rows, err := conn.Query(context.Background(), `
		SELECT id FROM products
		WHERE id = ANY($1::bigint[])
		ORDER BY array_position($1::bigint[], id)`, ids)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	defer rows.Close()

	var got []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan: %v", err)
		}
		got = append(got, id)
	}
	if rows.Err() != nil {
		t.Fatalf("row iteration: %v", rows.Err())
	}
	return got
}

// An order placed before its product was archived must still resolve; the
// query itself carries no status filter, unlike List and Search.
func TestResolveByIDsIncludesArchivedProducts(t *testing.T) {
	conn := catalogConn(t)
	id := insertProduct(t, conn, "resolve-int-arch", "Archived resolvable", "")
	setProductStatus(t, conn, id, "archived")

	got := resolveProductIDs(t, conn, []int64{id})
	if len(got) != 1 || got[0] != id {
		t.Fatalf("want the archived product resolved, got %v", got)
	}
}

// A deleted or never-existing id must be silently absent, not an error.
func TestResolveByIDsOmitsUnknownIDs(t *testing.T) {
	conn := catalogConn(t)
	const unknownID = 987654321

	got := resolveProductIDs(t, conn, []int64{unknownID})
	if len(got) != 0 {
		t.Fatalf("want no items for an unknown id, got %v", got)
	}
}

// The response contract states results are ordered to match the requested id
// order; array_position must deliver that from the one query, not from a
// reorder step in Go.
func TestResolveByIDsOrdersResultsByRequestedIDOrder(t *testing.T) {
	conn := catalogConn(t)
	a := insertProduct(t, conn, "resolve-int-a", "A", "")
	b := insertProduct(t, conn, "resolve-int-b", "B", "")

	got := resolveProductIDs(t, conn, []int64{b, a})
	if len(got) != 2 || got[0] != b || got[1] != a {
		t.Fatalf("want [%d, %d] in requested order, got %v", b, a, got)
	}
}
