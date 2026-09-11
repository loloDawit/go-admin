// Package auth hashes and verifies passwords.
package auth

import (
	"github.com/loloDawit/go-admin/internal/errs"
	"golang.org/x/crypto/bcrypt"
)

const (
	DefaultCost = 12 // ~250ms: resists offline cracking without becoming a login DoS

	// bcrypt permits 4, but below 10 is not defensible for real passwords.
	// Tests bypass this via NewTestHasher.
	MinAllowedCost = 10
)

// Hasher is a value rather than a global so cost stays an explicit dependency.
type Hasher struct {
	cost int
}

// Operator-supplied costs are validated in config; this clamp is a backstop.
func NewHasher(cost int) Hasher {
	switch {
	case cost < bcrypt.MinCost:
		cost = bcrypt.MinCost
	case cost > bcrypt.MaxCost:
		cost = bcrypt.MaxCost
	}
	return Hasher{cost: cost}
}

// NewTestHasher is the cheapest cost bcrypt allows. Use it in every test that
// creates a user; hashing otherwise dominates the suite's runtime.
func NewTestHasher() Hasher { return Hasher{cost: bcrypt.MinCost} }

func (h Hasher) Cost() int { return h.cost }

func (h Hasher) Hash(plain string) (string, error) {
	if plain == "" {
		return "", errs.PasswordEmpty
	}
	cost := h.cost
	if cost == 0 {
		cost = DefaultCost // a zero-value Hasher must not hash at bcrypt's minimum
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(plain), cost)
	if err != nil {
		return "", errs.Internal.Wrap(err)
	}
	return string(hashed), nil
}

// Cost is read from the stored hash, so any Hasher verifies any hash.
func (h Hasher) Check(hash, plain string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
}
