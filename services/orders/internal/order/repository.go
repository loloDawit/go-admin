package order

import (
	"context"
	"time"

	"github.com/loloDawit/go-admin/services/orders/internal/catalog"
)

// Repository is the data access this package needs; every error not named
// below is a driver-level failure the service wraps.
type Repository interface {
	// RunInTx gives fn a repository bound to one transaction, so the order
	// number read and every insert it drives commit or roll back together.
	RunInTx(ctx context.Context, fn func(Repository) error) error

	// NextOrderNumber takes the sequence's value and the database's clock in
	// one read, so the order number's year and PlacedAt cannot disagree.
	NextOrderNumber(ctx context.Context) (seq int64, at time.Time, err error)

	InsertOrder(ctx context.Context, in NewOrder) (Order, error)
	InsertItems(ctx context.Context, orderID int64, items []Item) ([]Item, error)
	InsertEvent(ctx context.Context, in NewEvent) error

	GetByID(ctx context.Context, id int64) (Order, error)

	// GetStatusForUpdate locks the order row for the remainder of the
	// caller's transaction, so the status it reports cannot change under a
	// concurrent transition before UpdateStatus writes the new one.
	GetStatusForUpdate(ctx context.Context, id int64) (Status, error)

	// UpdateStatus is a compare-and-set: it writes only if the row's status
	// still matches from. Call it only after GetStatusForUpdate in the same
	// transaction, which is what makes that guaranteed rather than racy.
	UpdateStatus(ctx context.Context, id int64, from, to Status) (Order, error)

	ListOrders(ctx context.Context, q ListQuery) ([]Order, int, error)

	ListEvents(ctx context.Context, orderID int64) ([]Event, error)
}

// NewOrder is InsertOrder's input: the header row before its items exist.
type NewOrder struct {
	Number     string
	CustomerID int64
	TotalMinor int64
	Currency   string
	PlacedAt   time.Time
}

// NewEvent is InsertEvent's input. FromStatus nil means the order coming
// into existence: there is no prior status to record.
type NewEvent struct {
	OrderID    int64
	FromStatus *Status
	ToStatus   Status
	ActorID    string
	Reason     string
}

// ProductResolver is the subset of catalog.Client the service calls; a fake
// satisfies it in tests without a network round trip.
type ProductResolver interface {
	Resolve(ctx context.Context, ids []string) ([]catalog.Product, error)
}
