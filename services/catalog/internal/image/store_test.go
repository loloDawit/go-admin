//go:build integration

// This suite needs a running MinIO; it has no place in the unit suite, which
// runs untagged and has none.
package image_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/loloDawit/go-admin/services/catalog/internal/image"
)

func testConfig(t *testing.T, ttl time.Duration) image.Config {
	t.Helper()
	return image.Config{
		Endpoint:  envOr("S3_TEST_ENDPOINT", "127.0.0.1:9000"),
		Bucket:    envOr("S3_TEST_BUCKET", "catalog-images"),
		AccessKey: envOr("S3_TEST_ACCESS_KEY", "dev_only_minio"),
		SecretKey: envOr("S3_TEST_SECRET_KEY", "dev_only_minio_secret"),
		UseSSL:    false,
		URLTTL:    ttl,
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func newStore(t *testing.T, ttl time.Duration) *image.MinIOStore {
	t.Helper()
	store, err := image.New(testConfig(t, ttl))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	return store
}

func TestPutPresignFetchDelete(t *testing.T) {
	store := newStore(t, time.Minute)
	ctx := t.Context()
	key := "task-8-put-fetch-delete.txt"
	body := []byte("catalog image store round trip")

	if err := store.Put(ctx, key, bytes.NewReader(body), int64(len(body)), "text/plain"); err != nil {
		t.Fatalf("put: %v", err)
	}

	url, err := store.Presign(ctx, key)
	if err != nil {
		t.Fatalf("presign: %v", err)
	}

	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("fetch presigned url: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("fetch: want 200, got %d", resp.StatusCode)
	}
	got, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if !bytes.Equal(got, body) {
		t.Fatalf("body: want %q, got %q", body, got)
	}

	if err := store.Delete(ctx, key); err != nil {
		t.Fatalf("delete: %v", err)
	}

	confirmURL, err := store.Presign(ctx, key)
	if err != nil {
		t.Fatalf("presign after delete: %v", err)
	}
	confirmResp, err := http.Get(confirmURL)
	if err != nil {
		t.Fatalf("fetch after delete: %v", err)
	}
	defer confirmResp.Body.Close()
	if confirmResp.StatusCode == http.StatusOK {
		t.Fatal("object fetched successfully after Delete — it was not actually removed")
	}
}

// An untested expiry is a guess: this proves MinIO itself refuses the URL
// once the TTL elapses, not merely that Presign returned some string.
func TestPresignedURLIsRefusedAfterItExpires(t *testing.T) {
	store := newStore(t, time.Second)
	ctx := t.Context()
	key := "task-8-expiry.txt"
	body := []byte("expires soon")

	if err := store.Put(ctx, key, bytes.NewReader(body), int64(len(body)), "text/plain"); err != nil {
		t.Fatalf("put: %v", err)
	}
	t.Cleanup(func() { _ = store.Delete(context.Background(), key) })

	url, err := store.Presign(ctx, key)
	if err != nil {
		t.Fatalf("presign: %v", err)
	}

	fresh, err := http.Get(url)
	if err != nil {
		t.Fatalf("fetch before expiry: %v", err)
	}
	fresh.Body.Close()
	if fresh.StatusCode != http.StatusOK {
		t.Fatalf("fetch before expiry: want 200, got %d", fresh.StatusCode)
	}

	time.Sleep(2 * time.Second)

	stale, err := http.Get(url)
	if err != nil {
		t.Fatalf("fetch after expiry: %v", err)
	}
	defer stale.Body.Close()
	if stale.StatusCode == http.StatusOK {
		t.Fatal("presigned URL still served the object after its TTL elapsed")
	}
}
