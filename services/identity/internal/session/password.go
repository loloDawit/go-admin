// Package session will hold login, token issuance, and session handlers
// (Task 5). This file contributes only password hashing.
package session

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

// ErrEmptyPassword is returned when Hash is called with an empty string.
// bcrypt itself accepts an empty password; rejecting it here keeps a blank
// password from ever reaching a stored hash.
var ErrEmptyPassword = errors.New("password must not be empty")

// Hasher wraps bcrypt with a fixed cost. The cost is configuration
// (BCRYPT_COST), never a literal in code.
type Hasher struct {
	cost int
}

// NewHasher builds a Hasher at the given bcrypt cost. Callers validate the
// cost against bcrypt.MinCost/MaxCost before constructing one.
func NewHasher(cost int) *Hasher {
	return &Hasher{cost: cost}
}

// Hash returns the bcrypt hash of plain, salted by bcrypt itself.
func (h *Hasher) Hash(plain string) (string, error) {
	if plain == "" {
		return "", ErrEmptyPassword
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), h.cost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// Compare reports whether plain matches hash, returning a non-nil error on
// any mismatch or malformed hash.
func (h *Hasher) Compare(hash, plain string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
}
