package catalog_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/loloDawit/go-admin/platform/requestid"
	"github.com/loloDawit/go-admin/services/orders/internal/catalog"
	"github.com/loloDawit/go-admin/services/orders/internal/errs"
)

func TestResolveClassifiesAnUnreachableCatalog(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	client := catalog.NewClient(url, time.Second, 4)
	if _, err := client.Resolve(context.Background(), []string{"1"}); !errors.Is(err, errs.ErrCatalogUnavailable) {
		t.Fatalf("want ErrCatalogUnavailable, got %v", err)
	}
}

func TestResolveClassifiesATimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer srv.Close()

	client := catalog.NewClient(srv.URL, 10*time.Millisecond, 4)
	_, err := client.Resolve(context.Background(), []string{"1"})
	if !errors.Is(err, errs.ErrCatalogTimeout) {
		t.Fatalf("want ErrCatalogTimeout, got %v", err)
	}
	if errors.Is(err, errs.ErrCatalogUnavailable) {
		t.Fatal("a timeout must not also classify as ErrCatalogUnavailable")
	}
}

func TestResolveClassifiesA500(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := catalog.NewClient(srv.URL, time.Second, 4)
	if _, err := client.Resolve(context.Background(), []string{"1"}); !errors.Is(err, errs.ErrCatalogUnavailable) {
		t.Fatalf("want ErrCatalogUnavailable, got %v", err)
	}
}

func TestResolveClassifiesMalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	client := catalog.NewClient(srv.URL, time.Second, 4)
	if _, err := client.Resolve(context.Background(), []string{"1"}); !errors.Is(err, errs.ErrCatalogRejected) {
		t.Fatalf("want ErrCatalogRejected, got %v", err)
	}
}

func TestResolveForwardsTheRequestID(t *testing.T) {
	var seen string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.Header.Get(requestid.Header)
		json.NewEncoder(w).Encode(map[string]any{"products": []any{}})
	}))
	defer srv.Close()

	client := catalog.NewClient(srv.URL, time.Second, 4)

	var resolveErr error
	handler := requestid.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, resolveErr = client.Resolve(r.Context(), []string{"1"})
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(requestid.Header, "req-abc")
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if resolveErr != nil {
		t.Fatalf("want success, got %v", resolveErr)
	}
	if seen != "req-abc" {
		t.Errorf("X-Request-Id reaching the stub: want req-abc, got %q", seen)
	}
}

func TestResolveIsCancelledWhenTheCallerIsCancelled(t *testing.T) {
	serverSawCancel := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The server only starts watching for connection close once the request body is drained.
		io.Copy(io.Discard, r.Body)
		select {
		case <-r.Context().Done():
			close(serverSawCancel)
		case <-time.After(5 * time.Second):
		}
	}))
	defer srv.Close()

	// The client's own deadline is far longer than this test's patience, so a
	// closed connection can only mean the caller's cancellation travelled. With
	// a short deadline the client times out on its own and the server sees the
	// close either way, which passes whether or not cancellation works.
	client := catalog.NewClient(srv.URL, 30*time.Second, 4)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	_, err := client.Resolve(ctx, []string{"1"})
	if err == nil {
		t.Fatal("want an error when the caller cancels mid-flight")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want a cancellation, got %v — a deadline here would mean the client timed out rather than observing the caller", err)
	}

	select {
	case <-serverSawCancel:
	case <-time.After(2 * time.Second):
		t.Fatal("the outgoing call was not cancelled when the caller cancelled")
	}
}

func TestResolveOmitsIDsWithNoMatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"products": []map[string]any{
				{"id": "1", "title": "Mug", "priceMinor": 1200, "currency": "USD", "status": "archived"},
			},
		})
	}))
	defer srv.Close()

	client := catalog.NewClient(srv.URL, time.Second, 4)
	products, err := client.Resolve(context.Background(), []string{"1", "2"})
	if err != nil {
		t.Fatalf("want success, got %v", err)
	}
	if len(products) != 1 {
		t.Fatalf("want 1 product (id 2 unmatched), got %d", len(products))
	}
	if products[0].Status != "archived" {
		t.Errorf("an archived product must still be resolved, got status %q", products[0].Status)
	}
}
