package order_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/loloDawit/go-admin/platform/httpx"
	"github.com/loloDawit/go-admin/platform/observability"
	"github.com/loloDawit/go-admin/platform/principal"
	"github.com/loloDawit/go-admin/services/orders/internal/catalog"
	"github.com/loloDawit/go-admin/services/orders/internal/httperr"
	"github.com/loloDawit/go-admin/services/orders/internal/order"
)

var failureTestKey = []byte("a-test-signing-key-at-least-32-bytes-long")

// newRealHandler wires the actual httperr.Writer, unlike the rest of this
// package's tests: §16's six failure behaviours are about the status code
// and body httperr decides, not just that Create returned an error.
func newRealHandler(repo order.Repository, catalogURL string) *order.Handler {
	client := catalog.NewClient(catalogURL, 30*time.Second, 4)
	svc := order.NewService(repo, client, testPageSizeMax)
	logger, _ := observability.NewCaptured()
	return order.NewHandler(svc, 1<<20, httperr.New(logger).Write)
}

func createRequest(customerID, productID string) *http.Request {
	body := `{"customerId":"` + customerID + `","items":[{"productId":"` + productID + `","quantity":1}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

// signedRequest runs the handler through principal.Middleware exactly as
// the router wires it, since Create's actor comes from the verified
// principal.
func signedRequest(t *testing.T, req *http.Request, h http.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	p := principal.Principal{StaffID: "staff-1", IssuedAt: time.Now(), ExpiresAt: time.Now().Add(time.Minute)}
	header, sig, err := principal.Sign(p, failureTestKey)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	req.Header.Set(principal.HeaderPrincipal, header)
	req.Header.Set(principal.HeaderSignature, sig)

	var captured error
	middleware := principal.Middleware(failureTestKey, func(w http.ResponseWriter, r *http.Request, err error) {
		captured = err
		w.WriteHeader(http.StatusUnauthorized)
	})(h)

	rec := httptest.NewRecorder()
	middleware.ServeHTTP(rec, req)
	if captured != nil {
		t.Fatalf("principal middleware rejected a validly signed principal: %v", captured)
	}
	return rec
}

func productResponse(id, title string, priceMinor int64, status string) map[string]any {
	return map[string]any{"id": id, "title": title, "priceMinor": priceMinor, "currency": "GBP", "status": status}
}

func TestCreateSucceedsWhenCatalogIsAvailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"products": []any{productResponse("1", "Widget", 500, "active")}})
	}))
	defer srv.Close()

	repo := &fakeRepo{nextSeq: 1, nextAt: time.Now()}
	h := newRealHandler(repo, srv.URL)

	rec := signedRequest(t, createRequest("1", "1"), h.Create)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status: want 201, got %d, body=%s", rec.Code, rec.Body.String())
	}
	if repo.runInTxCalls != 1 {
		t.Fatalf("want the order persisted, got %d transactions", repo.runInTxCalls)
	}
}

func TestCreateRefusesAMissingProductWith422(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"products": []any{}})
	}))
	defer srv.Close()

	repo := &fakeRepo{}
	h := newRealHandler(repo, srv.URL)

	rec := signedRequest(t, createRequest("1", "missing-product"), h.Create)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status: want 422, got %d, body=%s", rec.Code, rec.Body.String())
	}
	var body httpx.ErrorBody
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Code != "product_unavailable" {
		t.Fatalf("code: want product_unavailable, got %q", body.Code)
	}
	if !strings.Contains(body.Message, "missing-product") {
		t.Errorf("message must name the id, got %q", body.Message)
	}
	if repo.runInTxCalls != 0 {
		t.Fatalf("no order should be written, got %d transactions", repo.runInTxCalls)
	}
}

func TestCreateMapsA500FromCatalogTo502WithNoPartialOrder(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	repo := &fakeRepo{}
	h := newRealHandler(repo, srv.URL)

	rec := signedRequest(t, createRequest("1", "1"), h.Create)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status: want 502, got %d, body=%s", rec.Code, rec.Body.String())
	}
	var body httpx.ErrorBody
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Code != "catalog_unavailable" {
		t.Fatalf("code: want catalog_unavailable, got %q", body.Code)
	}
	if repo.runInTxCalls != 0 {
		t.Fatalf("a 502 with a row written is worse than a 500; got %d transactions", repo.runInTxCalls)
	}
}

func TestCreateMapsATimeoutTo504WithNoPartialOrder(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
	}))
	defer srv.Close()

	repo := &fakeRepo{}
	client := catalog.NewClient(srv.URL, 20*time.Millisecond, 4)
	svc := order.NewService(repo, client, testPageSizeMax)
	logger, _ := observability.NewCaptured()
	h := order.NewHandler(svc, 1<<20, httperr.New(logger).Write)

	rec := signedRequest(t, createRequest("1", "1"), h.Create)

	if rec.Code != http.StatusGatewayTimeout {
		t.Fatalf("status: want 504, got %d, body=%s", rec.Code, rec.Body.String())
	}
	var body httpx.ErrorBody
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Code != "catalog_timeout" {
		t.Fatalf("code: want catalog_timeout, got %q", body.Code)
	}
	if repo.runInTxCalls != 0 {
		t.Fatalf("no order should be written, got %d transactions", repo.runInTxCalls)
	}
}

func TestCreateMapsAnUnreachableCatalogTo502WithNoPartialOrder(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close() // closed before use: nothing is listening at url any more.

	repo := &fakeRepo{}
	h := newRealHandler(repo, url)

	rec := signedRequest(t, createRequest("1", "1"), h.Create)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status: want 502, got %d, body=%s", rec.Code, rec.Body.String())
	}
	var body httpx.ErrorBody
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Code != "catalog_unavailable" {
		t.Fatalf("code: want catalog_unavailable, got %q", body.Code)
	}
	if repo.runInTxCalls != 0 {
		t.Fatalf("no order should be written, got %d transactions", repo.runInTxCalls)
	}
}

// The stub must SEE the cancellation, not merely have the client return an
// error; the client's 30s deadline is far longer than this test's patience.
func TestCreateClientCancellationReachesTheStubAndWritesNothing(t *testing.T) {
	arrived := make(chan struct{})
	saw := make(chan error, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(arrived)
		// The server only starts watching for connection close once the
		// request body is drained.
		io.Copy(io.Discard, r.Body)
		select {
		case <-r.Context().Done():
			saw <- r.Context().Err()
		case <-time.After(5 * time.Second):
			saw <- nil
		}
	}))
	defer srv.Close()

	repo := &fakeRepo{}
	h := newRealHandler(repo, srv.URL)

	req := createRequest("1", "1")
	p := principal.Principal{StaffID: "staff-1", IssuedAt: time.Now(), ExpiresAt: time.Now().Add(time.Minute)}
	header, sig, err := principal.Sign(p, failureTestKey)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	req.Header.Set(principal.HeaderPrincipal, header)
	req.Header.Set(principal.HeaderSignature, sig)

	ctx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		principal.Middleware(failureTestKey, func(http.ResponseWriter, *http.Request, error) {})(http.HandlerFunc(h.Create)).ServeHTTP(rec, req)
		close(done)
	}()

	select {
	case <-arrived:
	case <-time.After(2 * time.Second):
		t.Fatal("the request never reached the stub")
	}
	cancel()

	select {
	case gotErr := <-saw:
		if gotErr == nil {
			t.Fatal("the stub's request context was never cancelled — a deadline elsewhere would let this pass wrongly")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the outgoing call was not cancelled when the caller cancelled")
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Create never returned after cancellation")
	}

	if rec.Code >= 500 {
		t.Fatalf("a cancelled caller must not be reported as a 5xx dependency failure, got %d", rec.Code)
	}
	if repo.runInTxCalls != 0 {
		t.Fatalf("no order should be written, got %d transactions", repo.runInTxCalls)
	}
}

// The actor on every order event comes from the verified principal, never a
// caller-supplied field: a caller-supplied actor is an audit log that lies.
func TestTheActorComesFromThePrincipalNotTheBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"products": []any{productResponse("1", "Widget", 500, "active")}})
	}))
	defer srv.Close()

	repo := &fakeRepo{nextSeq: 1, nextAt: time.Now()}
	h := newRealHandler(repo, srv.URL)

	body := `{"customerId":"1","items":[{"productId":"1","quantity":1}],"actorId":"someone-else"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := signedRequest(t, req, h.Create)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("a body-supplied actor field must be refused as an unknown field, got %d: %s", rec.Code, rec.Body.String())
	}
	if repo.runInTxCalls != 0 {
		t.Fatalf("no order should be written, got %d transactions", repo.runInTxCalls)
	}
}
