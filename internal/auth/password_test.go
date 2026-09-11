package auth

import (
	"errors"
	"testing"

	"github.com/loloDawit/go-admin/internal/errs"
)

// Tests use the cheapest cost: hashing dominates their runtime otherwise.
var h = NewTestHasher()

func TestHashPasswordThenCheckSucceeds(t *testing.T) {
	hash, err := h.Hash("correct-horse-battery-staple")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if err := h.Check(hash, "correct-horse-battery-staple"); err != nil {
		t.Fatalf("the correct password must verify, got: %v", err)
	}
}

// This is the regression test for the defect. Before the fix, SetPassword
// hashed the literal "test", so the real password did NOT verify and the
// literal "test" did.
func TestCheckPasswordRejectsWrongPassword(t *testing.T) {
	hash, err := h.Hash("correct-horse-battery-staple")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if err := h.Check(hash, "test"); err == nil {
		t.Fatal("the literal \"test\" must NOT verify against a different password")
	}
	if err := h.Check(hash, "wrong"); err == nil {
		t.Fatal("a wrong password must not verify")
	}
}

func TestHashPasswordRejectsEmpty(t *testing.T) {
	_, err := h.Hash("")
	if err == nil {
		t.Fatal("an empty password must be rejected")
	}
	if !errors.Is(err, errs.PasswordEmpty) {
		t.Fatalf("want errs.PasswordEmpty, got %v", err)
	}
}

func TestHashPasswordIsSalted(t *testing.T) {
	a, _ := h.Hash("same-password")
	b, _ := h.Hash("same-password")
	if a == b {
		t.Fatal("two hashes of the same password must differ (bcrypt salts each)")
	}
}
