package product

import "context"

// Repository is the data access this package needs; every error not named
// below is a driver-level failure the service wraps.
type Repository interface {
	Create(ctx context.Context, in CreateProduct) (Product, error)

	// GetByID returns a product regardless of status: archived products stay
	// retrievable by id even though they drop out of List and Search.
	GetByID(ctx context.Context, id int64) (Product, error)

	// Update refuses ErrProductArchived rather than silently applying the change.
	Update(ctx context.Context, id int64, in UpdateProduct) (Product, error)

	// Archive is a no-op turned error, not a silent success, against an
	// already-archived product.
	Archive(ctx context.Context, id int64) (Product, error)

	// List and Search both return the page's items and the total matching
	// row count from an independent query.
	List(ctx context.Context, q ListQuery) ([]Product, int, error)
	Search(ctx context.Context, q SearchQuery) ([]Product, int, error)

	// ResolveByIDs answers a batch lookup in one query: archived products are
	// included, an unknown id is simply absent from the result rather than an
	// error, and the result is ordered to match the order of ids.
	ResolveByIDs(ctx context.Context, ids []int64) ([]Product, error)
}
