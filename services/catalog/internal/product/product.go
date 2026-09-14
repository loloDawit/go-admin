// Package product manages catalog products: creation, partial update,
// archival, listing, and search.
package product

import (
	"time"

	"github.com/loloDawit/go-admin/services/catalog/internal/errs"
)

var (
	ErrProductNotFound      = errs.ErrProductNotFound
	ErrSkuTaken             = errs.ErrSkuTaken
	ErrProductArchived      = errs.ErrProductArchived
	ErrInvalidPrice         = errs.ErrInvalidPrice
	ErrInvalidSort          = errs.ErrInvalidSort
	ErrEmptySearchQuery     = errs.ErrEmptySearchQuery
	ErrResolveBatchTooLarge = errs.ErrResolveBatchTooLarge
)

type Status string

const (
	StatusDraft    Status = "draft"
	StatusActive   Status = "active"
	StatusArchived Status = "archived"
)

type Product struct {
	ID          int64
	SKU         string
	Title       string
	Description string
	PriceMinor  int64
	Currency    string
	Status      Status
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// CreateProduct is Create's input. Currency, left blank, is filled from the
// service's configured default rather than defaulting silently in SQL.
type CreateProduct struct {
	SKU         string
	Title       string
	Description string
	PriceMinor  int64
	Currency    string
}

// UpdateProduct is Update's input; a nil field means "leave unchanged".
type UpdateProduct struct {
	Title       *string
	Description *string
	PriceMinor  *int64
	Currency    *string
}

// ListQuery carries a listing request. Status nil means the default view:
// draft and active, archived excluded.
type ListQuery struct {
	Status   *Status
	Sort     string
	Page     int
	PageSize int
}

// SearchQuery mirrors ListQuery's status/paging shape alongside the search text.
type SearchQuery struct {
	Text     string
	Status   *Status
	Page     int
	PageSize int
}

// Page is the response shape List and Search both answer with; Total comes
// from a separate count query, never a window function over the paged rows.
type Page struct {
	Items    []Product
	Page     int
	PageSize int
	Total    int
}
