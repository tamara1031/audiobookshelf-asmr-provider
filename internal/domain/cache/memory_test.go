package cache

import (
	"testing"
	"time"

	"audiobookshelf-asmr-provider/internal/service"
)

func TestMemoryCache_GetPut(t *testing.T) {
	c := NewMemoryCache()
	key := "test_key"
	data := []service.AbsBookMetadata{{Title: "Test"}}
	ttl := 1 * time.Hour

	c.Put(key, data, ttl)

	got, ok := c.Get(key)
	if !ok {
		t.Fatal("expected item to be in cache")
	}
	if len(got) != 1 || got[0].Title != "Test" {
		t.Errorf("unexpected data: %+v", got)
	}
}

func TestMemoryCache_Expiration(t *testing.T) {
	c := NewMemoryCache()
	key := "expired_key"
	data := []service.AbsBookMetadata{{Title: "Expired"}}
	ttl := 1 * time.Millisecond

	c.Put(key, data, ttl)
	time.Sleep(10 * time.Millisecond)

	_, ok := c.Get(key)
	if ok {
		t.Error("expected item to be expired")
	}
}

func TestMemoryCache_EvictExpired(t *testing.T) {
	c := NewMemoryCache()
	key := "key"
	data := []service.AbsBookMetadata{{Title: "Data"}}
	ttl := 1 * time.Millisecond

	c.Put(key, data, ttl)
	time.Sleep(10 * time.Millisecond)

	c.EvictExpired()

	if c.Len() != 0 {
		t.Errorf("expected 0 items, got %d", c.Len())
	}
}

func TestMemoryCache_Len(t *testing.T) {
	c := NewMemoryCache()
	c.Put("a", []service.AbsBookMetadata{}, 1*time.Hour)
	c.Put("b", []service.AbsBookMetadata{}, 1*time.Hour)
	if c.Len() != 2 {
		t.Errorf("expected Len 2, got %d", c.Len())
	}
}
