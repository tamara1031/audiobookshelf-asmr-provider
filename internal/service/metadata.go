package service

import (
	"context"
	"log/slog"
	"time"
)

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

// Search queries all registered providers by delegating to the "all" provider.
func (s *Service) Search(ctx context.Context, query string) (*AbsMetadataResponse, error) {
	return s.SearchByProviderID(ctx, "all", query)
}

// SearchByProviderID queries a specific provider by its ID.
func (s *Service) SearchByProviderID(ctx context.Context, providerID, query string) (*AbsMetadataResponse, error) {
	p := s.getProvider(providerID)
	if p == nil {
		// Provider not found, return valid empty result (void behavior)
		return &AbsMetadataResponse{Matches: []AbsBookMetadata{}}, nil
	}

	matches, err := s.searchProviderWithCache(ctx, p, query)
	if err != nil {
		return nil, err
	}

	if matches == nil {
		matches = []AbsBookMetadata{}
	}
	return &AbsMetadataResponse{Matches: matches}, nil

}

// getProvider helper to find a provider by ID. If not found, returns nil.
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

	slog.Debug("Provider response", "provider", p.ID(), "count", len(matches), "results", matches)

	// Save to Cache
	ttl := p.CacheTTL()
	if ttl == 0 {
		ttl = 1 * time.Hour
	}
	s.cache.Put(cacheKey, matches, ttl)

	return matches, nil
}
