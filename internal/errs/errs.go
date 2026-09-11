// Package errs is the single registry of application errors.
//
// Every error the API can return is declared in registry.go with a stable
// machine-readable Code, a client-safe Message, and the HTTP Status it maps
// to. Handlers return these values rather than constructing status codes and
// message strings inline, so that:
//
//   - a client can switch on Code without parsing prose;
//   - changing a message or status happens in exactly one place;
//   - no handler can invent a code that the frontend has never seen.
//
// This package deliberately imports no web framework. The HTTP mapping lives
// in internal/httpx, so errs can be used from seeding, CLI commands, and
// domain code that has no request in hand.
package errs

import (
	"errors"
	"fmt"
	"net/http"
)

// Error is an application error. Values in the registry are treated as
// immutable templates: Wrap and WithMessage return copies.
type Error struct {
	// Code is a stable, machine-readable identifier. It is part of the API
	// contract — renaming one is a breaking change.
	Code string

	// Message is safe to return to a client. It must never contain driver
	// output, SQL, file paths, or anything else that leaks internals.
	Message string

	// Status is the HTTP status this error maps to.
	Status int

	// cause is the underlying error. It is logged, never serialized.
	cause error
}

func (e *Error) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *Error) Unwrap() error { return e.cause }

// Is matches on Code, so a wrapped copy still satisfies errors.Is against the
// registry entry it was derived from.
func (e *Error) Is(target error) bool {
	var t *Error
	if !errors.As(target, &t) {
		return false
	}
	return t.Code == e.Code
}

// Wrap returns a copy carrying cause. The cause is available to logs via
// errors.Unwrap but is never sent to the client.
func (e *Error) Wrap(cause error) *Error {
	c := *e
	c.cause = cause
	return &c
}

// WithMessage returns a copy with a more specific client-facing message,
// keeping the same Code and Status. Use it to add detail a caller can act on
// ("title is required"), never to add internal detail.
func (e *Error) WithMessage(format string, args ...any) *Error {
	c := *e
	c.Message = fmt.Sprintf(format, args...)
	return &c
}

// Cause returns the wrapped error, or nil.
func (e *Error) Cause() error { return e.cause }

// define registers an error. It is the only constructor; every application
// error must be declared in registry.go so the set stays enumerable.
func define(status int, code, message string) *Error {
	return &Error{Status: status, Code: code, Message: message}
}

// From maps any error to an *Error. An unrecognised error becomes Internal
// with the original wrapped as the cause, so an unexpected failure can never
// leak its text to a client.
func From(err error) *Error {
	if err == nil {
		return nil
	}
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr
	}
	return Internal.Wrap(err)
}

// StatusOf returns the HTTP status for any error.
func StatusOf(err error) int {
	if err == nil {
		return http.StatusOK
	}
	return From(err).Status
}
