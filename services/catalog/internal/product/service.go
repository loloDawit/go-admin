package product

import (
	"context"
	"errors"

	"github.com/loloDawit/go-admin/services/catalog/internal/errs"
)

// defaultPageSize applies when a caller sends no pageSize at all; it is
// separate from pageSizeMax, which bounds what a caller may ask for.
const defaultPageSize = 20

type Service struct {
	repo            Repository
	pageSizeMax     int
	defaultCurrency string
}

func NewService(repo Repository, pageSizeMax int, defaultCurrency string) *Service {
	return &Service{repo: repo, pageSizeMax: pageSizeMax, defaultCurrency: defaultCurrency}
}

func (s *Service) Create(ctx context.Context, in CreateProduct) (Product, error) {
	if in.PriceMinor < 0 {
		return Product{}, ErrInvalidPrice
	}
	if in.Currency == "" {
		in.Currency = s.defaultCurrency
	}

	created, err := s.repo.Create(ctx, in)
	if err != nil {
		if errors.Is(err, ErrSkuTaken) {
			return Product{}, err
		}
		return Product{}, errs.Wrap(errs.OpCreateProduct, err)
	}
	return created, nil
}

func (s *Service) Get(ctx context.Context, id int64) (Product, error) {
	got, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrProductNotFound) {
			return Product{}, err
		}
		return Product{}, errs.Wrap(errs.OpGetProduct, err)
	}
	return got, nil
}

func (s *Service) Update(ctx context.Context, id int64, in UpdateProduct) (Product, error) {
	if in.PriceMinor != nil && *in.PriceMinor < 0 {
		return Product{}, ErrInvalidPrice
	}

	updated, err := s.repo.Update(ctx, id, in)
	if err != nil {
		if errors.Is(err, ErrProductNotFound) || errors.Is(err, ErrProductArchived) {
			return Product{}, err
		}
		return Product{}, errs.Wrap(errs.OpUpdateProduct, err)
	}
	return updated, nil
}

func (s *Service) Archive(ctx context.Context, id int64) (Product, error) {
	archived, err := s.repo.Archive(ctx, id)
	if err != nil {
		if errors.Is(err, ErrProductNotFound) || errors.Is(err, ErrProductArchived) {
			return Product{}, err
		}
		return Product{}, errs.Wrap(errs.OpArchiveProduct, err)
	}
	return archived, nil
}

func (s *Service) List(ctx context.Context, q ListQuery) (Page, error) {
	q.Page, q.PageSize = normalizePage(q.Page, q.PageSize, s.pageSizeMax)

	items, total, err := s.repo.List(ctx, q)
	if err != nil {
		if errors.Is(err, ErrInvalidSort) {
			return Page{}, err
		}
		return Page{}, errs.Wrap(errs.OpListProducts, err)
	}
	return Page{Items: items, Page: q.Page, PageSize: q.PageSize, Total: total}, nil
}

func (s *Service) Search(ctx context.Context, q SearchQuery) (Page, error) {
	if q.Text == "" {
		return Page{}, ErrEmptySearchQuery
	}
	q.Page, q.PageSize = normalizePage(q.Page, q.PageSize, s.pageSizeMax)

	items, total, err := s.repo.Search(ctx, q)
	if err != nil {
		return Page{}, errs.Wrap(errs.OpSearchProducts, err)
	}
	return Page{Items: items, Page: q.Page, PageSize: q.PageSize, Total: total}, nil
}

// normalizePage clamps pageSize to max rather than refusing it: the caller
// still gets a page, just not the size they asked for.
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
