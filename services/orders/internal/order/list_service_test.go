package order_test

import (
	"testing"

	"github.com/loloDawit/go-admin/services/orders/internal/order"
)

// PageSize is clamped to the configured max, not refused; the response
// states the effective size rather than echoing what the caller asked for.
func TestPageSizeIsClampedNotRefused(t *testing.T) {
	repo := &fakeRepo{listResult: []order.Order{}, listTotal: 0}
	svc := order.NewService(repo, &fakeCatalog{}, testPageSizeMax)

	page, err := svc.List(t.Context(), order.ListQuery{Page: 1, PageSize: testPageSizeMax + 1000})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if page.PageSize != testPageSizeMax {
		t.Fatalf("PageSize: want clamped to %d, got %d", testPageSizeMax, page.PageSize)
	}
}

func TestPageSizeDefaultsWhenAbsent(t *testing.T) {
	repo := &fakeRepo{}
	svc := order.NewService(repo, &fakeCatalog{}, testPageSizeMax)

	page, err := svc.List(t.Context(), order.ListQuery{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if page.PageSize != 20 {
		t.Fatalf("PageSize: want default 20, got %d", page.PageSize)
	}
	if page.Page != 1 {
		t.Fatalf("Page: want default 1, got %d", page.Page)
	}
}

func TestFilterByStatusAndCustomer(t *testing.T) {
	repo := &fakeRepo{}
	svc := order.NewService(repo, &fakeCatalog{}, testPageSizeMax)

	status := order.StatusPaid
	customerID := int64(42)
	_, err := svc.List(t.Context(), order.ListQuery{Status: &status, CustomerID: &customerID, Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if repo.lastListQ.Status == nil || *repo.lastListQ.Status != order.StatusPaid {
		t.Fatalf("status filter did not reach the repository: %+v", repo.lastListQ)
	}
	if repo.lastListQ.CustomerID == nil || *repo.lastListQ.CustomerID != customerID {
		t.Fatalf("customer filter did not reach the repository: %+v", repo.lastListQ)
	}
}

func TestListSurfacesAnInvalidSort(t *testing.T) {
	repo := &fakeRepo{listErr: order.ErrInvalidSort}
	svc := order.NewService(repo, &fakeCatalog{}, testPageSizeMax)

	_, err := svc.List(t.Context(), order.ListQuery{Sort: "not-a-column"})
	if err != order.ErrInvalidSort {
		t.Fatalf("want ErrInvalidSort, got %v", err)
	}
}
