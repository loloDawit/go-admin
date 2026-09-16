package reporting

import (
	"context"
	"time"

	"github.com/loloDawit/go-admin/services/orders/internal/errs"
)

// RecentOrderLimit is the dashboard's list length: a layout decision, not
// something an operator tunes.
const RecentOrderLimit = 10

type RecentOrder struct {
	ID         int64
	Number     string
	Status     string
	TotalMinor int64
	Currency   string
	PlacedAt   time.Time
}

type RevenueDay struct {
	Day             time.Time
	Currency        string
	PlacedCount     int
	PaidCount       int
	RecognisedMinor int64
	RefundedMinor   int64
	NetMinor        int64
}

type ReadRepository interface {
	StatusCounts(ctx context.Context) (map[string]int, error)
	RecentOrders(ctx context.Context, limit int) ([]RecentOrder, error)
	Revenue(ctx context.Context, windowDays int) ([]RevenueDay, error)
}

type Report struct {
	Counts  map[string]int
	Recent  []RecentOrder
	Revenue []RevenueDay
}

type Service struct {
	repo       ReadRepository
	windowDays int
}

func NewService(repo ReadRepository, windowDays int) *Service {
	return &Service{repo: repo, windowDays: windowDays}
}

func (s *Service) Dashboard(ctx context.Context) (Report, error) {
	counts, err := s.repo.StatusCounts(ctx)
	if err != nil {
		return Report{}, errs.Wrap(errs.OpDashboardReport, err)
	}
	recent, err := s.repo.RecentOrders(ctx, RecentOrderLimit)
	if err != nil {
		return Report{}, errs.Wrap(errs.OpDashboardReport, err)
	}
	revenue, err := s.repo.Revenue(ctx, s.windowDays)
	if err != nil {
		return Report{}, errs.Wrap(errs.OpDashboardReport, err)
	}

	for i := range revenue {
		revenue[i].NetMinor = revenue[i].RecognisedMinor - revenue[i].RefundedMinor
	}
	return Report{Counts: counts, Recent: recent, Revenue: revenue}, nil
}
