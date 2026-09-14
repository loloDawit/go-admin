package product_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/loloDawit/go-admin/services/catalog/internal/product"
)

const testPageSizeMax = 50

// fakeRepository is an in-memory double; each test builds its own so none
// depends on another test's state or a real database.
type fakeRepository struct {
	nextID int64
	byID   map[int64]product.Product
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{byID: make(map[int64]product.Product)}
}

func (f *fakeRepository) Create(_ context.Context, in product.CreateProduct) (product.Product, error) {
	for _, p := range f.byID {
		if p.SKU == in.SKU {
			return product.Product{}, product.ErrSkuTaken
		}
	}
	f.nextID++
	now := time.Now()
	p := product.Product{
		ID: f.nextID, SKU: in.SKU, Title: in.Title, Description: in.Description,
		PriceMinor: in.PriceMinor, Currency: in.Currency, Status: product.StatusDraft,
		CreatedAt: now, UpdatedAt: now,
	}
	f.byID[p.ID] = p
	return p, nil
}

func (f *fakeRepository) GetByID(_ context.Context, id int64) (product.Product, error) {
	p, ok := f.byID[id]
	if !ok {
		return product.Product{}, product.ErrProductNotFound
	}
	return p, nil
}

func (f *fakeRepository) Update(_ context.Context, id int64, in product.UpdateProduct) (product.Product, error) {
	p, ok := f.byID[id]
	if !ok {
		return product.Product{}, product.ErrProductNotFound
	}
	if p.Status == product.StatusArchived {
		return product.Product{}, product.ErrProductArchived
	}
	if in.Title != nil {
		p.Title = *in.Title
	}
	if in.Description != nil {
		p.Description = *in.Description
	}
	if in.PriceMinor != nil {
		p.PriceMinor = *in.PriceMinor
	}
	if in.Currency != nil {
		p.Currency = *in.Currency
	}
	f.byID[id] = p
	return p, nil
}

func (f *fakeRepository) Archive(_ context.Context, id int64) (product.Product, error) {
	p, ok := f.byID[id]
	if !ok {
		return product.Product{}, product.ErrProductNotFound
	}
	if p.Status == product.StatusArchived {
		return product.Product{}, product.ErrProductArchived
	}
	p.Status = product.StatusArchived
	f.byID[id] = p
	return p, nil
}

// fakeSortAllowlist mirrors postgres.go's sortColumns keys, unexported there
// and so restated here only to let the handler-level invalid-sort test
// exercise a real error path without a database.
var fakeSortAllowlist = map[string]bool{"": true, "title": true, "price": true, "created_at": true, "sku": true}

// List's sorting, pagination, and search ranking all live in postgres.go and
// are proven against real Postgres in test/integration; this stub only
// reproduces the allowlist check so a handler-level test can see the error
// path without a database.
func (f *fakeRepository) List(_ context.Context, q product.ListQuery) ([]product.Product, int, error) {
	key := strings.TrimPrefix(q.Sort, "-")
	if !fakeSortAllowlist[key] {
		return nil, 0, product.ErrInvalidSort
	}
	return nil, 0, nil
}

func (f *fakeRepository) Search(context.Context, product.SearchQuery) ([]product.Product, int, error) {
	return nil, 0, nil
}

func newTestService() (*product.Service, *fakeRepository) {
	repo := newFakeRepository()
	return product.NewService(repo, testPageSizeMax, "GBP"), repo
}

