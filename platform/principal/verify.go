package principal

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"
)

var (
	ErrMissing      = errors.New("principal: header or signature missing")
	ErrBadSignature = errors.New("principal: signature does not verify")
	ErrExpired      = errors.New("principal: principal has expired")
)

// Verify checks signature against header before unmarshalling the payload:
// the payload is attacker-controlled JSON until the signature has verified,
// so decoding it earlier would be work performed on unauthenticated input.
func Verify(header, signature string, key []byte, now time.Time) (Principal, error) {
	if header == "" || signature == "" {
		return Principal{}, ErrMissing
	}

	want, err := base64.RawURLEncoding.DecodeString(signature)
	if err != nil {
		return Principal{}, ErrBadSignature
	}

	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(header))
	// hmac.Equal, not ==: a timing-variable comparison leaks the signature
	// byte by byte to an attacker who can measure it.
	if !hmac.Equal(want, mac.Sum(nil)) {
		return Principal{}, ErrBadSignature
	}

	payload, err := base64.RawURLEncoding.DecodeString(header)
	if err != nil {
		return Principal{}, ErrBadSignature
	}

	var p Principal
	if err := json.Unmarshal(payload, &p); err != nil {
		return Principal{}, ErrBadSignature
	}

	if now.After(p.ExpiresAt) {
		return Principal{}, ErrExpired
	}

	return p, nil
}
