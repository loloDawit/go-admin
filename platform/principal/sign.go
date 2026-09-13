package principal

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
)

// The signature covers the encoded header, not the raw payload: Verify must
// MAC the same bytes this produces, or a correct signature will never match.
func Sign(p Principal, key []byte) (header, signature string, err error) {
	payload, err := json.Marshal(p)
	if err != nil {
		return "", "", fmt.Errorf("marshal principal: %w", err)
	}
	header = base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(header))
	return header, base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}
