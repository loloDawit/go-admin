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
	ErrInvalidImageMaxBytes = errors.New("IMAGE_MAX_BYTES must be positive")
	ErrInvalidPageSizeMax   = errors.New("PRODUCT_PAGE_SIZE_MAX must be positive")
	ErrInvalidCurrency      = errors.New("DEFAULT_CURRENCY must be a 3-letter code")
	ErrShortPrincipalKey    = errors.New("PRINCIPAL_SIGNING_KEY must be at least 32 bytes")

	ErrProductNotFound = errors.New("product not found")
	ErrSkuTaken        = errors.New("sku is already in use")
	ErrProductArchived = errors.New("product is archived")
	ErrInvalidPrice    = errors.New("price must not be negative")

	// ErrInvalidSort: the sort key named no column in the allowlist.
	ErrInvalidSort = errors.New("sort is not supported")

	ErrEmptySearchQuery = errors.New("q must not be empty")
)

// Operation strings for Wrap, named here so a log line and its test expectation cannot drift independently.
const (
	OpImageStoreConnect = "image store: connect"
	OpImageStorePut     = "image store: put"
	OpImageStorePresign = "image store: presign"
	OpImageStoreDelete  = "image store: delete"

	OpCreateProduct  = "create product"
	OpUpdateProduct  = "update product"
	OpArchiveProduct = "archive product"
	OpGetProduct     = "look up product by id"
	OpListProducts   = "list products"
	OpSearchProducts = "search products"
)

// Wrap names the step that failed, so a log line reads "look up product: connection refused", not a bare driver message.
func Wrap(op string, err error) error {
	return fmt.Errorf("%s: %w", op, err)
}
