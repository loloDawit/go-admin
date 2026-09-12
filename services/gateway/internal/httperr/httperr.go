// Package httperr owns the gateway's client-facing errors. Transport failures
// name upstream hosts; nothing from them reaches a response body.
package httperr

import (
	"net/http"

	"github.com/loloDawit/go-admin/platform/httpx"
)

func WriteUnknownRoute(w http.ResponseWriter) {
	httpx.WriteError(w, http.StatusNotFound, "not_found", "no such route")
}

func WriteUpstreamUnavailable(w http.ResponseWriter) {
	httpx.WriteError(w, http.StatusBadGateway, "upstream_unavailable", "the service is temporarily unavailable")
}
