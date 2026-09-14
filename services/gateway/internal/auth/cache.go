package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"
)

type entry struct {
	r       resolved
	expires time.Time
}

// Cache is per-instance; a revoked session can survive up to its TTL on each
// replica independently. Accepted, not solved, here.
type Cache struct {
	mu  sync.Mutex
	ttl time.Duration
	m   map[string]entry
}

func NewCache(ttl time.Duration) *Cache {
	return &Cache{ttl: ttl, m: make(map[string]entry)}
}

func (c *Cache) Get(token string) (resolved, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	e, ok := c.m[cacheKey(token)]
	if !ok || time.Now().After(e.expires) {
		return resolved{}, false
	}
	return e.r, true
}

func (c *Cache) Set(token string, r resolved) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.m[cacheKey(token)] = entry{r: r, expires: time.Now().Add(c.ttl)}
}

// Keyed by token hash: a heap dump must not yield live session tokens.
func cacheKey(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
