// Package errs is the catalog service's one error registry: every sentinel
// and operation string a wrap names lives here. httperr maps the ones it
// recognizes to a client-facing status; the rest are startup-only and never
// reach httperr, since they can only occur before the server starts serving
// requests.
package errs

import (
	"errors"
	"fmt"
)

// Sentinels. Where httperr maps one to a status code, nothing else decides one.
var (
	// httperr maps all three to 503: each means not ready, never a request fault.
	ErrNoMigrations        = errors.New("no migrations applied")
	ErrDirtySchema         = errors.New("schema is in a dirty state")
	ErrDatabaseUnavailable = errors.New("database is unavailable")

	// Startup-only: raised while validating config, before the server accepts any request.
	ErrInvalidImageMaxBytes       = errors.New("IMAGE_MAX_BYTES must be positive")
	ErrInvalidPageSizeMax         = errors.New("PRODUCT_PAGE_SIZE_MAX must be positive")
	ErrInvalidResolveBatchMax     = errors.New("PRODUCT_RESOLVE_BATCH_MAX must be positive")
	ErrInvalidMaxRequestBodyBytes = errors.New("MAX_REQUEST_BODY_BYTES must be positive")
	ErrInvalidCurrency            = errors.New("DEFAULT_CURRENCY must be a 3-letter code")
	ErrShortPrincipalKey          = errors.New("PRINCIPAL_SIGNING_KEY must be at least 32 bytes")

	ErrProductNotFound = errors.New("product not found")
	ErrSkuTaken        = errors.New("sku is already in use")
	ErrProductArchived = errors.New("product is archived")
	ErrInvalidPrice    = errors.New("price must not be negative")

	// ErrInvalidSort: the sort key named no column in the allowlist.
	ErrInvalidSort = errors.New("sort is not supported")

	ErrEmptySearchQuery = errors.New("q must not be empty")

	// ErrResolveBatchTooLarge: the requested id list exceeds the configured cap.
	ErrResolveBatchTooLarge = errors.New("too many ids requested")

	// ErrUnsupportedImageType: the sniffed content type is outside the allowlist, or does not match what was declared.
	ErrUnsupportedImageType = errors.New("unsupported image type")
	ErrImageTooLarge        = errors.New("image exceeds the maximum upload size")
	ErrImageNotFound        = errors.New("image not found")
)

// Operation strings for Wrap, named here so a log line and its test expectation cannot drift independently.
const (
	OpImageStoreConnect = "image store: connect"
	OpImageStorePut     = "image store: put"
	OpImageStorePresign = "image store: presign"
	OpImageStoreDelete  = "image store: delete"

	OpCreateProduct   = "create product"
	OpUpdateProduct   = "update product"
	OpArchiveProduct  = "archive product"
	OpGetProduct      = "look up product by id"
	OpListProducts    = "list products"
	OpSearchProducts  = "search products"
	OpResolveProducts = "resolve products"

	OpUploadImage = "upload image"
	OpDeleteImage = "delete image"
	OpListImages  = "list images for product"
)

// Wrap names the step that failed, so a log line reads "look up product: connection refused", not a bare driver message.
func Wrap(op string, err error) error {
	return fmt.Errorf("%s: %w", op, err)
}
