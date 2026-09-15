// Package order manages order creation: resolving lines against Catalog,
// snapshotting price and title at purchase time, and writing the order, its
// items, and its first event in one transaction.
package order

import (
	"fmt"
	"time"

	"github.com/loloDawit/go-admin/services/orders/internal/errs"
)

var (
	ErrProductUnavailable = errs.ErrProductUnavailable
	ErrOrderNotFound      = errs.ErrOrderNotFound
	ErrCurrencyMismatch   = errs.ErrCurrencyMismatch
	ErrEmptyOrder         = errs.ErrEmptyOrder
	ErrInvalidQuantity    = errs.ErrInvalidQuantity
)

type Status string

const (
	StatusPending   Status = "pending"
	StatusPaid      Status = "paid"
	StatusPacked    Status = "packed"
	StatusShipped   Status = "shipped"
	StatusDelivered Status = "delivered"
	StatusCancelled Status = "cancelled"
	StatusRefunded  Status = "refunded"
)

// activeProductStatus is Catalog's wire value for a purchasable product; any
// other status (draft, archived) is not open for new order lines.
const activeProductStatus = "active"

type Order struct {
	ID         int64
	Number     string
	CustomerID int64
	Status     Status
	TotalMinor int64
	Currency   string
	PlacedAt   time.Time
	UpdatedAt  time.Time
	Items      []Item
}

type Item struct {
	ID             int64
	ProductID      int64
	TitleSnapshot  string
	UnitPriceMinor int64
	Currency       string
	Quantity       int
	LineTotalMinor int64
}

// CreateOrder is Create's input. ActorID is set by the handler from the
// verified principal, never decoded from the request body.
type CreateOrder struct {
	CustomerID int64
	ActorID    string
	Items      []CreateOrderItem
}

type CreateOrderItem struct {
	ProductID string
	Quantity  int
}

// formatOrderNumber is done in Go, not SQL: lpad truncates past six digits
// (1,000,000 -> "100000", colliding with order 100,000) and to_char overflows
// to "######" instead of widening.
func formatOrderNumber(year int, seq int64) string {
	return fmt.Sprintf("ORD-%d-%06d", year, seq)
}
