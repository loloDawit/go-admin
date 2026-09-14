package principal

import (
	"context"
	"net/http"
	"time"
)

const (
	HeaderPrincipal = "X-Principal"
	HeaderSignature = "X-Principal-Signature"
)

type ctxKey struct{}

// Middleware never decides a status code itself on a rejection; that mapping belongs to the caller's HTTP boundary.
func Middleware(key []byte, onErr func(http.ResponseWriter, *http.Request, error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, err := Verify(r.Header.Get(HeaderPrincipal), r.Header.Get(HeaderSignature), key, time.Now())
			if err != nil {
				onErr(w, r, err)
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, p)))
		})
	}
}

func FromContext(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(ctxKey{}).(Principal)
	return p, ok
}
