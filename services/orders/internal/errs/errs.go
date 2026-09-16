// Package errs is the orders service's one error registry: every sentinel
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

	// ErrCatalogUnavailable: Catalog could not be reached, or answered with a 5xx.
	ErrCatalogUnavailable = errors.New("catalog is unavailable")
	// ErrCatalogTimeout: the call to Catalog exceeded its configured deadline. Distinct from ErrCatalogUnavailable so the two can be mapped to different statuses.
	ErrCatalogTimeout = errors.New("catalog request timed out")
	// ErrCatalogRejected: Catalog answered with a 4xx, or a body this client could not parse.
	ErrCatalogRejected = errors.New("catalog rejected the request")

	ErrCustomerNotFound   = errors.New("customer not found")
	ErrCustomerEmailTaken = errors.New("email is already in use")
	// ErrMixedCurrencyHistory: a lifetime value summed across currencies is a
	// number that means nothing, so it is refused rather than returned.
	ErrMixedCurrencyHistory = errors.New("orders span more than one currency")

	ErrInvalidCustomerEmail = errors.New("email must not be empty")

	ErrUnauthenticated = errors.New("no principal was presented")
	ErrForbidden       = errors.New("caller lacks the required permission")

	// ErrProductUnavailable: a line names a product Catalog did not return, or one that is not active.
	ErrProductUnavailable = errors.New("product unavailable")
	ErrOrderNotFound      = errors.New("order not found")
	// ErrCurrencyMismatch: an order's lines quote more than one currency, so a single total cannot be trusted.
	ErrCurrencyMismatch = errors.New("currency mismatch across order lines")
	ErrEmptyOrder       = errors.New("order has no items")
	ErrInvalidQuantity  = errors.New("order line quantity must be positive")
	// ErrInvalidTransition: the requested status change is not reachable from the order's current status,
	// or targets cancelled/refunded through the generic status endpoint, which those transitions refuse.
	ErrInvalidTransition = errors.New("invalid order status transition")
	// ErrInvalidSort: a caller's sort key is absent from the fixed column allowlist.
	ErrInvalidSort = errors.New("sort is not supported")
)

// Operation strings for Wrap, named here so a log line and its test expectation cannot drift independently.
const (
	OpResolveProducts = "resolve products from catalog"

	OpCreateCustomer        = "create customer"
	OpGetCustomer           = "look up customer by id"
	OpListCustomers         = "list customers"
	OpLookupCustomerByEmail = "look up customer by email"
	OpCustomerLifetimeValue = "compute customer lifetime value"

	OpCreateOrder          = "create order"
	OpGetOrder             = "look up order by id"
	OpTransitionOrder      = "transition order status"
	OpListOrders           = "list orders"
	OpListOrderEvents      = "list order events"

	OpConnectBroker  = "connect to the broker"
	OpEnsureStream   = "ensure the event stream"
	OpEnsureConsumer = "ensure the event consumer"
	OpReadOutbox     = "read the outbox"
	OpMarkPublished  = "mark an outbox row published"
	OpPublishEvent   = "publish an outbox event"
	OpProjectEvent   = "project an event"
	OpCustomerOrderHistory = "look up customer order history"
)

// Wrap names the step that failed, so a log line reads "resolve products from catalog: connection refused", not a bare driver message.
func Wrap(op string, err error) error {
	return fmt.Errorf("%s: %w", op, err)
}
