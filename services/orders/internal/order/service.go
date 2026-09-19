package order

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/loloDawit/go-admin/services/orders/internal/catalog"

	"github.com/loloDawit/go-admin/services/orders/internal/errs"
	"github.com/loloDawit/go-admin/services/orders/internal/outbox"
)

const defaultPageSize = 20

type Service struct {
	repo        Repository
	catalog     ProductResolver
	pageSizeMax int
}

func NewService(repo Repository, resolver ProductResolver, pageSizeMax int) *Service {
	return &Service{repo: repo, catalog: resolver, pageSizeMax: pageSizeMax}
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

		if err := tx.InsertEvent(ctx, NewEvent{
			OrderID:  created.ID,
			ToStatus: created.Status,
			ActorID:  in.ActorID,
		}); err != nil {
			return err
		}

		rec, err := outbox.New(ctx, outbox.TypeOrderCreated, strconv.FormatInt(created.ID, 10), in.ActorID, created.PlacedAt, map[string]any{
			"totalMinor": created.TotalMinor,
			"currency":   created.Currency,
			"placedAt":   created.PlacedAt,
		})
		if err != nil {
			return err
		}
		return tx.InsertOutbox(ctx, rec)
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

// SetStatus drives the generic status endpoint, which must never reach
// cancelled or refunded: those have preconditions of their own, modelled by
// Cancel and Refund, and a generic endpoint that could reach them would be a
// way around those rules.
func (s *Service) SetStatus(ctx context.Context, id int64, actorID string, to Status) (Order, error) {
	if to == StatusCancelled || to == StatusRefunded {
		return Order{}, ErrInvalidTransition
	}
	return s.transition(ctx, id, actorID, to, "")
}

// Cancel is reachable from pending, paid and packed, per the transition
// table; once shipped, an order can only be refunded, not cancelled.
func (s *Service) Cancel(ctx context.Context, id int64, actorID, reason string) (Order, error) {
	return s.transition(ctx, id, actorID, StatusCancelled, reason)
}

// Refund is reachable from paid, packed, shipped and delivered: every one of
// those is reached only by way of paid, so no separate history check is
// needed to require that the order was paid at some point.
func (s *Service) Refund(ctx context.Context, id int64, actorID, reason string) (Order, error) {
	return s.transition(ctx, id, actorID, StatusRefunded, reason)
}

// transition locks the order's current status, checks it against the
// transition table, and writes the new status and its event in the same
// transaction: a status column and an event log that could disagree would
// be worse than no event log at all.
func (s *Service) transition(ctx context.Context, id int64, actorID string, to Status, reason string) (Order, error) {
	var updated Order
	err := s.repo.RunInTx(ctx, func(tx Repository) error {
		current, err := tx.GetStatusForUpdate(ctx, id)
		if err != nil {
			return err
		}
		if !CanTransition(current, to) {
			return ErrInvalidTransition
		}
		updated, err = tx.UpdateStatus(ctx, id, current, to)
		if err != nil {
			return err
		}
		from := current
		if err := tx.InsertEvent(ctx, NewEvent{OrderID: id, FromStatus: &from, ToStatus: to, ActorID: actorID, Reason: reason}); err != nil {
			return err
		}

		rec, err := outbox.New(ctx, outbox.TypeOrderStatusChanged, strconv.FormatInt(id, 10), actorID, updated.UpdatedAt, map[string]any{
			"from":       string(from),
			"to":         string(to),
			"totalMinor": updated.TotalMinor,
			"currency":   updated.Currency,
		})
		if err != nil {
			return err
		}
		return tx.InsertOutbox(ctx, rec)
	})
	if err != nil {
		if errors.Is(err, ErrOrderNotFound) || errors.Is(err, ErrInvalidTransition) {
			return Order{}, err
		}
		return Order{}, errs.Wrap(errs.OpTransitionOrder, err)
	}
	return updated, nil
}

// Events reads the order first so a missing order is a 404 rather than an
// empty list, which would claim an order exists and has no history.
func (s *Service) Events(ctx context.Context, id int64) ([]Event, error) {
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		if errors.Is(err, ErrOrderNotFound) {
			return nil, err
		}
		return nil, errs.Wrap(errs.OpListOrderEvents, err)
	}

	events, err := s.repo.ListEvents(ctx, id)
	if err != nil {
		return nil, errs.Wrap(errs.OpListOrderEvents, err)
	}
	return events, nil
}

func (s *Service) List(ctx context.Context, q ListQuery) (Page, error) {
	q.Page, q.PageSize = normalizePage(q.Page, q.PageSize, s.pageSizeMax)

	items, total, err := s.repo.ListOrders(ctx, q)
	if err != nil {
		if errors.Is(err, ErrInvalidSort) {
			return Page{}, err
		}
		return Page{}, errs.Wrap(errs.OpListOrders, err)
	}
	return Page{Items: items, Page: q.Page, PageSize: q.PageSize, Total: total}, nil
}

// normalizePage clamps pageSize to max rather than refusing it: the caller
// still gets a page, just not the size they asked for.
func normalizePage(page, pageSize, max int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > max {
		pageSize = max
	}
	return page, pageSize
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
