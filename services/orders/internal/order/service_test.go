package order_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/loloDawit/go-admin/services/orders/internal/catalog"
	"github.com/loloDawit/go-admin/services/orders/internal/order"
)

const testPageSizeMax = 50

// fakeCatalog is a Resolve double: no HTTP round trip, no retry semantics.
type fakeCatalog struct {
	products []catalog.Product
	err      error
	calls    int
}

func (f *fakeCatalog) Resolve(_ context.Context, _ []string) ([]catalog.Product, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return f.products, nil
}

// fakeRepo is an in-memory Repository double; RunInTx runs fn against itself
// directly, so tests can assert whether it was ever entered at all.
type fakeRepo struct {
	runInTxCalls int
	nextSeq      int64
	nextAt       time.Time

	insertOrderErr error
	insertedOrder  order.NewOrder
	insertedItems  []order.Item
	insertedEvent  order.NewEvent

	getByIDResult order.Order
	getByIDErr    error

	status              order.Status
	getStatusErr        error
	updateStatusCalls   int
	updateStatusErr     error
	insertedEventReason string

	listResult []order.Order
	listTotal  int
	listErr    error
	lastListQ  order.ListQuery
}

func (f *fakeRepo) RunInTx(_ context.Context, fn func(order.Repository) error) error {
	f.runInTxCalls++
	return fn(f)
}

func (f *fakeRepo) NextOrderNumber(context.Context) (int64, time.Time, error) {
	return f.nextSeq, f.nextAt, nil
}

func (f *fakeRepo) InsertOrder(_ context.Context, in order.NewOrder) (order.Order, error) {
	if f.insertOrderErr != nil {
		return order.Order{}, f.insertOrderErr
	}
	f.insertedOrder = in
	return order.Order{
		ID: 1, Number: in.Number, CustomerID: in.CustomerID,
		Status: order.StatusPending, TotalMinor: in.TotalMinor,
		Currency: in.Currency, PlacedAt: in.PlacedAt,
	}, nil
}

func (f *fakeRepo) InsertItems(_ context.Context, _ int64, items []order.Item) ([]order.Item, error) {
	f.insertedItems = items
	out := make([]order.Item, len(items))
	for i, it := range items {
		it.ID = int64(i + 1)
		out[i] = it
	}
	return out, nil
}

func (f *fakeRepo) InsertEvent(_ context.Context, in order.NewEvent) error {
	f.insertedEvent = in
	f.insertedEventReason = in.Reason
	return nil
}

func (f *fakeRepo) GetByID(context.Context, int64) (order.Order, error) {
	return f.getByIDResult, f.getByIDErr
}

func (f *fakeRepo) GetStatusForUpdate(context.Context, int64) (order.Status, error) {
	if f.getStatusErr != nil {
		return "", f.getStatusErr
	}
	return f.status, nil
}

func (f *fakeRepo) UpdateStatus(_ context.Context, id int64, from, to order.Status) (order.Order, error) {
	f.updateStatusCalls++
	if f.updateStatusErr != nil {
		return order.Order{}, f.updateStatusErr
	}
	f.status = to
	return order.Order{ID: id, Status: to}, nil
}

func (f *fakeRepo) ListOrders(_ context.Context, q order.ListQuery) ([]order.Order, int, error) {
	f.lastListQ = q
	return f.listResult, f.listTotal, f.listErr
}

func newActiveProduct(id, title, currency string, priceMinor int64) catalog.Product {
	return catalog.Product{ID: id, Title: title, PriceMinor: priceMinor, Currency: currency, Status: "active"}
}

