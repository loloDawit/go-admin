// Package customer manages Orders' customers: creation, lookup by id or
// email, listing, and lifetime value.
package customer

import (
	"time"

	"github.com/loloDawit/go-admin/services/orders/internal/errs"
)

var (
	ErrCustomerNotFound     = errs.ErrCustomerNotFound
	ErrCustomerEmailTaken   = errs.ErrCustomerEmailTaken
	ErrInvalidCustomerEmail = errs.ErrInvalidCustomerEmail
	ErrMixedCurrencyHistory = errs.ErrMixedCurrencyHistory
	ErrInvalidSort          = errs.ErrInvalidSort
)

type Customer struct {
	ID        int64
	Email     string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CreateCustomer struct {
	Email string
	Name  string
}

// ListQuery carries a listing request; a zero PageSize means the service's
// default. An empty Q means no search filter.
type ListQuery struct {
	Q        string
	Sort     string
	Page     int
	PageSize int
}

// Page is the response shape List answers with; Total comes from a separate
// count query, never a window function over the paged rows.
type Page struct {
	Items    []Customer
	Page     int
	PageSize int
	Total    int
}

// OrderSummary is one row of a customer's order history: header fields only,
// queried directly against the orders table the same way LifetimeValueMinor
// already does, rather than importing the order package's full model.
type OrderSummary struct {
	ID         int64
	Number     string
	Status     string
	TotalMinor int64
	Currency   string
	PlacedAt   time.Time
}

// OrderHistoryQuery carries paging for OrderHistory; a zero PageSize means
// the service's default.
type OrderHistoryQuery struct {
	Page     int
	PageSize int
}

// OrderHistoryPage is OrderHistory's response shape; Total comes from a
// separate count query.
type OrderHistoryPage struct {
	Items    []OrderSummary
	Page     int
	PageSize int
	Total    int
}
