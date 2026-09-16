// Package web serves the built single-page application from the gateway, so
// the app and the API share an origin and the session cookie needs no CORS
// handling or hardcoded API host.
package web

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// assetPrefix is the build's content-hashed output directory; those filenames
// change whenever their contents do, so they can be cached indefinitely.
const assetPrefix = "/assets/"

const indexFile = "index.html"

// Handler serves root, falling back to index.html for any path that is not a
// file on disk: the router owns those paths, and the server cannot know them.
func Handler(root string) (http.Handler, error) {
	index := filepath.Join(root, indexFile)
	if _, err := os.Stat(index); err != nil {
		return nil, err
	}

	files := http.FileServer(http.Dir(root))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		name := filepath.Join(root, filepath.Clean("/"+r.URL.Path))
		if info, err := os.Stat(name); err == nil && !info.IsDir() {
			if strings.HasPrefix(r.URL.Path, assetPrefix) {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}
			files.ServeHTTP(w, r)
			return
		}

		// The client must re-fetch this on every visit; it names the hashed
		// bundles, so a stale copy pins the app to a build that may be gone.
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFile(w, r, index)
	}), nil
}
