//go:build integration

package integration_test

import "testing"

// TestAnOrdersLinesSurviveTheProductChangingUnderneath needs POST
// /api/v1/orders through the gateway; router wiring is a later task's.
func TestAnOrdersLinesSurviveTheProductChangingUnderneath(t *testing.T) {
	t.Skip("unskip once orders routes are wired through the gateway (see order/handler_failure_test.go for the six §16 behaviours in the meantime)")
}
