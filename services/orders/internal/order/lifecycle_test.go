package order

import "testing"

// TestEveryTransitionIsExplicit walks every ordered pair of statuses against
// a matrix written independently of transitions: deriving "want" from the
// map under test would make this tautological, which is exactly how an
// unintended edge would survive.
func TestEveryTransitionIsExplicit(t *testing.T) {
	want := map[Status]map[Status]bool{
		StatusPending: {
			StatusPending: false, StatusPaid: true, StatusPacked: false, StatusShipped: false,
			StatusDelivered: false, StatusCancelled: true, StatusRefunded: false,
		},
		StatusPaid: {
			StatusPending: false, StatusPaid: false, StatusPacked: true, StatusShipped: false,
			StatusDelivered: false, StatusCancelled: true, StatusRefunded: true,
		},
		StatusPacked: {
			StatusPending: false, StatusPaid: false, StatusPacked: false, StatusShipped: true,
			StatusDelivered: false, StatusCancelled: true, StatusRefunded: true,
		},
		StatusShipped: {
			StatusPending: false, StatusPaid: false, StatusPacked: false, StatusShipped: false,
			StatusDelivered: true, StatusCancelled: false, StatusRefunded: true,
		},
		StatusDelivered: {
			StatusPending: false, StatusPaid: false, StatusPacked: false, StatusShipped: false,
			StatusDelivered: false, StatusCancelled: false, StatusRefunded: true,
		},
		StatusCancelled: {
			StatusPending: false, StatusPaid: false, StatusPacked: false, StatusShipped: false,
			StatusDelivered: false, StatusCancelled: false, StatusRefunded: false,
		},
		StatusRefunded: {
			StatusPending: false, StatusPaid: false, StatusPacked: false, StatusShipped: false,
			StatusDelivered: false, StatusCancelled: false, StatusRefunded: false,
		},
	}

	for _, from := range allStatuses {
		for _, to := range allStatuses {
			got := CanTransition(from, to)
			if got != want[from][to] {
				t.Errorf("CanTransition(%s, %s): got %v, want %v", from, to, got, want[from][to])
			}
		}
	}
}
