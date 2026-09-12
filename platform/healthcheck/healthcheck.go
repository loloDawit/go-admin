// Package healthcheck lets a distroless service binary check itself. These
// images carry no shell, curl, or wget, so Docker's HEALTHCHECK must invoke
// the binary itself; this package is what that invocation runs before the
// normal server start path.
package healthcheck

import (
	"net/http"
	"time"
)

// Run issues a GET to this process's own /healthz on the given port and
// returns a process exit code: 0 if it answered 200, 1 otherwise (including
// any transport failure, such as nothing listening yet). It never touches the
// database — /healthz is liveness, not readiness — so this stays accurate
// during the window before the DB pool is reachable.
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
