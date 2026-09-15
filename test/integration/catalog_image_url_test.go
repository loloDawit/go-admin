//go:build integration

package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"testing"
	"time"
)

// A presigned URL names a host the browser must reach. The service reaches the
// object store over the compose network, which no client can resolve, so the
// two endpoints differ and only fetching the URL from outside proves it.
func TestAPresignedImageURLIsFetchableFromOutsideTheNetwork(t *testing.T) {
	c := loggedInClient(t)
	created := createProduct(t, c, uniqueSKU(t, "img"), "Photographed", "", 1000)

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreatePart(map[string][]string{
		"Content-Disposition": {`form-data; name="file"; filename="photo.png"`},
		"Content-Type":        {"image/png"},
	})
	if err != nil {
		t.Fatalf("multipart: %v", err)
	}
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	img.Set(1, 1, color.RGBA{200, 100, 50, 255})
	if err := png.Encode(part, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	w.Close()

	req, err := http.NewRequest(http.MethodPost, gatewayURL()+"/api/v1/products/"+created.ID+"/images", &buf)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("upload: want 201, got %d (%s)", resp.StatusCode, body)
	}

	var uploaded struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&uploaded); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if uploaded.URL == "" {
		t.Fatal("no presigned url returned")
	}

	fetched, err := (&http.Client{Timeout: 10 * time.Second}).Get(uploaded.URL)
	if err != nil {
		t.Fatalf("the presigned url is unreachable from a client: %v", err)
	}
	defer fetched.Body.Close()
	if fetched.StatusCode != http.StatusOK {
		t.Fatalf("presigned url %q: want 200, got %d", hostOf(uploaded.URL), fetched.StatusCode)
	}
	if n, _ := io.Copy(io.Discard, fetched.Body); n == 0 {
		t.Fatal("presigned url returned an empty body")
	}
}

func hostOf(rawURL string) string {
	parts := bytes.SplitN([]byte(rawURL), []byte("/"), 4)
	if len(parts) < 3 {
		return rawURL
	}
	return fmt.Sprintf("%s", parts[2])
}
