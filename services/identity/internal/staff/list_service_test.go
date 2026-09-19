package staff_test

import (
	"testing"

	"github.com/loloDawit/go-admin/services/identity/internal/session"
	"github.com/loloDawit/go-admin/services/identity/internal/staff"
)

const testStaffPageSizeMax = 100

// A page size beyond the configured maximum is clamped, not refused, and the
// response states the effective size rather than echoing the request.
func TestStaffPageSizeIsClampedNotRefused(t *testing.T) {
	svc := staff.NewService(&fakeRepository{}, session.NewHasher(testCost), testStaffPageSizeMax)

	page, err := svc.List(t.Context(), staff.ListQuery{Page: 1, PageSize: testStaffPageSizeMax + 500})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if page.PageSize != testStaffPageSizeMax {
		t.Fatalf("PageSize: want clamped to %d, got %d", testStaffPageSizeMax, page.PageSize)
	}
}

func TestStaffPageDefaultsWhenAbsent(t *testing.T) {
	svc := staff.NewService(&fakeRepository{}, session.NewHasher(testCost), testStaffPageSizeMax)

	page, err := svc.List(t.Context(), staff.ListQuery{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if page.Page != 1 {
		t.Fatalf("Page: want default 1, got %d", page.Page)
	}
	if page.PageSize != staff.DefaultPageSize {
		t.Fatalf("PageSize: want default %d, got %d", staff.DefaultPageSize, page.PageSize)
	}
}
