package all

import (
	"context"
	"errors"
	"testing"
	"time"

	"audiobookshelf-asmr-provider/internal/service"
)

type mockProvider struct {
	id      string
	results []service.AbsBookMetadata
	err     error
}

func (m *mockProvider) ID() string              { return m.id }
func (m *mockProvider) CacheTTL() time.Duration { return 1 * time.Hour }
func (m *mockProvider) Search(_ context.Context, _ string) ([]service.AbsBookMetadata, error) {
	return m.results, m.err
}

func TestAllProvider_Search(t *testing.T) {
	p1 := &mockProvider{
		id:      "p1",
		results: []service.AbsBookMetadata{{Title: "Result 1"}},
	}
	p2 := &mockProvider{
		id:      "p2",
		results: []service.AbsBookMetadata{{Title: "Result 2"}},
	}
	pFail := &mockProvider{
		id:  "fail",
		err: errors.New("provider failed"),
	}

	ap := NewProvider(p1, p2, pFail)

	results, err := ap.Search(context.Background(), "test")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	// Should have 2 results, from p1 and p2. pFail should be skipped by the search goroutine.
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	foundP1 := false
	foundP2 := false
	for _, r := range results {
		if r.Title == "Result 1" {
			foundP1 = true
		}
		if r.Title == "Result 2" {
			foundP2 = true
		}
	}

	if !foundP1 || !foundP2 {
		t.Errorf("missing results: p1=%v, p2=%v", foundP1, foundP2)
	}
}

func TestAllProvider_ID(t *testing.T) {
	ap := NewProvider()
	if ap.ID() != "all" {
		t.Errorf("expected ID 'all', got %s", ap.ID())
	}
}

func TestAllProvider_CacheTTL(t *testing.T) {
	ap := NewProvider()
	if ap.CacheTTL() != 1*time.Hour {
		t.Errorf("expected 1h TTL, got %v", ap.CacheTTL())
	}
}
