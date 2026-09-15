package customer

import (
	"context"
	"errors"

	"github.com/loloDawit/go-admin/services/orders/internal/errs"
)

const defaultPageSize = 20

type Service struct {
	repo        Repository
	pageSizeMax int
}

func NewService(repo Repository, pageSizeMax int) *Service {
	return &Service{repo: repo, pageSizeMax: pageSizeMax}
}

func (s *Service) Create(ctx context.Context, in CreateCustomer) (Customer, error) {
	if in.Email == "" {
		return Customer{}, ErrInvalidCustomerEmail
	}

	created, err := s.repo.Create(ctx, in)
	if err != nil {
		if errors.Is(err, ErrCustomerEmailTaken) {
			return Customer{}, err
		}
		return Customer{}, errs.Wrap(errs.OpCreateCustomer, err)
	}
	return created, nil
}

func (s *Service) Get(ctx context.Context, id int64) (Customer, error) {
	got, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrCustomerNotFound) {
			return Customer{}, err
		}
		return Customer{}, errs.Wrap(errs.OpGetCustomer, err)
	}
	return got, nil
}

func (s *Service) ByEmail(ctx context.Context, email string) (Customer, error) {
	got, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrCustomerNotFound) {
			return Customer{}, err
		}
		return Customer{}, errs.Wrap(errs.OpLookupCustomerByEmail, err)
	}
	return got, nil
}

func (s *Service) List(ctx context.Context, q ListQuery) (Page, error) {
	q.Page, q.PageSize = normalizePage(q.Page, q.PageSize, s.pageSizeMax)

	items, total, err := s.repo.List(ctx, q)
	if err != nil {
		return Page{}, errs.Wrap(errs.OpListCustomers, err)
	}
	return Page{Items: items, Page: q.Page, PageSize: q.PageSize, Total: total}, nil
}

// LifetimeValue is money actually taken, in the currency it was taken in.
func (s *Service) LifetimeValue(ctx context.Context, id int64) (int64, string, error) {
	total, currency, err := s.repo.LifetimeValueMinor(ctx, id)
	if err != nil {
		if errors.Is(err, ErrMixedCurrencyHistory) {
			return 0, "", err
		}
		return 0, "", errs.Wrap(errs.OpCustomerLifetimeValue, err)
	}
	return total, currency, nil
}

func (s *Service) OrderHistory(ctx context.Context, customerID int64, q OrderHistoryQuery) (OrderHistoryPage, error) {
	q.Page, q.PageSize = normalizePage(q.Page, q.PageSize, s.pageSizeMax)

	items, total, err := s.repo.OrderHistory(ctx, customerID, q)
	if err != nil {
		return OrderHistoryPage{}, errs.Wrap(errs.OpCustomerOrderHistory, err)
	}
	return OrderHistoryPage{Items: items, Page: q.Page, PageSize: q.PageSize, Total: total}, nil
}

func normalizePage(page, pageSize, max int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > max {
		pageSize = max
	}
	return page, pageSize
}
