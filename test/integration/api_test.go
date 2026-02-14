package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"audiobookshelf-asmr-provider/internal/handler"
	"audiobookshelf-asmr-provider/internal/service"
)

// MockProvider for integration test
type MockProvider struct {
	IDVal         string
	SearchResults []service.AbsBookMetadata
	SearchErr     error
}

func (m *MockProvider) ID() string {
	return m.IDVal
}

func (m *MockProvider) Search(_ context.Context, _ string) ([]service.AbsBookMetadata, error) {
	return m.SearchResults, m.SearchErr
}

func (m *MockProvider) CacheTTL() time.Duration {
	return 1 * time.Hour
}

// integrationCache implements service.Cache for integration tests.
type integrationCache struct{}

func (i *integrationCache) Get(_ string) ([]service.AbsBookMetadata, bool)             { return nil, false }
func (i *integrationCache) Put(_ string, _ []service.AbsBookMetadata, _ time.Duration) {}

func TestAPI_Search_Integration(t *testing.T) {
	mockData := []service.AbsBookMetadata{{Title: "Integration Test Title", ISBN: "RJ123456"}}
	dlsite := &MockProvider{IDVal: "dlsite", SearchResults: mockData}
	all := &MockProvider{IDVal: "all", SearchResults: mockData}

	svc := service.NewService(&integrationCache{}, dlsite, all)
	h := handler.NewHandler(svc)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/search", h.Search)
	mux.HandleFunc("GET /api/{provider}/search", h.Search)

	server := httptest.NewServer(mux)
	defer server.Close()

	// A. Test Aggregated Search (defaults to "all")
	resp, err := http.Get(server.URL + "/api/search?q=RJ123456")
	if err != nil {
		t.Fatalf("Failed to make GET request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var body map[string]string
		_ = json.NewDecoder(resp.Body).Decode(&body)
		t.Fatalf("Expected status 200, got %d (body: %+v)", resp.StatusCode, body)
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
