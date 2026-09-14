//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// dirtyTarget names one service's database role and the host-mapped ports at
// which its own /healthz, /readyz and /_platform can be reached directly
// (bypassing the gateway, which does not expose /readyz per service).
type dirtyTarget struct {
	service  string
	user     string
	password string
	database string
	hostURL  string
	// hasPlatform is false for identity: its walking skeleton was retired once
	// it gained real routes. /readyz carries the same assertion.
	hasPlatform bool
}

func dirtyTargets() []dirtyTarget {
	return []dirtyTarget{
		{"identity", "identity_user", "dev_only_identity", "identity_db", hostURL("IDENTITY_HOST_URL", "http://localhost:8081"), false},
		{"catalog", "catalog_user", "dev_only_catalog", "catalog_db", hostURL("CATALOG_HOST_URL", "http://localhost:8082"), true},
		{"orders", "orders_user", "dev_only_orders", "orders_db", hostURL("ORDERS_HOST_URL", "http://localhost:8083"), true},
	}
}

func hostURL(env, fallback string) string {
	if v := os.Getenv(env); v != "" {
		return v
	}
	return fallback
}

// setDirty opens its own short-lived connection and closes it before
// returning. It is deliberately self-contained (never a connection held open
// across a test's assertions) so it is safe to call again from t.Cleanup even
// after the test body has failed and unwound: there is no shared, possibly
// already-closed connection to reuse.
func setDirty(t *testing.T, tgt dirtyTarget, dirty bool) {
	t.Helper()
	ctx := context.Background()

	conn, err := pgx.Connect(ctx, dsn(t, tgt.user, tgt.password, tgt.database))
	if err != nil {
		t.Fatalf("connect to %s: %v", tgt.database, err)
	}
	defer conn.Close(ctx)

	if _, err := conn.Exec(ctx, "UPDATE schema_migrations SET dirty = $1", dirty); err != nil {
		t.Fatalf("set dirty=%v on %s: %v", dirty, tgt.database, err)
	}
}

type envelope struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func getEnvelope(t *testing.T, client *http.Client, url string, wantStatus int) envelope {
	t.Helper()
	resp, err := client.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != wantStatus {
		t.Errorf("GET %s: status: want %d, got %d", url, wantStatus, resp.StatusCode)
	}

	if wantStatus == http.StatusOK {
		return envelope{}
	}

	var body envelope
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("GET %s: decode body: %v", url, err)
	}
	return body
}

// TestDirtyMigrationFailsReadiness proves spec §12: a half-applied migration
// (schema_migrations.dirty = true) must fail readiness, not report the
// service healthy. It marks one service's database dirty, asserts both
// /readyz (direct) and /_platform (direct and via the gateway) report 503
// with the standard envelope while /healthz stays 200, then restores the
// clean state and asserts recovery.
//
// t.Cleanup guarantees the restore runs even if an assertion above it fails
// and aborts the test body early — a failure here must never leave the
// cluster dirty for every later test or developer.
func TestDirtyMigrationFailsReadiness(t *testing.T) {
	client := &http.Client{Timeout: 10 * time.Second}

	for _, tgt := range dirtyTargets() {
		tgt := tgt
		t.Run(tgt.service, func(t *testing.T) {
			t.Cleanup(func() { setDirty(t, tgt, false) })

			setDirty(t, tgt, true)

			if body := getEnvelope(t, client, tgt.hostURL+"/readyz", http.StatusServiceUnavailable); body.Code != "schema_dirty" {
				t.Errorf("/readyz code: want schema_dirty, got %q", body.Code)
			}
			if tgt.hasPlatform {
				if body := getEnvelope(t, client, tgt.hostURL+"/_platform", http.StatusServiceUnavailable); body.Code != "schema_dirty" {
					t.Errorf("/_platform code: want schema_dirty, got %q", body.Code)
				}
				getEnvelope(t, client, gatewayURL()+"/_platform/"+tgt.service, http.StatusServiceUnavailable)
			}
			getEnvelope(t, client, tgt.hostURL+"/healthz", http.StatusOK)

			setDirty(t, tgt, false)

			getEnvelope(t, client, tgt.hostURL+"/readyz", http.StatusOK)
			if tgt.hasPlatform {
				getEnvelope(t, client, tgt.hostURL+"/_platform", http.StatusOK)
				getEnvelope(t, client, gatewayURL()+"/_platform/"+tgt.service, http.StatusOK)
			}
		})
	}
}
