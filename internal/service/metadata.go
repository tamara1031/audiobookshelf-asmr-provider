package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"audiobookshelf-asmr-provider/internal/domain"
)

// Service orchestrates metadata fetching from multiple providers with caching support.
type Service struct {
	providers []domain.Provider
	cache     *Cache
}

// NewService creates a new metadata service with the given providers.
func NewService(providers ...domain.Provider) *Service {
	return &Service{
		providers: providers,
		cache:     NewCache(),
	}
}

// Providers returns the list of registered providers.
func (s *Service) Providers() []domain.Provider {
	return s.providers
}

// Search queries all registered providers and returns aggregated results.
func (s *Service) Search(ctx context.Context, query string) (*domain.AbsMetadataResponse, error) {
	var allMatches []domain.AbsBookMetadata

	for _, provider := range s.providers {
		matches, err := s.searchProviderWithCache(ctx, provider, query)
		if err != nil {
			log.Printf("Error searching provider %s: %v", provider.ID(), err)
			continue
		}
		allMatches = append(allMatches, matches...)
	}

	return &domain.AbsMetadataResponse{Matches: allMatches}, nil
}

// SearchByProviderID queries a specific provider by its ID.
func (s *Service) SearchByProviderID(ctx context.Context, providerID, query string) (*domain.AbsMetadataResponse, error) {
	provider := s.getProvider(providerID)
	if provider == nil {
		return nil, fmt.Errorf("provider not found: %s", providerID)
	}

	matches, err := s.searchProviderWithCache(ctx, provider, query)
	if err != nil {
		return nil, err
	}

	return &domain.AbsMetadataResponse{Matches: matches}, nil
}

// getProvider helper to find a provider by ID.
func (s *Service) getProvider(id string) domain.Provider {
	for _, p := range s.providers {
		if p.ID() == id {
			return p
		}
	}
	return nil
}

// searchProviderWithCache handles the caching logic for provider searches.
func (s *Service) searchProviderWithCache(ctx context.Context, p domain.Provider, query string) ([]domain.AbsBookMetadata, error) {
	cacheKey := p.ID() + ":" + query

	// Check Cache
	if data, ok := s.cache.Get(cacheKey); ok {
		return data, nil
	}

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