func TestCreateSnapshotsTitleAndPriceFromCatalog(t *testing.T) {
	repo := &fakeRepo{nextSeq: 1, nextAt: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)}
	cat := &fakeCatalog{products: []catalog.Product{newActiveProduct("1", "Widget", "GBP", 500)}}
	svc := order.NewService(repo, cat, testPageSizeMax)

	_, err := svc.Create(context.Background(), order.CreateOrder{
		CustomerID: 7, ActorID: "staff-1",
		Items: []order.CreateOrderItem{{ProductID: "1", Quantity: 2}},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if len(repo.insertedItems) != 1 {
		t.Fatalf("want 1 stored item, got %d", len(repo.insertedItems))
	}
	got := repo.insertedItems[0]
	if got.TitleSnapshot != "Widget" || got.UnitPriceMinor != 500 {
		t.Fatalf("line does not carry Catalog's title/price snapshot: %+v", got)
	}
}

// The discriminator: Catalog omits ids it cannot match, so pairing request
// line 2 (id "2") with Catalog's second returned product (id "3") by
// position would silently build an order for the wrong product.
func TestCreateRejectsAnIdCatalogDidNotReturn(t *testing.T) {
	repo := &fakeRepo{}
	cat := &fakeCatalog{products: []catalog.Product{
		newActiveProduct("1", "A", "GBP", 100),
		newActiveProduct("3", "C", "GBP", 300),
	}}
	svc := order.NewService(repo, cat, testPageSizeMax)

	_, err := svc.Create(context.Background(), order.CreateOrder{
		CustomerID: 1, ActorID: "staff-1",
		Items: []order.CreateOrderItem{
			{ProductID: "1", Quantity: 1},
			{ProductID: "2", Quantity: 1},
			{ProductID: "3", Quantity: 1},
		},
	})
	if !errors.Is(err, order.ErrProductUnavailable) {
		t.Fatalf("want ErrProductUnavailable, got %v", err)
	}
	if !strings.Contains(err.Error(), "2") {
		t.Fatalf("error must name the id Catalog did not return: %v", err)
	}
	if repo.runInTxCalls != 0 {
		t.Fatalf("no transaction should open when a line cannot be resolved, got %d", repo.runInTxCalls)
	}
}

func TestCreateRefusesMixedCurrencies(t *testing.T) {
	repo := &fakeRepo{}
	cat := &fakeCatalog{products: []catalog.Product{
		newActiveProduct("1", "A", "GBP", 100),
		newActiveProduct("2", "B", "EUR", 200),
	}}
	svc := order.NewService(repo, cat, testPageSizeMax)

	_, err := svc.Create(context.Background(), order.CreateOrder{
		CustomerID: 1, ActorID: "staff-1",
		Items: []order.CreateOrderItem{{ProductID: "1", Quantity: 1}, {ProductID: "2", Quantity: 1}},
	})
	if !errors.Is(err, order.ErrCurrencyMismatch) {
		t.Fatalf("want ErrCurrencyMismatch, got %v", err)
	}
	if repo.runInTxCalls != 0 {
		t.Fatalf("no transaction should open on a currency mismatch, got %d", repo.runInTxCalls)
	}
}

func TestTotalIsTheSumOfLineTotals(t *testing.T) {
	repo := &fakeRepo{nextSeq: 1, nextAt: time.Now()}
	cat := &fakeCatalog{products: []catalog.Product{
		newActiveProduct("1", "A", "GBP", 500),
		newActiveProduct("2", "B", "GBP", 250),
	}}
	svc := order.NewService(repo, cat, testPageSizeMax)

	_, err := svc.Create(context.Background(), order.CreateOrder{
		CustomerID: 1, ActorID: "staff-1",
		Items: []order.CreateOrderItem{{ProductID: "1", Quantity: 2}, {ProductID: "2", Quantity: 3}},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	const want = 2*500 + 3*250
	if repo.insertedOrder.TotalMinor != want {
		t.Fatalf("total: want %d, got %d", want, repo.insertedOrder.TotalMinor)
	}
}

func TestCreateOpensNoTransactionUntilCatalogHasAnswered(t *testing.T) {
	repo := &fakeRepo{}
	cat := &fakeCatalog{err: errors.New("catalog is unavailable")}
	svc := order.NewService(repo, cat, testPageSizeMax)

	_, err := svc.Create(context.Background(), order.CreateOrder{
		CustomerID: 1, ActorID: "staff-1",
		Items: []order.CreateOrderItem{{ProductID: "1", Quantity: 1}},
	})
	if err == nil {
		t.Fatal("want an error when Catalog cannot answer")
	}
	if repo.runInTxCalls != 0 {
		t.Fatalf("RunInTx must not run before Catalog answers, got %d calls", repo.runInTxCalls)
	}
}

func TestCreateRejectsAnInactiveProduct(t *testing.T) {
	repo := &fakeRepo{}
	cat := &fakeCatalog{products: []catalog.Product{
		{ID: "1", Title: "Discontinued", PriceMinor: 100, Currency: "GBP", Status: "archived"},
	}}
	svc := order.NewService(repo, cat, testPageSizeMax)

	_, err := svc.Create(context.Background(), order.CreateOrder{
		CustomerID: 1, ActorID: "staff-1",
		Items: []order.CreateOrderItem{{ProductID: "1", Quantity: 1}},
	})
	if !errors.Is(err, order.ErrProductUnavailable) {
		t.Fatalf("want ErrProductUnavailable for an archived product, got %v", err)
	}
	if repo.runInTxCalls != 0 {
		t.Fatalf("no transaction should open for an inactive product, got %d", repo.runInTxCalls)
	}
}

func TestCreateRejectsEmptyItems(t *testing.T) {
	repo := &fakeRepo{}
	cat := &fakeCatalog{}
	svc := order.NewService(repo, cat, testPageSizeMax)

	_, err := svc.Create(context.Background(), order.CreateOrder{CustomerID: 1, ActorID: "staff-1"})
	if err == nil {
		t.Fatal("want an error for an order with no items")
	}
	if cat.calls != 0 {
		t.Fatalf("Catalog must not be called for a shape-invalid request, got %d calls", cat.calls)
	}
}

// The first order_events row is the order coming into existence: from_status
// is nil, and the actor comes from CreateOrder.ActorID, never a line field.
func TestCreateRecordsTheFirstEventWithNoFromStatus(t *testing.T) {
	repo := &fakeRepo{nextSeq: 1, nextAt: time.Now()}
	cat := &fakeCatalog{products: []catalog.Product{newActiveProduct("1", "A", "GBP", 100)}}
	svc := order.NewService(repo, cat, testPageSizeMax)

	_, err := svc.Create(context.Background(), order.CreateOrder{
		CustomerID: 1, ActorID: "staff-42",
		Items: []order.CreateOrderItem{{ProductID: "1", Quantity: 1}},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if repo.insertedEvent.FromStatus != nil {
		t.Fatalf("want nil FromStatus on the first event, got %v", repo.insertedEvent.FromStatus)
	}
	if repo.insertedEvent.ActorID != "staff-42" {
		t.Fatalf("actor: want staff-42, got %q", repo.insertedEvent.ActorID)
	}
}

func TestGetWrapsRepositoryNotFound(t *testing.T) {
	repo := &fakeRepo{getByIDErr: order.ErrOrderNotFound}
	svc := order.NewService(repo, &fakeCatalog{}, testPageSizeMax)

	_, err := svc.Get(context.Background(), 99)
	if !errors.Is(err, order.ErrOrderNotFound) {
		t.Fatalf("want ErrOrderNotFound, got %v", err)
	}
}
