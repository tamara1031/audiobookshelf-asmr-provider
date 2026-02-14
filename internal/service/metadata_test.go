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
	// Service.Search delegates to provider with ID "all"
	mockProvider := &MockProvider{
		IDVal:         "all",
		SearchResults: mockData,
		MockCacheTTL:  1 * time.Hour,
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

	svc := NewService(cache, mockProvider)

	// 1. Initial Search (should call provider "all")
	resp, err := svc.Search(context.Background(), "RJ123456")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(resp.Matches) != 1 {
		t.Errorf("Expected 1 match, got %d", len(resp.Matches))
	}
}

func TestService_SearchByProviderID(t *testing.T) {
	p1 := &MockProvider{IDVal: "p1", SearchResults: []AbsBookMetadata{{Title: "Result 1"}}}
	pAll := &MockProvider{IDVal: "all", SearchResults: []AbsBookMetadata{{Title: "Result 1"}, {Title: "Result 2"}}}

	svc := NewService(&MockCache{}, p1, pAll)

	t.Run("valid provider", func(t *testing.T) {
		resp, err := svc.SearchByProviderID(context.Background(), "p1", "test")
		if err != nil {
			t.Fatalf("SearchByProviderID failed: %v", err)
		}
		if len(resp.Matches) != 1 || resp.Matches[0].Title != "Result 1" {
			t.Errorf("unexpected matches: %+v", resp.Matches)
		}
	})

	t.Run("all providers", func(t *testing.T) {
		resp, err := svc.SearchByProviderID(context.Background(), "all", "test")
		if err != nil {
			t.Fatalf("SearchByProviderID failed for 'all': %v", err)
		}
		if len(resp.Matches) != 2 {
			t.Errorf("expected 2 matches from 'all' mock, got %d", len(resp.Matches))
		}
	})

	t.Run("unknown provider returns empty result (void)", func(t *testing.T) {
		resp, err := svc.SearchByProviderID(context.Background(), "p3", "test")
		if err != nil {
			t.Fatalf("Expected no error for unknown provider, got %v", err)
		}
		if len(resp.Matches) != 0 {
			t.Errorf("expected 0 matches for unknown provider, got %d", len(resp.Matches))
		}
	})

	t.Run("nil matches returns empty slice", func(t *testing.T) {
		pNil := &MockProvider{IDVal: "pNil", SearchResults: nil}
		svcNil := NewService(&MockCache{}, pNil)
		resp, err := svcNil.SearchByProviderID(context.Background(), "pNil", "test")
		if err != nil {
			t.Fatalf("SearchByProviderID failed: %v", err)
		}
		if resp.Matches == nil {
			t.Error("Expected matches to be empty slice, got nil")
		}
		if len(resp.Matches) != 0 {
			t.Errorf("Expected 0 matches, got %d", len(resp.Matches))
		}
	})
}

func TestService_Search_PartialFailure(t *testing.T) {
	// Since Search delegates to "all", we test failure on the "all" provider
	failingAll := &MockProvider{
		IDVal:     "all",
		SearchErr: errors.New("failing all"),
	}
	cache := &MockCache{}
	svc := NewService(cache, failingAll)

	_, err := svc.Search(context.Background(), "test")
	if err == nil {
		t.Error("expected error when delegated 'all' search fails")
	}
}

func TestService_SearchProviderWithCache_ZeroTTL(t *testing.T) {
	mock := &MockProvider{
		IDVal:         "all",
		SearchResults: []AbsBookMetadata{{Title: "Cached"}},
		MockCacheTTL:  0,
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

	_, _ = svc.Search(context.Background(), "q")

	mock.SearchErr = context.DeadlineExceeded
	resp, err := svc.Search(context.Background(), "q")
	if err != nil {
		t.Fatalf("cached search failed: %v", err)
	}
	if len(resp.Matches) != 1 || resp.Matches[0].Title != "Cached" {
		t.Errorf("expected cached result, got %+v", resp.Matches)
	}
}
