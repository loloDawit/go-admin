// Package errs is the single registry of application errors. Handlers return
// these values instead of building statuses and messages inline;
// convention_test.go enforces that.
//
// It imports no web framework, so seeding and CLI code can use it. The HTTP
// mapping lives in internal/httpx.
package errs

import (
	"errors"
	"fmt"
	"net/http"
)

// Registry values are immutable templates; Wrap and WithMessage copy.
type Error struct {
	Code    string // part of the API contract; renaming one is a breaking change
	Message string // client-safe: never driver output, SQL, or paths
	Status  int
	cause   error // logged, never serialized
}

func (e *Error) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *Error) Unwrap() error { return e.cause }

// Matches on Code so wrapped copies satisfy errors.Is against their template.
func (e *Error) Is(target error) bool {
	var t *Error
	if !errors.As(target, &t) {
		return false
	}
	return t.Code == e.Code
}

// Wrap attaches a cause for logging. It never reaches the client.
func (e *Error) Wrap(cause error) *Error {
	c := *e
	c.cause = cause
	return &c
}

// WithMessage adds detail a caller can act on, never internal detail.
func (e *Error) WithMessage(format string, args ...any) *Error {
	c := *e
	c.Message = fmt.Sprintf(format, args...)
	return &c
}

func (e *Error) Cause() error { return e.cause }

// The only constructor, so the error set stays enumerable.
func define(status int, code, message string) *Error {
	return &Error{Status: status, Code: code, Message: message}
}

// Unregistered errors become Internal with the original as cause, so no
// unexpected failure can leak its text to a client.
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

func StatusOf(err error) int {
	if err == nil {
		return http.StatusOK
	}
	return From(err).Status
}
