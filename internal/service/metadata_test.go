package service

import (
	"context"
	"errors"
	"testing"
	"time"
)

// MockProvider implements Provider for testing.
type MockProvider struct {
	IDVal         string
	SearchResults []AbsBookMetadata
	SearchErr     error
	MockCacheTTL  time.Duration
}

func (m *MockProvider) ID() string { return m.IDVal }

func (m *MockProvider) Search(_ context.Context, _ string) ([]AbsBookMetadata, error) {
	return m.SearchResults, m.SearchErr
}

func (m *MockProvider) CacheTTL() time.Duration { return m.MockCacheTTL }

// MockCache implements Cache for testing.
type MockCache struct {
	GetFunc func(key string) ([]AbsBookMetadata, bool)
	PutFunc func(key string, data []AbsBookMetadata, ttl time.Duration)
}

func (m *MockCache) Get(key string) ([]AbsBookMetadata, bool) {
	if m.GetFunc != nil {
		return m.GetFunc(key)
	}
	return nil, false
}

func (m *MockCache) Put(key string, data []AbsBookMetadata, ttl time.Duration) {
	if m.PutFunc != nil {
		m.PutFunc(key, data, ttl)
	}
}

func TestService_Search(t *testing.T) {
	mockData := []AbsBookMetadata{
		{Title: "Test Book", ISBN: "RJ123456"},
	}
	mockProvider := &MockProvider{
		IDVal:         "test_provider",
		SearchResults: mockData,
		MockCacheTTL:  1 * time.Hour,
	}

	// Simple map-based mock for this test
	store := make(map[string][]AbsBookMetadata)
	cache := &MockCache{
		GetFunc: func(key string) ([]AbsBookMetadata, bool) {
			d, ok := store[key]
			return d, ok
		},
		PutFunc: func(key string, data []AbsBookMetadata, _ time.Duration) {
			store[key] = data
		},
	}

	svc := NewService(cache, mockProvider)

	// 1. Initial Search (should call provider)
	resp, err := svc.Search(context.Background(), "RJ123456")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(resp.Matches) != 1 {
		t.Errorf("Expected 1 match, got %d", len(resp.Matches))
	}
	if resp.Matches[0].Title != "Test Book" {
		t.Errorf("Expected title 'Test Book', got '%s'", resp.Matches[0].Title)
	}

	// 2. Cached Search (should not fail even if provider errors)
	mockProvider.SearchErr = context.DeadlineExceeded // simulate failure
	resp, err = svc.Search(context.Background(), "RJ123456")
	if err != nil {
		t.Fatalf("Cached search failed: %v", err)
	}
	if len(resp.Matches) != 1 {
		t.Errorf("Expected 1 match from cache, got %d", len(resp.Matches))
	}
}

func TestService_SearchByProviderID(t *testing.T) {
	mockProvider := &MockProvider{
		IDVal:         "provider_a",
		SearchResults: []AbsBookMetadata{{Title: "A"}},
	}
	cache := &MockCache{}
	svc := NewService(cache, mockProvider)

	// Valid Provider
	resp, err := svc.SearchByProviderID(context.Background(), "provider_a", "query")
	if err != nil {
		t.Fatalf("SearchByProviderID failed: %v", err)
	}
	if len(resp.Matches) != 1 {
		t.Errorf("Expected 1 match, got %d", len(resp.Matches))
	}

	// Invalid Provider
	_, err = svc.SearchByProviderID(context.Background(), "provider_b", "query")
	if err == nil {
		t.Error("Expected error for non-existent provider, got nil")
	}

	// Provider returns error
	mockProvider.SearchErr = errors.New("provider failure")
	_, err = svc.SearchByProviderID(context.Background(), "provider_a", "new_query")
	if err == nil {
		t.Error("Expected error when provider fails, got nil")
	}
}

func TestService_Providers(t *testing.T) {
	p1 := &MockProvider{IDVal: "a"}
	p2 := &MockProvider{IDVal: "b"}
	cache := &MockCache{}
	svc := NewService(cache, p1, p2)

	providers := svc.Providers()
	if len(providers) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(providers))
	}
	if providers[0].ID() != "a" || providers[1].ID() != "b" {
		t.Errorf("unexpected provider IDs: %s, %s", providers[0].ID(), providers[1].ID())
	}
}

func TestService_Search_ProviderErrorContinues(t *testing.T) {
	failing := &MockProvider{
		IDVal:     "fail",
		SearchErr: context.DeadlineExceeded,
	}
	working := &MockProvider{
		IDVal:         "ok",
		SearchResults: []AbsBookMetadata{{Title: "OK"}},
		MockCacheTTL:  1 * time.Hour,
	}
	cache := &MockCache{}
	svc := NewService(cache, failing, working)

	resp, err := svc.Search(context.Background(), "test")
	if err != nil {
		t.Fatalf("Search should not error when one provider fails: %v", err)
	}
	if len(resp.Matches) != 1 || resp.Matches[0].Title != "OK" {
		t.Errorf("expected 1 match from working provider, got %+v", resp.Matches)
	}
}

func TestService_SearchProviderWithCache_ZeroTTL(t *testing.T) {
	mock := &MockProvider{
		IDVal:         "zero_ttl",
		SearchResults: []AbsBookMetadata{{Title: "Cached"}},
		MockCacheTTL:  0, // zero TTL
	}
	store := make(map[string][]AbsBookMetadata)
	cache := &MockCache{
		GetFunc: func(key string) ([]AbsBookMetadata, bool) {
			d, ok := store[key]
			return d, ok
		},
		PutFunc: func(key string, data []AbsBookMetadata, _ time.Duration) {
			store[key] = data
		},
	}
	svc := NewService(cache, mock)

	// First call populates cache
	_, err := svc.Search(context.Background(), "q")
	if err != nil {
		t.Fatalf("first search failed: %v", err)
	}

	// Make provider return error — cached result should still work
	mock.SearchErr = context.DeadlineExceeded
	resp, err := svc.Search(context.Background(), "q")
	if err != nil {
		t.Fatalf("cached search failed: %v", err)
	}
	if len(resp.Matches) != 1 || resp.Matches[0].Title != "Cached" {
		t.Errorf("expected cached result, got %+v", resp.Matches)
	}
}
