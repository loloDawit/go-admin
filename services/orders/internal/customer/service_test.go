package customer_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/loloDawit/go-admin/services/orders/internal/customer"
)

const testPageSizeMax = 50

// fakeRepository is an in-memory double; each test builds its own so none
// depends on another test's state or a real database.
type fakeRepository struct {
	lifetimeValue    int64
	lifetimeCurrency string
	lifetimeErr      error
	nextID           int64
	byID             map[int64]customer.Customer
	ordersByCustomer map[int64][]customer.OrderSummary
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{byID: make(map[int64]customer.Customer)}
}

// Create mirrors citext's fold: a duplicate differing only in case is still a duplicate.
func (f *fakeRepository) Create(_ context.Context, in customer.CreateCustomer) (customer.Customer, error) {
	for _, c := range f.byID {
		if strings.EqualFold(c.Email, in.Email) {
			return customer.Customer{}, customer.ErrCustomerEmailTaken
		}
	}
	f.nextID++
	now := time.Now()
	c := customer.Customer{ID: f.nextID, Email: in.Email, Name: in.Name, CreatedAt: now, UpdatedAt: now}
	f.byID[c.ID] = c
	return c, nil
}

func (f *fakeRepository) GetByID(_ context.Context, id int64) (customer.Customer, error) {
	c, ok := f.byID[id]
	if !ok {
		return customer.Customer{}, customer.ErrCustomerNotFound
	}
	return c, nil
}

// GetByEmail mirrors citext's fold, same as Create.
func (f *fakeRepository) GetByEmail(_ context.Context, email string) (customer.Customer, error) {
	for _, c := range f.byID {
		if strings.EqualFold(c.Email, email) {
			return c, nil
		}
	}
	return customer.Customer{}, customer.ErrCustomerNotFound
}

func (f *fakeRepository) List(_ context.Context, q customer.ListQuery) ([]customer.Customer, int, error) {
	return nil, len(f.byID), nil
}

func (f *fakeRepository) LifetimeValueMinor(_ context.Context, id int64) (int64, string, error) {
	return f.lifetimeValue, f.lifetimeCurrency, f.lifetimeErr
}

// OrderHistory filters ordersByCustomer by the customerID argument: a fake
// that ignored it would let TestCustomerOrderHistoryIsScopedToThatCustomer
// pass wrongly, the same failure mode a WHERE-less real query would have.
func (f *fakeRepository) OrderHistory(_ context.Context, customerID int64, q customer.OrderHistoryQuery) ([]customer.OrderSummary, int, error) {
	items := f.ordersByCustomer[customerID]
	return items, len(items), nil
}

func newTestService() (*customer.Service, *fakeRepository) {
	repo := newFakeRepository()
	return customer.NewService(repo, testPageSizeMax), repo
}

func TestCreateRejectsAnEmptyEmail(t *testing.T) {
	svc, _ := newTestService()
	_, err := svc.Create(t.Context(), customer.CreateCustomer{Email: "", Name: "No Email"})
	if !errors.Is(err, customer.ErrInvalidCustomerEmail) {
		t.Fatalf("want ErrInvalidCustomerEmail, got %v", err)
	}
}

// citext: "A@b.com" collides with "a@b.com" — a duplicate differing only in
// case must be refused, the same as an exact duplicate.
func TestEmailIsCaseInsensitivelyUnique(t *testing.T) {
	svc, _ := newTestService()
	if _, err := svc.Create(t.Context(), customer.CreateCustomer{Email: "A@b.com", Name: "First"}); err != nil {
		t.Fatalf("first create: %v", err)
	}
	_, err := svc.Create(t.Context(), customer.CreateCustomer{Email: "a@b.com", Name: "Second"})
	if !errors.Is(err, customer.ErrCustomerEmailTaken) {
		t.Fatalf("want ErrCustomerEmailTaken, got %v", err)
	}
}

