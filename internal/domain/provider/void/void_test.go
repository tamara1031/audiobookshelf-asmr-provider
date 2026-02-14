package void

import (
	"context"
	"testing"
	"time"
)

func TestVoidProvider_Search(t *testing.T) {
	p := NewProvider()
	results, err := p.Search(context.Background(), "any query")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestVoidProvider_ID(t *testing.T) {
	p := NewProvider()
	if p.ID() != "void" {
		t.Errorf("expected ID 'void', got %s", p.ID())
	}
}

func TestVoidProvider_CacheTTL(t *testing.T) {
	p := NewProvider()
	if p.CacheTTL() != 24*time.Hour {
		t.Errorf("expected 24h TTL, got %v", p.CacheTTL())
	}
}
