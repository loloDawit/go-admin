package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// ErrMalformedBody is DecodeJSON's one sentinel for every decode failure; the caller's httperr package maps it to a status code.
var ErrMalformedBody = errors.New("malformed request body")

// DecodeJSON refuses unknown fields: a field the DTO lacks must fail loudly, never be silently dropped.
// maxBytes is caller-supplied since platform/httpx has no access to service config.
func DecodeJSON(r *http.Request, v any, maxBytes int64) error {
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, maxBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("%w: %s", ErrMalformedBody, err)
	}
	return nil
}
