package models

import (
	"math"
	"testing"
)

func TestLastPageRoundsUpOnPartialFinalPage(t *testing.T) {
	cases := []struct {
		total, perPage, want int
	}{
		{total: 12, perPage: 5, want: 3},
		{total: 10, perPage: 5, want: 2},
		{total: 1, perPage: 5, want: 1},
		{total: 0, perPage: 5, want: 1}, // an empty list still has one page
		{total: 26, perPage: 25, want: 2},
	}

	for _, c := range cases {
		if got := lastPage(int64(c.total), c.perPage); got != c.want {
			t.Errorf("lastPage(%d, %d) = %d, want %d", c.total, c.perPage, got, c.want)
		}
	}
}

func TestNormalizePerPageIsBounded(t *testing.T) {
	cases := []struct{ in, want int }{
		{in: 0, want: defaultPerPage},
		{in: -5, want: defaultPerPage},
		{in: 10, want: 10},
		{in: 5000, want: maxPerPage}, // must not let a client request everything
	}
	for _, c := range cases {
		if got := normalizePerPage(c.in); got != c.want {
			t.Errorf("normalizePerPage(%d) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestNormalizePageIsBounded(t *testing.T) {
	cases := []struct{ in, want int }{
		{in: 0, want: 1},
		{in: -5, want: 1},
		{in: 5, want: 5},
		{in: math.MaxInt64, want: maxPage}, // an oversized ?page= must not overflow the offset arithmetic
	}
	for _, c := range cases {
		if got := normalizePage(c.in); got != c.want {
			t.Errorf("normalizePage(%d) = %d, want %d", c.in, got, c.want)
		}
	}
}
