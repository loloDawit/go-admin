package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// ErrMalformedBody is DecodeJSON's one sentinel for every decode failure:
// invalid JSON, an unknown field, or a body over the cap. httpx owns the wire
// shape only; the caller's own httperr package maps this to a status code.
var ErrMalformedBody = errors.New("malformed request body")

// DecodeJSON refuses unknown fields: a request carrying role_id where the DTO
// has none must fail loudly, not be silently dropped. That silence is how the
// original application granted itself admin.
//
// maxBytes bounds how much of the body is read before it is rejected as
// oversized; platform/httpx has no access to service config, so the caller
// supplies the limit rather than this package hardcoding one.
func DecodeJSON(r *http.Request, v any, maxBytes int64) error {
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, maxBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("%w: %s", ErrMalformedBody, err)
	}
	return nil
}
