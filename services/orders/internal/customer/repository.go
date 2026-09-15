package customer

import "context"

// Repository is the data access this package needs; every error not named
// below is a driver-level failure the service wraps.
type Repository interface {
	Create(ctx context.Context, in CreateCustomer) (Customer, error)

	GetByID(ctx context.Context, id int64) (Customer, error)

	// GetByEmail relies on the email column being citext: the lookup is
	// case-insensitive without a lower() index.
	GetByEmail(ctx context.Context, email string) (Customer, error)

	List(ctx context.Context, q ListQuery) ([]Customer, int, error)

	// LifetimeValueMinor is money actually taken, with the currency it was taken in.
	LifetimeValueMinor(ctx context.Context, id int64) (int64, string, error)

	// OrderHistory is scoped to customerID by its WHERE clause; a query
	// missing it would still pass with a single customer in the database,
	// which is why it is tested with two.
	OrderHistory(ctx context.Context, customerID int64, q OrderHistoryQuery) ([]OrderSummary, int, error)
}
