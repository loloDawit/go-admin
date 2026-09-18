package reporting_test

import (
	"context"
	"testing"
	"time"

	"github.com/loloDawit/go-admin/services/orders/internal/reporting"
)

type fakeReadRepo struct {
	counts    map[string]int
	recent    []reporting.RecentOrder
	revenue   []reporting.RevenueDay
	limitGot  int
	windowGot int
}

func (f *fakeReadRepo) StatusCounts(context.Context) (map[string]int, error) {
	return f.counts, nil
}

func (f *fakeReadRepo) RecentOrders(_ context.Context, limit int) ([]reporting.RecentOrder, error) {
	f.limitGot = limit
	return f.recent, nil
}

func (f *fakeReadRepo) Revenue(_ context.Context, windowDays int) ([]reporting.RevenueDay, error) {
	f.windowGot = windowDays
	return f.revenue, nil
}

// Net revenue is recognised minus refunded, per currency. Summing across
// currencies would produce a number that means nothing, which is the rule
// LifetimeValue already enforces by refusing.
func TestDashboardReportsNetRevenuePerCurrency(t *testing.T) {
	day := time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC)
	repo := &fakeReadRepo{
		counts: map[string]int{"pending": 2, "paid": 1},
		revenue: []reporting.RevenueDay{
			{Day: day, Currency: "USD", PlacedCount: 3, PaidCount: 2, RecognisedMinor: 5000, RefundedMinor: 1200},
			{Day: day, Currency: "EUR", PlacedCount: 1, PaidCount: 1, RecognisedMinor: 900, RefundedMinor: 0},
		},
	}
	svc := reporting.NewService(repo, 30)

	report, err := svc.Dashboard(t.Context())
	if err != nil {
		t.Fatalf("Dashboard: %v", err)
	}
	if repo.windowGot != 30 {
		t.Fatalf("window = %d days, want the configured 30", repo.windowGot)
	}
	if repo.limitGot != reporting.RecentOrderLimit {
		t.Fatalf("recent limit = %d, want %d", repo.limitGot, reporting.RecentOrderLimit)
	}
	if len(report.Revenue) != 2 {
		t.Fatalf("revenue rows = %d, want 2", len(report.Revenue))
	}
	if report.Revenue[0].NetMinor != 3800 {
		t.Fatalf("USD net = %d, want 3800", report.Revenue[0].NetMinor)
	}
	if report.Revenue[1].NetMinor != 900 {
		t.Fatalf("EUR net = %d, want 900", report.Revenue[1].NetMinor)
	}
	if report.Counts["pending"] != 2 {
		t.Fatalf("counts = %+v", report.Counts)
	}
}
