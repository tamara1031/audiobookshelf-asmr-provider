package cache

import (
	"log/slog"
	"sync"
	"time"

	"audiobookshelf-asmr-provider/internal/service"
)

// cacheEntry holds the cached metadata and its expiration time.
type cacheEntry struct {
	data   []service.AbsBookMetadata
	expiry time.Time
}

// MemoryCache provides thread-safe in-memory caching for metadata results.
type MemoryCache struct {
	entries         map[string]cacheEntry
	mu              sync.RWMutex
	maxSize         int
	cleanupInterval time.Duration
}

// NewMemoryCache creates a new cache and starts a background goroutine to evict expired entries.
func NewMemoryCache() *MemoryCache {
	c := &MemoryCache{
		entries:         make(map[string]cacheEntry),
		maxSize:         10000,
		cleanupInterval: 1 * time.Hour,
	}
	go c.startCleanup()
	return c
}

// Get retrieves cached data for the given key, if it exists and has not expired.
func (c *MemoryCache) Get(key string) ([]service.AbsBookMetadata, bool) {
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
func (c *MemoryCache) Put(key string, data []service.AbsBookMetadata, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = cacheEntry{
		data:   data,
		expiry: time.Now().Add(ttl),
	}

	// Size limit protection
	if len(c.entries) > c.maxSize {
		// Evict a random entry (map iteration order is random)
		for k := range c.entries {
			delete(c.entries, k)
			break
		}
	}
}

// EvictExpired removes all expired entries from the cache.
func (c *MemoryCache) EvictExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	initialSize := len(c.entries)
	now := time.Now()
	for k, v := range c.entries {
		if now.After(v.expiry) {
			delete(c.entries, k)
		}
	}
	evictedCount := initialSize - len(c.entries)
	if evictedCount > 0 {
		slog.Debug("Evicted expired cache entries", "count", evictedCount)
	}
}

// Len returns the number of entries in the cache.
func (c *MemoryCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

// startCleanup periodically removes expired entries.
func (c *MemoryCache) startCleanup() {
	ticker := time.NewTicker(c.cleanupInterval)
	defer ticker.Stop()
	for range ticker.C {
		c.EvictExpired()
	}
}
