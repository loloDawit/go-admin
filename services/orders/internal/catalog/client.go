package catalog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/loloDawit/go-admin/platform/requestid"
	"github.com/loloDawit/go-admin/services/orders/internal/errs"
)

// Resolve has no retry: a struggling Catalog would only be made worse by a retry storm.
func (c *Client) Resolve(ctx context.Context, ids []string) ([]Product, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	body, err := json.Marshal(resolveRequest{IDs: ids})
	if err != nil {
		return nil, errs.Wrap(errs.OpResolveProducts, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/internal/products/resolve", bytes.NewReader(body))
	if err != nil {
		return nil, errs.Wrap(errs.OpResolveProducts, err)
	}
	req.Header.Set("Content-Type", "application/json")
	if id := requestid.FromContext(ctx); id != "" {
		req.Header.Set(requestid.Header, id)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// context.DeadlineExceeded is how a slow Catalog is told apart from an unreachable one.
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, errs.Wrap(errs.OpResolveProducts, errs.ErrCatalogTimeout)
		}
		return nil, errs.Wrap(errs.OpResolveProducts, errs.ErrCatalogUnavailable)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusInternalServerError {
		return nil, errs.Wrap(errs.OpResolveProducts, errs.ErrCatalogUnavailable)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, errs.Wrap(errs.OpResolveProducts, errs.ErrCatalogRejected)
	}

	var out resolveResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, errs.Wrap(errs.OpResolveProducts, errs.ErrCatalogRejected)
	}
	return out.toProducts(), nil
}
