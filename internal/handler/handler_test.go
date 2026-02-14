package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"audiobookshelf-asmr-provider/internal/service"
)

// mockCache implements service.Cache for testing.
type mockCache struct{}

func (m *mockCache) Get(_ string) ([]service.AbsBookMetadata, bool)             { return nil, false }
func (m *mockCache) Put(_ string, _ []service.AbsBookMetadata, _ time.Duration) {}

// mockProvider implements service.Provider for testing.
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

func TestSearch_WithQueryParam(t *testing.T) {
	mock := &mockProvider{
		id:      "test",
		results: []service.AbsBookMetadata{{Title: "Result", ISBN: "RJ123456"}},
	}
	svc := service.NewService(&mockCache{}, mock)
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/search?q=RJ123456", nil)
	rec := httptest.NewRecorder()

	h.Search(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp service.AbsMetadataResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Matches) != 1 || resp.Matches[0].Title != "Result" {
		t.Errorf("unexpected matches: %+v", resp.Matches)
	}
}

func TestSearch_WithQueryFallbackParam(t *testing.T) {
	mock := &mockProvider{
		id:      "test",
		results: []service.AbsBookMetadata{{Title: "Fallback"}},
	}
	svc := service.NewService(&mockCache{}, mock)
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/search?query=test", nil)
	rec := httptest.NewRecorder()

	h.Search(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestSearch_MissingQuery(t *testing.T) {
	svc := service.NewService(&mockCache{})
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/search", nil)
	rec := httptest.NewRecorder()

	h.Search(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestSearch_ProviderError(t *testing.T) {
	mock := &mockProvider{
		id:  "test",
		err: errors.New("provider failure"),
	}
	svc := service.NewService(&mockCache{}, mock)
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/search?q=test", nil)
	rec := httptest.NewRecorder()

	h.Search(rec, req)

	// Search aggregates errors — it logs and continues, so it returns 200 with empty matches.
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 (aggregated search skips errors), got %d", rec.Code)
	}
}

func TestSearchSingle_ValidQuery(t *testing.T) {
	mock := &mockProvider{
		id:      "dlsite",
		results: []service.AbsBookMetadata{{Title: "DLsite Result"}},
	}
	svc := service.NewService(&mockCache{}, mock)
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/dlsite/search?q=RJ123456", nil)
	rec := httptest.NewRecorder()

	h.SearchSingle(rec, req, "dlsite")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp service.AbsMetadataResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Matches) != 1 || resp.Matches[0].Title != "DLsite Result" {
		t.Errorf("unexpected matches: %+v", resp.Matches)
	}
}

func TestSearchSingle_MissingQuery(t *testing.T) {
	svc := service.NewService(&mockCache{})
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/dlsite/search", nil)
	rec := httptest.NewRecorder()

	h.SearchSingle(rec, req, "dlsite")

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestSearchSingle_UnknownProvider(t *testing.T) {
	svc := service.NewService(&mockCache{})
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/unknown/search?q=test", nil)
	rec := httptest.NewRecorder()

	h.SearchSingle(rec, req, "unknown")

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}
