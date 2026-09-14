package auth

import (
	"testing"
	"time"
)

func TestCacheKeyIsTheHashOfTheTokenNotTheToken(t *testing.T) {
	c := NewCache(time.Minute)
	const token = "super-secret-live-session-token"
	c.Set(token, resolved{StaffID: "staff-1"})

	if _, ok := c.m[token]; ok {
		t.Fatal("cache stored the plaintext token as its own key")
	}
	if len(c.m) != 1 {
		t.Fatalf("want exactly one entry, got %d", len(c.m))
	}
	for key := range c.m {
		if key == token {
			t.Fatal("cache key equals the plaintext token")
		}
		if len(key) != 64 { // hex-encoded SHA-256 is 64 characters
			t.Errorf("key length: want 64 (sha256 hex), got %d (%q)", len(key), key)
		}
	}
}

func TestCacheGetReturnsWhatWasSet(t *testing.T) {
	c := NewCache(time.Minute)
	want := resolved{StaffID: "staff-1", Permissions: []string{"orders_view"}}
	c.Set("tok", want)

	got, ok := c.Get("tok")
	if !ok {
		t.Fatal("want a cache hit")
	}
	if got.StaffID != want.StaffID {
		t.Errorf("StaffID: want %q, got %q", want.StaffID, got.StaffID)
	}
}

func TestCacheGetMissesAnUnknownToken(t *testing.T) {
	c := NewCache(time.Minute)
	if _, ok := c.Get("never-set"); ok {
		t.Fatal("want a cache miss for a token never set")
	}
}

func TestCacheEntryExpiresAfterItsTTL(t *testing.T) {
	c := NewCache(10 * time.Millisecond)
	c.Set("tok", resolved{StaffID: "staff-1"})

	if _, ok := c.Get("tok"); !ok {
		t.Fatal("want a hit immediately after Set")
	}

	time.Sleep(30 * time.Millisecond)

	if _, ok := c.Get("tok"); ok {
		t.Fatal("want a miss once the entry's TTL has passed")
	}
}
