// Package auth holds password hashing. It is deliberately separate from the
// models package so it can be tested without a database — the original code
// buried this logic in models.User, which is why the bug it replaces
// (hashing the literal "test" for every account) survived unnoticed.
package auth

import (
	"github.com/loloDawit/go-admin/internal/errs"
	"golang.org/x/crypto/bcrypt"
)

const (
	// DefaultCost is the cost used when configuration does not specify one.
	// ~250ms on current hardware: slow enough to resist offline cracking,
	// fast enough not to be a login DoS vector. The original code hardcoded
	// 14 (~1s per hash), which is also why its test suite would have crawled.
	DefaultCost = 12

	// MinAllowedCost is the floor we accept from configuration. bcrypt itself
	// permits 4, but anything below 10 is not defensible for real passwords.
	// Tests may drop below this by constructing a Hasher directly.
	MinAllowedCost = 10
)

// Hasher hashes and verifies passwords at a configured cost. It is a value,
// not a package-level global, so the cost is always an explicit dependency of
// whatever needs to hash — and so tests can use a cheap cost without mutating
// shared state.
type Hasher struct {
	cost int
}

// NewHasher returns a Hasher at the given cost, clamped to bcrypt's own
// limits. Validation of operator-supplied values belongs in config; this
// clamp is a last line of defence against a nonsensical cost reaching bcrypt.
func NewHasher(cost int) Hasher {
	switch {
	case cost < bcrypt.MinCost:
		cost = bcrypt.MinCost
	case cost > bcrypt.MaxCost:
		cost = bcrypt.MaxCost
	}
	return Hasher{cost: cost}
}

// NewTestHasher returns the cheapest possible Hasher. Hashing dominates the
// runtime of any test that creates a user, so tests should always use this.
func NewTestHasher() Hasher { return Hasher{cost: bcrypt.MinCost} }

func (h Hasher) Cost() int { return h.cost }

func (h Hasher) Hash(plain string) (string, error) {
	if plain == "" {
		return "", errs.PasswordEmpty
	}
	cost := h.cost
	if cost == 0 {
		// A zero-value Hasher{} would otherwise silently hash at bcrypt's
		// minimum. Treat it as the safe default instead.
		cost = DefaultCost
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(plain), cost)
	if err != nil {
		return "", errs.Internal.Wrap(err)
	}
	return string(hashed), nil
}

// Check verifies plain against hash. It is a method for symmetry, but the
// cost is read from the stored hash, so any Hasher can verify any hash —
// including one written when the configured cost was different.
func (h Hasher) Check(hash, plain string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
}
