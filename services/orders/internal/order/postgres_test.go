package order

import "testing"

// A sort column cannot be a bind parameter, so this is the one place
// injection is structurally possible: anything absent from the allowlist
// must be refused rather than interpolated into the ORDER BY clause.
func TestSortColumnIsAnAllowlistNotAString(t *testing.T) {
	if _, _, err := resolveOrderSort("status; DROP TABLE orders"); err != ErrInvalidSort {
		t.Fatalf("want ErrInvalidSort for an unknown sort key, got %v", err)
	}

	col, desc, err := resolveOrderSort("total_minor")
	if err != nil {
		t.Fatalf("resolveOrderSort(total_minor): %v", err)
	}
	if col != "o.total_minor" || desc {
		t.Fatalf("got col=%q desc=%v, want o.total_minor ascending", col, desc)
	}
}

func TestSortColumnPrefixMeansDescending(t *testing.T) {
	col, desc, err := resolveOrderSort("-placed_at")
	if err != nil {
		t.Fatalf("resolveOrderSort(-placed_at): %v", err)
	}
	if col != "o.placed_at" || !desc {
		t.Fatalf("got col=%q desc=%v, want o.placed_at descending", col, desc)
	}
}

func TestSortColumnDefaultsWhenAbsent(t *testing.T) {
	col, desc, err := resolveOrderSort("")
	if err != nil {
		t.Fatalf("resolveOrderSort(\"\"): %v", err)
	}
	if col != "o.placed_at" || !desc {
		t.Fatalf("got col=%q desc=%v, want the default of o.placed_at descending", col, desc)
	}
}

// The listing query joins customers, so an unqualified column would become
// ambiguous the moment both tables carry one of the same name.
func TestSortColumnsAreQualifiedForTheJoin(t *testing.T) {
	col, _, err := resolveOrderSort("customer")
	if err != nil {
		t.Fatalf("resolveOrderSort(customer): %v", err)
	}
	if col != "c.name" {
		t.Fatalf("got col=%q, want c.name", col)
	}
	for key, want := range map[string]string{"status": "o.status", "number": "o.number"} {
		got, _, err := resolveOrderSort(key)
		if err != nil {
			t.Fatalf("resolveOrderSort(%s): %v", key, err)
		}
		if got != want {
			t.Fatalf("resolveOrderSort(%s) = %q, want %q", key, got, want)
		}
	}
}

func TestOffsetComputesFromPageAndPageSize(t *testing.T) {
	if got := offset(1, 20); got != 0 {
		t.Fatalf("page 1: want offset 0, got %d", got)
	}
	if got := offset(3, 20); got != 40 {
		t.Fatalf("page 3: want offset 40, got %d", got)
	}
}
