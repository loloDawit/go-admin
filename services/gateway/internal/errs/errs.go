// Package errs is the gateway's one error registry: every sentinel and every
// operation string lives here, wrapped via Wrap.
package errs

import (
	"errors"
	"fmt"
)

// ErrSessionInvalid: the gateway middleware treats it the same as a missing
// cookie — the request proceeds unauthenticated.
var ErrSessionInvalid = errors.New("session is not authenticated")

// ErrInvalidUpstreamURL is returned by routing.New at startup for a
// malformed or incomplete upstream URL.
var ErrInvalidUpstreamURL = errors.New("upstream is not a valid URL")

// ErrRateLimited is answered by the gateway itself; the request never reaches
// an upstream.
var ErrRateLimited = errors.New("too many requests")

// ErrFaultsNotPermitted and ErrInvalidErrorRate are startup failures: a
// misconfigured injector must never reach a request.
var (
	ErrFaultsNotPermitted = errors.New("fault injection is permitted only in the dev profile")
	ErrInvalidErrorRate   = errors.New("FAULT_ERROR_RATE must be between 0 and 1")
)

const (
	OpBuildValidateRequest   = "build session validate request"
	OpCallIdentityValidate   = "call identity validate endpoint"
	OpDecodeValidateResponse = "decode identity validate response"
	OpSignPrincipal          = "sign principal"
	OpParseUpstreamURL       = "parse upstream url"
)

func Wrap(op string, err error) error {
	return fmt.Errorf("%s: %w", op, err)
}
