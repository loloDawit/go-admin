package principal_test

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/loloDawit/go-admin/platform/principal"
)

var key = []byte("test-signing-key")

func testPrincipal(now time.Time) principal.Principal {
	return principal.Principal{
		StaffID:            "staff-1",
		Permissions:        []string{"view_staff"},
		MustChangePassword: true,
		IssuedAt:           now,
		ExpiresAt:          now.Add(time.Minute),
	}
}

func TestVerifyAcceptsWhatSignProduced(t *testing.T) {
	now := time.Now()
	p := testPrincipal(now)

	header, sig, err := principal.Sign(p, key)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	got, err := principal.Verify(header, sig, key, now)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if got.StaffID != p.StaffID {
		t.Errorf("StaffID: want %q, got %q", p.StaffID, got.StaffID)
	}
	if len(got.Permissions) != 1 || got.Permissions[0] != "view_staff" {
		t.Errorf("Permissions: got %v", got.Permissions)
	}
	if !got.MustChangePassword {
		t.Error("MustChangePassword: want true, got false")
	}
}

// forgePermission decodes the base64 header, appends a permission the signer
// never issued, and re-encodes it, leaving the original signature untouched.
func forgePermission(header, extra string) string {
	raw, err := base64.RawURLEncoding.DecodeString(header)
	if err != nil {
		panic(err)
	}
	var p principal.Principal
	if err := json.Unmarshal(raw, &p); err != nil {
		panic(err)
	}
	p.Permissions = append(p.Permissions, extra)
	tampered, err := json.Marshal(p)
	if err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(tampered)
}

func TestVerifyRejectsATamperedPayload(t *testing.T) {
	now := time.Now()
	p := testPrincipal(now)

	h, sig, err := principal.Sign(p, key)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	tampered := forgePermission(h, "edit_staff")
	if _, err := principal.Verify(tampered, sig, key, now); !errors.Is(err, principal.ErrBadSignature) {
		t.Fatalf("tampered payload accepted: %v", err)
	}
}

func TestVerifyRejectsADifferentKey(t *testing.T) {
	now := time.Now()
	p := testPrincipal(now)

	h, sig, err := principal.Sign(p, key)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	otherKey := []byte("a-different-key")
	if _, err := principal.Verify(h, sig, otherKey, now); !errors.Is(err, principal.ErrBadSignature) {
		t.Fatalf("want ErrBadSignature, got %v", err)
	}
}

func TestVerifyRejectsAnExpiredPrincipal(t *testing.T) {
	now := time.Now()
	p := testPrincipal(now)
	p.ExpiresAt = now.Add(-time.Second)

	h, sig, err := principal.Sign(p, key)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	if _, err := principal.Verify(h, sig, key, now); !errors.Is(err, principal.ErrExpired) {
		t.Fatalf("want ErrExpired, got %v", err)
	}
}

func TestVerifyRejectsAnEmptySignature(t *testing.T) {
	now := time.Now()
	p := testPrincipal(now)

	h, _, err := principal.Sign(p, key)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	if _, err := principal.Verify(h, "", key, now); !errors.Is(err, principal.ErrMissing) {
		t.Fatalf("want ErrMissing, got %v", err)
	}
	if _, err := principal.Verify("", "", key, now); !errors.Is(err, principal.ErrMissing) {
		t.Fatalf("want ErrMissing for empty header too, got %v", err)
	}
}
