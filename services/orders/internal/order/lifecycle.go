package order

// transitions is the one place a status change is permitted; any pair
// absent here is refused. cancelled and refunded never appear as a
// from-status: once an order reaches either, it does not move again.
var transitions = map[Status]map[Status]bool{
	StatusPending:   {StatusPaid: true, StatusCancelled: true},
	StatusPaid:      {StatusPacked: true, StatusCancelled: true, StatusRefunded: true},
	StatusPacked:    {StatusShipped: true, StatusCancelled: true, StatusRefunded: true},
	StatusShipped:   {StatusDelivered: true, StatusRefunded: true},
	StatusDelivered: {StatusRefunded: true},
	StatusCancelled: {},
	StatusRefunded:  {},
}

// allStatuses enumerates every status this service knows, for tests that
// must walk the full transition matrix rather than a handful of examples.
var allStatuses = []Status{
	StatusPending, StatusPaid, StatusPacked, StatusShipped,
	StatusDelivered, StatusCancelled, StatusRefunded,
}

// CanTransition reports whether to is reachable from's current status. Every
// path to paid, packed, shipped or delivered passes through paid, so a
// refund from any of those needs no separate "was ever paid" check.
func CanTransition(from, to Status) bool {
	return transitions[from][to]
}
