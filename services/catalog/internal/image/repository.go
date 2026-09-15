package image

import "context"

// Repository is the data access this package needs; every error not named
// below is a driver-level failure the service wraps.
type Repository interface {
	Create(ctx context.Context, img Image) (Image, error)

	// Get scopes by productID as well as id: an image belongs to exactly one
	// product and a mismatched pair is ErrImageNotFound, not a different
	// product's image.
	Get(ctx context.Context, productID, imageID int64) (Image, error)

	Delete(ctx context.Context, productID, imageID int64) error

	ListByProduct(ctx context.Context, productID int64) ([]Image, error)
}
