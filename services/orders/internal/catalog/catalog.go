// Package catalog is Orders' client for Catalog's internal resolve endpoint —
// the first service-to-service call on the request path.
package catalog

import (
	"net/http"
	"time"
)

// Client calls Catalog's internal, gateway-network-only endpoints.
type Client struct {
	httpClient *http.Client
	baseURL    string
	timeout    time.Duration
}

// NewClient bounds the connection pool at maxIdleConns: an unbounded pool
// turns a slow Catalog into an Orders outage by exhausting file descriptors.
func NewClient(baseURL string, timeout time.Duration, maxIdleConns int) *Client {
	return &Client{
		httpClient: &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:        maxIdleConns,
				MaxIdleConnsPerHost: maxIdleConns,
				MaxConnsPerHost:     maxIdleConns,
			},
		},
		baseURL: baseURL,
		timeout: timeout,
	}
}
