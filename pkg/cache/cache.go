package cache

import (
	"sync"
	"time"

	"github.com/stebennett/bin-notifier/pkg/scraper"
)

type cacheEntry struct {
	binTimes  []scraper.BinTime
	expiresAt time.Time
}

// ScraperCache wraps a BinScraper with in-memory TTL caching.
type ScraperCache struct {
	mu      sync.RWMutex
	entries map[string]cacheEntry
	ttl     time.Duration
	now     func() time.Time
}

// New creates a ScraperCache with the given TTL.
func New(ttl time.Duration) *ScraperCache {
	return &ScraperCache{
		entries: make(map[string]cacheEntry),
		ttl:     ttl,
		now:     time.Now,
	}
}

func cacheKey(postcode, addressCode string) string {
	return postcode + "|" + addressCode
}

// Get returns cached bin times if present and not expired.
func (c *ScraperCache) Get(postcode, addressCode string) ([]scraper.BinTime, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := cacheKey(postcode, addressCode)
	entry, ok := c.entries[key]
	if !ok {
		return nil, false
	}
	if c.now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.binTimes, true
}

// Set stores bin times in the cache.
func (c *ScraperCache) Set(postcode, addressCode string, binTimes []scraper.BinTime) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := cacheKey(postcode, addressCode)
	c.entries[key] = cacheEntry{
		binTimes:  binTimes,
		expiresAt: c.now().Add(c.ttl),
	}
}

// Invalidate removes a specific cache entry.
func (c *ScraperCache) Invalidate(postcode, addressCode string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, cacheKey(postcode, addressCode))
}
