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
	if col != "total_minor" || desc {
		t.Fatalf("got col=%q desc=%v, want total_minor ascending", col, desc)
	}
}

func TestSortColumnPrefixMeansDescending(t *testing.T) {
	col, desc, err := resolveOrderSort("-placed_at")
	if err != nil {
		t.Fatalf("resolveOrderSort(-placed_at): %v", err)
	}
	if col != "placed_at" || !desc {
		t.Fatalf("got col=%q desc=%v, want placed_at descending", col, desc)
	}
}

func TestSortColumnDefaultsWhenAbsent(t *testing.T) {
	col, desc, err := resolveOrderSort("")
	if err != nil {
		t.Fatalf("resolveOrderSort(\"\"): %v", err)
	}
	if col != "placed_at" || !desc {
		t.Fatalf("got col=%q desc=%v, want the default of placed_at descending", col, desc)
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