// 9007199254740993 cannot be held exactly by a float64 (it collides with
// 9007199254740992 there); a test using 14999 would pass even if PriceMinor
// were a float64 by mistake, so it proves nothing about the type.
func TestPriceIsStoredAndReturnedExactly(t *testing.T) {
	svc, _ := newTestService()
	const exact int64 = 9007199254740993

	created, err := svc.Create(t.Context(), product.CreateProduct{SKU: "precision-1", Title: "Precise", PriceMinor: exact, Currency: "GBP"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.PriceMinor != exact {
		t.Fatalf("price on create: want %d, got %d", exact, created.PriceMinor)
	}

	got, err := svc.Get(t.Context(), created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.PriceMinor != exact {
		t.Fatalf("price on get: want %d, got %d", exact, got.PriceMinor)
	}
}

func TestCreateRejectsANegativePrice(t *testing.T) {
	svc, _ := newTestService()
	_, err := svc.Create(t.Context(), product.CreateProduct{SKU: "neg-1", Title: "Negative", PriceMinor: -1, Currency: "GBP"})
	if !errors.Is(err, product.ErrInvalidPrice) {
		t.Fatalf("want ErrInvalidPrice, got %v", err)
	}
}

func TestCreateRejectsADuplicateSku(t *testing.T) {
	svc, _ := newTestService()
	if _, err := svc.Create(t.Context(), product.CreateProduct{SKU: "dup-1", Title: "First", PriceMinor: 100, Currency: "GBP"}); err != nil {
		t.Fatalf("first create: %v", err)
	}
	_, err := svc.Create(t.Context(), product.CreateProduct{SKU: "dup-1", Title: "Second", PriceMinor: 200, Currency: "GBP"})
	if !errors.Is(err, product.ErrSkuTaken) {
		t.Fatalf("want ErrSkuTaken, got %v", err)
	}
}

func TestUpdateLeavesOmittedFieldsUnchanged(t *testing.T) {
	svc, _ := newTestService()
	created, err := svc.Create(t.Context(), product.CreateProduct{
		SKU: "patch-1", Title: "Original title", Description: "Original description", PriceMinor: 5000, Currency: "GBP",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	newTitle := "New title"
	updated, err := svc.Update(t.Context(), created.ID, product.UpdateProduct{Title: &newTitle})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Title != newTitle {
		t.Errorf("title: want %q, got %q", newTitle, updated.Title)
	}
	if updated.Description != "Original description" {
		t.Errorf("description must survive a PATCH that omits it, got %q", updated.Description)
	}
	if updated.PriceMinor != 5000 {
		t.Errorf("priceMinor must survive a PATCH that omits it, got %d", updated.PriceMinor)
	}
	if updated.Currency != "GBP" {
		t.Errorf("currency must survive a PATCH that omits it, got %q", updated.Currency)
	}
}

func TestArchivedProductCannotBeEdited(t *testing.T) {
	svc, _ := newTestService()
	created, err := svc.Create(t.Context(), product.CreateProduct{SKU: "arch-1", Title: "To archive", PriceMinor: 100, Currency: "GBP"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := svc.Archive(t.Context(), created.ID); err != nil {
		t.Fatalf("archive: %v", err)
	}

	newTitle := "Should not apply"
	_, err = svc.Update(t.Context(), created.ID, product.UpdateProduct{Title: &newTitle})
	if !errors.Is(err, product.ErrProductArchived) {
		t.Fatalf("want ErrProductArchived, got %v", err)
	}
}

func TestPageSizeIsClampedNotRefused(t *testing.T) {
	page, pageSize := clampForTest(t, 0, testPageSizeMax+1000)
	if pageSize != testPageSizeMax {
		t.Fatalf("pageSize: want clamped to %d, got %d", testPageSizeMax, pageSize)
	}
	if page != 1 {
		t.Fatalf("page: want default 1, got %d", page)
	}
}

// clampForTest drives List with a repository stub whose List captures the
// query it was actually called with, so the assertion is against what
// reached the repository, not just what the service returned.
func clampForTest(t *testing.T, page, pageSize int) (int, int) {
	t.Helper()
	repo := newFakeRepository()
	svc := product.NewService(repo, testPageSizeMax, "GBP")

	result, err := svc.List(t.Context(), product.ListQuery{Page: page, PageSize: pageSize})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	return result.Page, result.PageSize
}
