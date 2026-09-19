package order

import "testing"

// A listing row carries the customer's name, not just the id: a list of order
// numbers against opaque customer ids is not something staff can scan.
func TestListRowCarriesTheCustomerName(t *testing.T) {
	page := newOrderPageResponse(Page{
		Items: []Order{{
			ID:           7,
			Number:       "ORD-2026-000007",
			CustomerID:   42,
			CustomerName: "Ada Okonkwo",
			Status:       StatusPaid,
			TotalMinor:   4000,
			Currency:     "USD",
		}},
		Page: 1, PageSize: 20, Total: 1,
	})

	if len(page.Items) != 1 {
		t.Fatalf("want one row, got %d", len(page.Items))
	}
	if page.Items[0].CustomerName != "Ada Okonkwo" {
		t.Fatalf("CustomerName = %q, want Ada Okonkwo", page.Items[0].CustomerName)
	}
	if page.Items[0].CustomerID != "42" {
		t.Fatalf("CustomerID = %q, want 42", page.Items[0].CustomerID)
	}
}
