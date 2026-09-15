package order_test

import (
	"errors"
	"testing"

	"github.com/loloDawit/go-admin/services/orders/internal/order"
)

func TestStatusEndpointCannotReachCancelledOrRefunded(t *testing.T) {
	repo := &fakeRepo{status: order.StatusPending}
	svc := order.NewService(repo, &fakeCatalog{}, testPageSizeMax)

	if _, err := svc.SetStatus(t.Context(), 1, "staff-1", order.StatusCancelled); !errors.Is(err, order.ErrInvalidTransition) {
		t.Fatalf("SetStatus to cancelled: want ErrInvalidTransition, got %v", err)
	}
	if _, err := svc.SetStatus(t.Context(), 1, "staff-1", order.StatusRefunded); !errors.Is(err, order.ErrInvalidTransition) {
		t.Fatalf("SetStatus to refunded: want ErrInvalidTransition, got %v", err)
	}
	if repo.updateStatusCalls != 0 {
		t.Fatalf("no status write should happen, got %d", repo.updateStatusCalls)
	}
}

func TestStatusEndpointAllowsAnOrdinaryTransition(t *testing.T) {
	repo := &fakeRepo{status: order.StatusPending}
	svc := order.NewService(repo, &fakeCatalog{}, testPageSizeMax)

	if _, err := svc.SetStatus(t.Context(), 1, "staff-1", order.StatusPaid); err != nil {
		t.Fatalf("SetStatus to paid: %v", err)
	}
	if repo.updateStatusCalls != 1 {
		t.Fatalf("want 1 status write, got %d", repo.updateStatusCalls)
	}
}

func TestCancelIsRefusedOnceShipped(t *testing.T) {
	repo := &fakeRepo{status: order.StatusShipped}
	svc := order.NewService(repo, &fakeCatalog{}, testPageSizeMax)

	_, err := svc.Cancel(t.Context(), 1, "staff-1", "customer request")
	if !errors.Is(err, order.ErrInvalidTransition) {
		t.Fatalf("want ErrInvalidTransition, got %v", err)
	}
	if repo.updateStatusCalls != 0 {
		t.Fatalf("no status write should happen, got %d", repo.updateStatusCalls)
	}
}

func TestCancelSucceedsFromPendingPaidAndPacked(t *testing.T) {
	for _, from := range []order.Status{order.StatusPending, order.StatusPaid, order.StatusPacked} {
		repo := &fakeRepo{status: from}
		svc := order.NewService(repo, &fakeCatalog{}, testPageSizeMax)

		if _, err := svc.Cancel(t.Context(), 1, "staff-1", "reason"); err != nil {
			t.Fatalf("cancel from %s: %v", from, err)
		}
	}
}

// Refund's transition table only allows it from paid, packed, shipped or
// delivered; every one of those is reached by way of paid, so this asserts
// "was paid at some point" without a separate event-history check.
func TestRefundRequiresTheOrderToHaveBeenPaid(t *testing.T) {
	repo := &fakeRepo{status: order.StatusPending}
	svc := order.NewService(repo, &fakeCatalog{}, testPageSizeMax)

	_, err := svc.Refund(t.Context(), 1, "staff-1", "customer request")
	if !errors.Is(err, order.ErrInvalidTransition) {
		t.Fatalf("want ErrInvalidTransition, got %v", err)
	}
	if repo.updateStatusCalls != 0 {
		t.Fatalf("no status write should happen, got %d", repo.updateStatusCalls)
	}
}

func TestRefundSucceedsOnceThereHasBeenPayment(t *testing.T) {
	for _, from := range []order.Status{order.StatusPaid, order.StatusPacked, order.StatusShipped, order.StatusDelivered} {
		repo := &fakeRepo{status: from}
		svc := order.NewService(repo, &fakeCatalog{}, testPageSizeMax)

		if _, err := svc.Refund(t.Context(), 1, "staff-1", "reason"); err != nil {
			t.Fatalf("refund from %s: %v", from, err)
		}
	}
}

// The status column and the event log must move together: an event whose
// from_status disagrees with what was actually written is worse than none.
func TestEveryTransitionAppendsAnEvent(t *testing.T) {
	repo := &fakeRepo{status: order.StatusPending}
	svc := order.NewService(repo, &fakeCatalog{}, testPageSizeMax)

	if _, err := svc.SetStatus(t.Context(), 1, "staff-9", order.StatusPaid); err != nil {
		t.Fatalf("SetStatus: %v", err)
	}
	if repo.runInTxCalls != 1 {
		t.Fatalf("the status write and its event must share one transaction, got %d RunInTx calls", repo.runInTxCalls)
	}
	if repo.insertedEvent.FromStatus == nil || *repo.insertedEvent.FromStatus != order.StatusPending {
		t.Fatalf("event FromStatus: want pending, got %v", repo.insertedEvent.FromStatus)
	}
	if repo.insertedEvent.ToStatus != order.StatusPaid {
		t.Fatalf("event ToStatus: want paid, got %v", repo.insertedEvent.ToStatus)
	}
	if repo.insertedEvent.ActorID != "staff-9" {
		t.Fatalf("event ActorID: want staff-9, got %q", repo.insertedEvent.ActorID)
	}
}

func TestSetStatusRejectsATransitionTheTableDoesNotAllow(t *testing.T) {
	repo := &fakeRepo{status: order.StatusDelivered}
	svc := order.NewService(repo, &fakeCatalog{}, testPageSizeMax)

	_, err := svc.SetStatus(t.Context(), 1, "staff-1", order.StatusPending)
	if !errors.Is(err, order.ErrInvalidTransition) {
		t.Fatalf("want ErrInvalidTransition, got %v", err)
	}
}
