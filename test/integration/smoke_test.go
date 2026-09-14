//go:build integration

package integration_test

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func gatewayURL() string {
	if v := os.Getenv("GATEWAY_URL"); v != "" {
		return v
	}
	return "http://localhost:8080"
}

type platformResponse struct {
	Service       string `json:"service"`
	SchemaVersion int    `json:"schemaVersion"`
	RequestID     string `json:"requestId"`
}

func TestWalkingSkeletonThroughTheGateway(t *testing.T) {
	client := &http.Client{Timeout: 10 * time.Second}

	for _, service := range []string{"catalog", "orders"} {
		t.Run(service, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, gatewayURL()+"/_platform/"+service, nil)
			if err != nil {
				t.Fatalf("build request: %v", err)
			}
			req.Header.Set("X-Request-Id", "smoke-"+service)

			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Fatalf("status: want 200, got %d", resp.StatusCode)
			}

			var body platformResponse
			if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
				t.Fatalf("decode: %v", err)
			}

			if body.Service != service {
				t.Errorf("service: want %q, got %q — the gateway routed to the wrong upstream", service, body.Service)
			}
			if body.SchemaVersion == 0 {
				t.Error("schemaVersion is 0 — migrations did not run against this service's database")
			}
			if body.RequestID != "smoke-"+service {
				t.Errorf("requestId: want %q, got %q — the ID did not survive the hop", "smoke-"+service, body.RequestID)
			}
			if echoed := resp.Header.Get("X-Request-Id"); echoed != "smoke-"+service {
				t.Errorf("response header: want %q, got %q", "smoke-"+service, echoed)
			}
		})
	}
}

func TestUnknownServiceReturnsTheStandardEnvelope(t *testing.T) {
	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get(gatewayURL() + "/_platform/nosuchservice")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status: want 404, got %d", resp.StatusCode)
	}

	var body struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Code == "" || body.Message == "" {
		t.Errorf("want the standard envelope, got %+v", body)
	}
}

// A rejected login still proves the path: through the gateway, into identity, to its own database.
func TestIdentityAnswersThroughTheGatewayFromItsOwnDatabase(t *testing.T) {
	client := &http.Client{Timeout: 10 * time.Second}

	body := strings.NewReader(`{"email":"nobody@example.com","password":"wrong"}`)
	req, err := http.NewRequest(http.MethodPost, gatewayURL()+"/api/v1/login", body)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-Id", "smoke-identity")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status: want 401, got %d", resp.StatusCode)
	}
	if got := resp.Header.Get("X-Request-Id"); got != "smoke-identity" {
		t.Errorf("request id: want smoke-identity, got %q", got)
	}

	var envelope struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if envelope.Code != "invalid_credentials" {
		t.Errorf("code: want invalid_credentials, got %q", envelope.Code)
	}
}
