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

	// LifetimeValueMinor excludes cancelled and refunded orders: money
	// returned to the customer is not part of their lifetime value.
	LifetimeValueMinor(ctx context.Context, id int64) (int64, error)
}