// The reason citext is there: a lookup must find a customer regardless of
// the case the caller typed the email in.
func TestLookupByEmailIsCaseInsensitive(t *testing.T) {
	svc, _ := newTestService()
	created, err := svc.Create(t.Context(), customer.CreateCustomer{Email: "Case@Example.com", Name: "Someone"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.Email != "Case@Example.com" {
		t.Fatalf("want the stored email echoed back, got %q", created.Email)
	}

	got, err := svc.ByEmail(t.Context(), "case@example.com")
	if err != nil {
		t.Fatalf("want a match on differing case, got %v", err)
	}
	if got.ID != created.ID || got.Email != created.Email {
		t.Fatalf("want the same customer, got id=%d email=%q vs id=%d email=%q", got.ID, got.Email, created.ID, created.Email)
	}
}

func TestByEmailReturnsNotFoundForAnUnknownEmail(t *testing.T) {
	svc, _ := newTestService()
	_, err := svc.ByEmail(t.Context(), "nobody@example.com")
	if !errors.Is(err, customer.ErrCustomerNotFound) {
		t.Fatalf("want ErrCustomerNotFound, got %v", err)
	}
}

func TestGetReturnsNotFoundForAnUnknownID(t *testing.T) {
	svc, _ := newTestService()
	_, err := svc.Get(t.Context(), 999999)
	if !errors.Is(err, customer.ErrCustomerNotFound) {
		t.Fatalf("want ErrCustomerNotFound, got %v", err)
	}
}

func TestPageSizeIsClampedNotRefused(t *testing.T) {
	svc, _ := newTestService()
	result, err := svc.List(t.Context(), customer.ListQuery{Page: 0, PageSize: testPageSizeMax + 1000})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if result.PageSize != testPageSizeMax {
		t.Fatalf("pageSize: want clamped to %d, got %d", testPageSizeMax, result.PageSize)
	}
	if result.Page != 1 {
		t.Fatalf("page: want default 1, got %d", result.Page)
	}
}

// A customer whose orders span two currencies has no single lifetime value, so
// the repository refuses rather than returning a sum across them.
func TestLifetimeValueRefusesAMixedCurrencyHistory(t *testing.T) {
	svc, repo := newTestService()
	repo.lifetimeErr = customer.ErrMixedCurrencyHistory

	if _, _, err := svc.LifetimeValue(context.Background(), 1); !errors.Is(err, customer.ErrMixedCurrencyHistory) {
		t.Fatalf("want ErrMixedCurrencyHistory, got %v", err)
	}
}

func TestLifetimeValueCarriesTheCurrencyItWasTakenIn(t *testing.T) {
	svc, repo := newTestService()
	repo.lifetimeValue, repo.lifetimeCurrency = 14999, "USD"

	total, currency, err := svc.LifetimeValue(context.Background(), 1)
	if err != nil {
		t.Fatalf("lifetime value: %v", err)
	}
	if total != 14999 || currency != "USD" {
		t.Fatalf("want 14999 USD, got %d %q", total, currency)
	}
}

// Asserted with two customers, not one: a query missing its WHERE would
// still pass this test with only customer A in the fixture.
func TestCustomerOrderHistoryIsScopedToThatCustomer(t *testing.T) {
	svc, repo := newTestService()
	repo.ordersByCustomer = map[int64][]customer.OrderSummary{
		1: {{ID: 100, Number: "ORD-A", Status: "paid", TotalMinor: 500, Currency: "GBP"}},
		2: {{ID: 200, Number: "ORD-B", Status: "paid", TotalMinor: 700, Currency: "GBP"}},
	}

	page, err := svc.OrderHistory(context.Background(), 1, customer.OrderHistoryQuery{})
	if err != nil {
		t.Fatalf("order history: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].Number != "ORD-A" {
		t.Fatalf("want only customer 1's order, got %+v", page.Items)
	}
	for _, o := range page.Items {
		if o.Number == "ORD-B" {
			t.Fatalf("customer 2's order leaked into customer 1's history: %+v", page.Items)
		}
	}
}

func TestOrderHistoryPageSizeIsClampedNotRefused(t *testing.T) {
	svc, repo := newTestService()
	repo.ordersByCustomer = map[int64][]customer.OrderSummary{1: {}}

	page, err := svc.OrderHistory(context.Background(), 1, customer.OrderHistoryQuery{PageSize: testPageSizeMax + 100})
	if err != nil {
		t.Fatalf("order history: %v", err)
	}
	if page.PageSize != testPageSizeMax {
		t.Fatalf("PageSize: want clamped to %d, got %d", testPageSizeMax, page.PageSize)
	}
}
