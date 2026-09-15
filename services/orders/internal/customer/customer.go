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

// ListQuery carries a listing request; a zero PageSize means the service's default.
type ListQuery struct {
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
