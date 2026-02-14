package service

import (
	"fmt"
	"testing"
	"time"

	"audiobookshelf-asmr-provider/internal/domain"
)

func TestCache_GetPut(t *testing.T) {
	c := NewCache()

	data := []domain.AbsBookMetadata{{Title: "Cached Item"}}
	c.Put("key1", data, 1*time.Hour)

	got, ok := c.Get("key1")
	if !ok {
		t.Fatal("expected cache hit for key1")
	}
	if len(got) != 1 || got[0].Title != "Cached Item" {
		t.Errorf("unexpected cached data: %+v", got)
	}
}

func TestCache_Get_Miss(t *testing.T) {
	c := NewCache()

	_, ok := c.Get("nonexistent")
	if ok {
		t.Error("expected cache miss for nonexistent key")
	}
}

func TestCache_Get_Expired(t *testing.T) {
	c := NewCache()

	c.Put("expired", []domain.AbsBookMetadata{{Title: "Old"}}, 1*time.Millisecond)
	time.Sleep(5 * time.Millisecond)

	_, ok := c.Get("expired")
	if ok {
		t.Error("expected cache miss for expired entry")
	}
}

func TestCache_EvictExpired(t *testing.T) {
	c := NewCache()

	c.mu.Lock()
	c.entries["expired"] = cacheEntry{
		data:   []domain.AbsBookMetadata{{Title: "Old"}},
		expiry: time.Now().Add(-1 * time.Hour),
	}
	c.entries["valid"] = cacheEntry{
		data:   []domain.AbsBookMetadata{{Title: "New"}},
		expiry: time.Now().Add(1 * time.Hour),
	}
	c.mu.Unlock()

	c.EvictExpired()

	if _, ok := c.Get("expired"); ok {
		t.Error("expected expired entry to be evicted")
	}
	if _, ok := c.Get("valid"); !ok {
		t.Error("expected valid entry to be kept")
	}
}

func TestCache_Put_Overflow(t *testing.T) {
	c := NewCache()

	// Fill cache beyond maxSize
	c.mu.Lock()
	for i := 0; i < c.maxSize+1; i++ {
		c.entries[fmt.Sprintf("key_%d", i)] = cacheEntry{
			data:   nil,
			expiry: time.Now().Add(1 * time.Hour),
		}
	}
	c.mu.Unlock()

	// Put should trigger eviction
	c.Put("new_key", []domain.AbsBookMetadata{{Title: "New"}}, 1*time.Hour)

	if c.Len() > c.maxSize+1 {
		t.Errorf("cache should not grow unbounded, got %d entries", c.Len())
	}
}

func TestCache_Len(t *testing.T) {
	c := NewCache()

	if c.Len() != 0 {
		t.Errorf("expected empty cache, got %d", c.Len())
	}

	c.Put("a", nil, 1*time.Hour)
	c.Put("b", nil, 1*time.Hour)

	if c.Len() != 2 {
		t.Errorf("expected 2 entries, got %d", c.Len())
	}
}
