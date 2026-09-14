// Package healthcheck lets a distroless service binary check itself: these images carry no shell, curl, or wget for Docker's HEALTHCHECK to invoke.
package healthcheck

import (
	"net/http"
	"time"
)

// Run never touches the database: /healthz is liveness, not readiness, so this stays accurate before the DB pool is reachable.
func Run(port string) int {
	client := &http.Client{Timeout: 2 * time.Second}

	resp, err := client.Get("http://127.0.0.1:" + port + "/healthz")
	if err != nil {
		return 1
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}
