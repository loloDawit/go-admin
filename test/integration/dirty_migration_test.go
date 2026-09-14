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

// dirtyTarget's hostURL reaches /healthz, /readyz and /_platform directly,
// bypassing the gateway, which does not expose /readyz per service.
type dirtyTarget struct {
	service     string
	user        string
	password    string
	database    string
	hostURL     string
	hasPlatform bool // false for identity: it has no /_platform route
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

// setDirty opens and closes its own connection so it is safe to call again
// from t.Cleanup even after the test body has failed and unwound.
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

// TestDirtyMigrationFailsReadiness proves spec §12: a half-applied migration must fail readiness.
// t.Cleanup restores the clean state even if an assertion above it fails, so a failure never leaves the cluster dirty.
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
