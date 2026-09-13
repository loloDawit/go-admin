package session_test

import (
	"errors"
	"testing"

	"github.com/loloDawit/go-admin/services/identity/internal/session"
)

// bcrypt.MinCost keeps these tests fast; production cost comes from config.
const testCost = 4

func TestHashRejectsAnEmptyPassword(t *testing.T) {
	if _, err := session.NewHasher(testCost).Hash(""); !errors.Is(err, session.ErrEmptyPassword) {
		t.Fatalf("empty password accepted: %v", err)
	}
}

func TestCompareRejectsTheWrongPassword(t *testing.T) {
	h := session.NewHasher(testCost)

	hash, err := h.Hash("correct-horse-battery-staple")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}

	if err := h.Compare(hash, "wrong-password"); err == nil {
		t.Fatal("wrong password compared as a match")
	}
}

func TestCompareAcceptsTheRightPassword(t *testing.T) {
	h := session.NewHasher(testCost)

	hash, err := h.Hash("correct-horse-battery-staple")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}

	if err := h.Compare(hash, "correct-horse-battery-staple"); err != nil {
		t.Fatalf("right password rejected: %v", err)
	}
}

func TestHashOfTheSamePasswordDiffers(t *testing.T) {
	h := session.NewHasher(testCost)

	a, err := h.Hash("correct-horse-battery-staple")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	b, err := h.Hash("correct-horse-battery-staple")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}

	if a == b {
		t.Fatal("two hashes of the same password matched: salt not applied")
	}
}
