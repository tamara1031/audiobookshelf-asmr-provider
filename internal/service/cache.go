package service

import (
	"sync"
	"time"

	"audiobookshelf-asmr-provider/internal/domain"
)

// cacheEntry holds the cached metadata and its expiration time.
type cacheEntry struct {
	data   []domain.AbsBookMetadata
	expiry time.Time
}

// Cache provides thread-safe in-memory caching for metadata results.
type Cache struct {
	entries         map[string]cacheEntry
	mu              sync.RWMutex
	maxSize         int
	cleanupInterval time.Duration
}

// NewCache creates a new cache and starts a background goroutine to evict expired entries.
func NewCache() *Cache {
	c := &Cache{
		entries:         make(map[string]cacheEntry),
		maxSize:         10000,
		cleanupInterval: 1 * time.Hour,
	}
	go c.startCleanup()
	return c
}

// Get retrieves cached data for the given key, if it exists and has not expired.
func (c *Cache) Get(key string) ([]domain.AbsBookMetadata, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, found := c.entries[key]
	if found && time.Now().Before(entry.expiry) {
		return entry.data, true
	}
	return nil, false
}

// Put stores data in the cache with the given TTL.
// If the cache exceeds maxSize, one entry is evicted.
func (c *Cache) Put(key string, data []domain.AbsBookMetadata, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = cacheEntry{
		data:   data,
		expiry: time.Now().Add(ttl),
	}

	// Size limit protection
	if len(c.entries) > c.maxSize {
		for k := range c.entries {
			delete(c.entries, k)
			break
		}
	}
}

// EvictExpired removes all expired entries from the cache.
func (c *Cache) EvictExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for k, v := range c.entries {
		if now.After(v.expiry) {
			delete(c.entries, k)
		}
	}
}

// Len returns the number of entries in the cache.
func (c *Cache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

// startCleanup periodically removes expired entries.
func (c *Cache) startCleanup() {
	ticker := time.NewTicker(c.cleanupInterval)
	for range ticker.C {
		c.EvictExpired()
	}
}
