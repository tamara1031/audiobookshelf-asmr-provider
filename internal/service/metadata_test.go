package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"audiobookshelf-asmr-provider/internal/domain"
)

// MockProvider implements domain.Provider for testing.
type MockProvider struct {
	IDVal         string
	SearchResults []domain.AbsBookMetadata
	SearchErr     error
	MockCacheTTL  time.Duration
}

func (m *MockProvider) ID() string { return m.IDVal }

func (m *MockProvider) Search(_ context.Context, _ string) ([]domain.AbsBookMetadata, error) {
	return m.SearchResults, m.SearchErr
}

func (m *MockProvider) CacheTTL() time.Duration { return m.MockCacheTTL }

func TestService_Search(t *testing.T) {
	mockData := []domain.AbsBookMetadata{
		{Title: "Test Book", ISBN: "RJ123456"},
	}
	mockProvider := &MockProvider{
		IDVal:         "test_provider",
		SearchResults: mockData,
		MockCacheTTL:  1 * time.Hour,
	}

	svc := NewService(mockProvider)

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
		SearchResults: []domain.AbsBookMetadata{{Title: "A"}},
	}
	svc := NewService(mockProvider)

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
	svc := NewService(p1, p2)

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
		SearchResults: []domain.AbsBookMetadata{{Title: "OK"}},
		MockCacheTTL:  1 * time.Hour,
	}
	svc := NewService(failing, working)

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
		SearchResults: []domain.AbsBookMetadata{{Title: "Cached"}},
		MockCacheTTL:  0, // zero TTL
	}
	svc := NewService(mock)

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
