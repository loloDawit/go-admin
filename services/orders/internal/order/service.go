package order

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/loloDawit/go-admin/services/orders/internal/catalog"
	"github.com/loloDawit/go-admin/services/orders/internal/errs"
)

type Service struct {
	repo    Repository
	catalog ProductResolver
}

func NewService(repo Repository, resolver ProductResolver) *Service {
	return &Service{repo: repo, catalog: resolver}
}

// Create resolves every line against Catalog, then opens one transaction to
// write the order, its items, and its first event. Resolving before opening
// the transaction is what makes "no partial order on failure" true by
// construction rather than by cleanup.
func (s *Service) Create(ctx context.Context, in CreateOrder) (Order, error) {
	if err := validateShape(in); err != nil {
		return Order{}, err
	}

	ids := make([]string, len(in.Items))
	for i, item := range in.Items {
		ids[i] = item.ProductID
	}

	products, err := s.catalog.Resolve(ctx, ids)
	if err != nil {
		return Order{}, errs.Wrap(errs.OpCreateOrder, err)
	}

	items, currency, total, err := buildItems(in.Items, products)
	if err != nil {
		return Order{}, err
	}

	var created Order
	err = s.repo.RunInTx(ctx, func(tx Repository) error {
		seq, at, err := tx.NextOrderNumber(ctx)
		if err != nil {
			return err
		}

		created, err = tx.InsertOrder(ctx, NewOrder{
			Number:     formatOrderNumber(at.Year(), seq),
			CustomerID: in.CustomerID,
			TotalMinor: total,
			Currency:   currency,
			PlacedAt:   at,
		})
		if err != nil {
			return err
		}

		insertedItems, err := tx.InsertItems(ctx, created.ID, items)
		if err != nil {
			return err
		}
		created.Items = insertedItems

		return tx.InsertEvent(ctx, NewEvent{
			OrderID:  created.ID,
			ToStatus: created.Status,
			ActorID:  in.ActorID,
		})
	})
	if err != nil {
		if errors.Is(err, errs.ErrCustomerNotFound) {
			return Order{}, err
		}
		return Order{}, errs.Wrap(errs.OpCreateOrder, err)
	}
	return created, nil
}

func (s *Service) Get(ctx context.Context, id int64) (Order, error) {
	got, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrOrderNotFound) {
			return Order{}, err
		}
		return Order{}, errs.Wrap(errs.OpGetOrder, err)
	}
	return got, nil
}

// validateShape rejects a request Catalog would never even see: an order
// with no lines, or a line quantity a CHECK constraint would otherwise catch
// as an opaque driver error.
func validateShape(in CreateOrder) error {
	if len(in.Items) == 0 {
		return ErrEmptyOrder
	}
	for _, item := range in.Items {
		if item.Quantity <= 0 {
			return fmt.Errorf("%w: product %s", ErrInvalidQuantity, item.ProductID)
		}
	}
	return nil
}

// buildItems matches every requested line to Catalog's response by id, never
// by position: Catalog omits an id it cannot match rather than erroring, so
// a positional zip would silently attach the wrong product to a line.
func buildItems(requested []CreateOrderItem, resolved []catalog.Product) ([]Item, string, int64, error) {
	byID := make(map[string]catalog.Product, len(resolved))
	for _, p := range resolved {
		byID[p.ID] = p
	}

	items := make([]Item, len(requested))
	var currency string
	var total int64
	for i, line := range requested {
		p, ok := byID[line.ProductID]
		if !ok || p.Status != activeProductStatus {
			return nil, "", 0, fmt.Errorf("%w: product %s", ErrProductUnavailable, line.ProductID)
		}
		if currency == "" {
			currency = p.Currency
		} else if currency != p.Currency {
			return nil, "", 0, ErrCurrencyMismatch
		}

		productID, err := strconv.ParseInt(p.ID, 10, 64)
		if err != nil {
			return nil, "", 0, fmt.Errorf("catalog product id %q is not numeric: %w", p.ID, err)
		}

		lineTotal := p.PriceMinor * int64(line.Quantity)
		total += lineTotal
		items[i] = Item{
			ProductID:      productID,
			TitleSnapshot:  p.Title,
			UnitPriceMinor: p.PriceMinor,
			Currency:       p.Currency,
			Quantity:       line.Quantity,
			LineTotalMinor: lineTotal,
		}
	}
	return items, currency, total, nil
}
