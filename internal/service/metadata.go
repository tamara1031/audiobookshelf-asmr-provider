package service

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// AbsMetadataResponse represents the top-level JSON response expected by Audiobookshelf.
type AbsMetadataResponse struct {
	Matches []AbsBookMetadata `json:"matches"`
}

// AbsBookMetadata matches the JSON structure for a book/work in the ABS custom provider API.
type AbsBookMetadata struct {
	Title         string   `json:"title"`
	Subtitle      string   `json:"subtitle,omitempty"`
	Author        string   `json:"author"`
	Narrator      string   `json:"narrator,omitempty"`
	Series        string   `json:"series,omitempty"`
	Description   string   `json:"description,omitempty"`
	Publisher     string   `json:"publisher,omitempty"`
	PublishedYear string   `json:"publishedYear,omitempty"`
	Genres        []string `json:"genres,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	Cover         string   `json:"cover,omitempty"`
	ISBN          string   `json:"isbn,omitempty"`
	ASIN          string   `json:"asin,omitempty"`
	Language      string   `json:"language,omitempty"`
	Explicit      bool     `json:"explicit,omitempty"`
}

// Provider defines the interface for a metadata provider plugin.
type Provider interface {
	// ID returns the unique identifier of the provider (e.g., "dlsite").
	ID() string

	// Search searches for works matching the query and returns ABS-compatible metadata.
	Search(ctx context.Context, query string) ([]AbsBookMetadata, error)

	// CacheTTL returns the duration for which results should be cached.
	CacheTTL() time.Duration
}

// Cache defines the interface for a metadata cache.
type Cache interface {
	Get(key string) ([]AbsBookMetadata, bool)
	Put(key string, data []AbsBookMetadata, ttl time.Duration)
}

// Service orchestrates metadata fetching from multiple providers with caching support.
type Service struct {
	providers []Provider
	cache     Cache
}

// NewService creates a new metadata service with the given providers and cache implementation.
func NewService(cache Cache, providers ...Provider) *Service {
	return &Service{
		providers: providers,
		cache:     cache,
	}
}

// Providers returns the list of registered providers.
func (s *Service) Providers() []Provider {
	return s.providers
}

// Search queries all registered providers and returns aggregated results in parallel.
func (s *Service) Search(ctx context.Context, query string) (*AbsMetadataResponse, error) {
	var (
		wg         sync.WaitGroup
		mu         sync.Mutex
		allMatches []AbsBookMetadata
	)

	slog.Info("Starting aggregated search", "query", query, "providers_count", len(s.providers))

	for _, p := range s.providers {
		wg.Add(1)
		go func(p Provider) {
			defer wg.Done()
			matches, err := s.searchProviderWithCache(ctx, p, query)
			if err != nil {
				slog.Error("Provider search failed", "provider", p.ID(), "error", err)
				return
			}

			mu.Lock()
			allMatches = append(allMatches, matches...)
			mu.Unlock()
		}(p)
	}

	wg.Wait()

	return &AbsMetadataResponse{Matches: allMatches}, nil
}

// SearchByProviderID queries a specific provider by its ID.
func (s *Service) SearchByProviderID(ctx context.Context, providerID, query string) (*AbsMetadataResponse, error) {
	provider := s.getProvider(providerID)
	if provider == nil {
		return nil, fmt.Errorf("provider not found: %s", providerID)
	}

	matches, err := s.searchProviderWithCache(ctx, provider, query)
	if err != nil {
		return nil, err
	}

	return &AbsMetadataResponse{Matches: matches}, nil
}

// getProvider helper to find a provider by ID.
func (s *Service) getProvider(id string) Provider {
	for _, p := range s.providers {
		if p.ID() == id {
			return p
		}
	}
	return nil
}

// searchProviderWithCache handles the caching logic for provider searches.
func (s *Service) searchProviderWithCache(ctx context.Context, p Provider, query string) ([]AbsBookMetadata, error) {
	cacheKey := p.ID() + ":" + query

	// Check Cache
	if data, ok := s.cache.Get(cacheKey); ok {
		slog.Debug("Cache hit", "provider", p.ID(), "query", query)
		return data, nil
	}

	slog.Debug("Fetching from provider", "provider", p.ID(), "query", query)

	// Fetch from Provider
	matches, err := p.Search(ctx, query)
	if err != nil {
		return nil, err
	}

	// Save to Cache
	ttl := p.CacheTTL()
	if ttl == 0 {
		ttl = 1 * time.Hour
	}
	s.cache.Put(cacheKey, matches, ttl)

	return matches, nil
}
