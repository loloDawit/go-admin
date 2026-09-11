package errs

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

// Codes are part of the API contract, so a duplicate would make two different
// failures indistinguishable to a client.
func TestCodesAreUnique(t *testing.T) {
	seen := make(map[string]bool, len(All))
	for _, e := range All {
		if seen[e.Code] {
			t.Errorf("duplicate error code %q", e.Code)
		}
		seen[e.Code] = true
	}
}

func TestEveryErrorIsWellFormed(t *testing.T) {
	for _, e := range All {
		if e.Code == "" {
			t.Errorf("error with status %d has no code", e.Status)
		}
		if e.Message == "" {
			t.Errorf("%s has no message", e.Code)
		}
		if e.Status < 400 || e.Status > 599 {
			t.Errorf("%s has status %d; application errors must be 4xx or 5xx", e.Code, e.Status)
		}
		if strings.ToLower(e.Code) != e.Code || strings.Contains(e.Code, " ") {
			t.Errorf("%s: codes must be lower_snake_case", e.Code)
		}
	}
}

// A 5xx message must not describe internals — those belong in the wrapped
// cause, which is logged but never serialized.
func TestServerErrorMessagesAreVague(t *testing.T) {
	leaky := []string{"sql", "gorm", "mysql", "panic", "nil pointer", "/Users", "goroutine"}
	for _, e := range All {
		if e.Status < 500 {
			continue
		}
		lower := strings.ToLower(e.Message)
		for _, bad := range leaky {
			if strings.Contains(lower, bad) {
				t.Errorf("%s message leaks an internal detail (%q): %q", e.Code, bad, e.Message)
			}
		}
	}
}

func TestWrapPreservesIdentityAndHidesCause(t *testing.T) {
	cause := errors.New("dial tcp 127.0.0.1:3306: connection refused")
	wrapped := Database.Wrap(cause)

	if !errors.Is(wrapped, Database) {
		t.Error("a wrapped error must still match its registry entry")
	}
	if !errors.Is(wrapped, cause) {
		t.Error("the cause must remain reachable via errors.Is for logging")
	}
	if strings.Contains(wrapped.Message, "connection refused") {
		t.Error("the client-facing Message must not absorb the cause")
	}
	if Database.Cause() != nil {
		t.Error("Wrap must not mutate the shared registry entry")
	}
}

func TestWithMessageKeepsCodeAndStatus(t *testing.T) {
	specific := ValidationFailed.WithMessage("title is required")

	if specific.Code != ValidationFailed.Code || specific.Status != ValidationFailed.Status {
		t.Error("WithMessage must preserve Code and Status")
	}
	if !errors.Is(specific, ValidationFailed) {
		t.Error("WithMessage must preserve identity")
	}
	if ValidationFailed.Message == "title is required" {
		t.Error("WithMessage must not mutate the shared registry entry")
	}
}

// An error from outside the registry must never reach a client verbatim.
func TestFromMapsUnknownErrorsToInternal(t *testing.T) {
	got := From(errors.New("ERROR 1146 (42S02): Table 'go_admin.secrets' doesn't exist"))

	if got.Code != Internal.Code {
		t.Errorf("want internal_error, got %s", got.Code)
	}
	if strings.Contains(got.Message, "go_admin.secrets") {
		t.Error("an unknown error's text must not become the client message")
	}
	if got.Status != http.StatusInternalServerError {
		t.Errorf("want 500, got %d", got.Status)
	}
}

func TestFromNilIsNil(t *testing.T) {
	if From(nil) != nil {
		t.Error("From(nil) must be nil")
	}
}
