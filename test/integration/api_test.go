package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	httphandler "audiobookshelf-asmr-provider/internal/adapter/http"
	"audiobookshelf-asmr-provider/internal/domain"
	"audiobookshelf-asmr-provider/internal/service"
)

// MockProvider for integration test
type MockProvider struct {
	IDVal         string
	SearchResults []domain.AbsBookMetadata
	SearchErr     error
}

func (m *MockProvider) ID() string {
	return m.IDVal
}

func (m *MockProvider) Search(ctx context.Context, query string) ([]domain.AbsBookMetadata, error) {
	return m.SearchResults, m.SearchErr
}

func (m *MockProvider) CacheTTL() time.Duration {
	return 1 * time.Hour
}

func TestAPI_Search_Integration(t *testing.T) {
	// 1. Setup Dependencies
	mockData := []domain.AbsBookMetadata{
		{
			Title:     "Integration Test Title",
			ISBN:      "RJ123456",
			Publisher: "DLsite",
		},
	}
	mockProvider := &MockProvider{
		IDVal:         "dlsite",
		SearchResults: mockData,
	}

	svc := service.NewService(mockProvider)
	handler := httphandler.NewHandler(svc)

	// 2. Setup Test Server (The API we are testing)
	mux := http.NewServeMux()
	// Register the routes as Main does
	mux.HandleFunc("/api/search", handler.Search)
	mux.HandleFunc("/api/dlsite/search", func(w http.ResponseWriter, r *http.Request) {
		handler.SearchSingle(w, r, "dlsite")
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	// 3. Execute Request against the Test Server
	// A. Test Aggregated Search
	resp, err := http.Get(server.URL + "/api/search?q=RJ123456")
	if err != nil {
		t.Fatalf("Failed to make GET request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result domain.AbsMetadataResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode JSON: %v", err)
	}

	if len(result.Matches) != 1 {
		t.Fatalf("Expected 1 match, got %d", len(result.Matches))
	}
	if result.Matches[0].Title != "Integration Test Title" {
		t.Errorf("Expected title 'Integration Test Title', got '%s'", result.Matches[0].Title)
	}

	// B. Test Provider Specific Search
	resp2, err := http.Get(server.URL + "/api/dlsite/search?q=RJ123456")
	if err != nil {
		t.Fatalf("Failed to make GET request: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp2.StatusCode)
	}
}
