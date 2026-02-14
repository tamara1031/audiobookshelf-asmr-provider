package service

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"audiobookshelf-asmr-provider/internal/domain"
)

// Service orchestrates metadata fetching from multiple providers with caching support.
type Service struct {
	providers []domain.Provider
	cache     domain.Cache
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

// Search queries all registered providers and returns aggregated results in parallel.
func (s *Service) Search(ctx context.Context, query string) (*domain.AbsMetadataResponse, error) {
	var (
		wg         sync.WaitGroup
		mu         sync.Mutex
		allMatches []domain.AbsBookMetadata
	)

	slog.Info("Starting aggregated search", "query", query, "providers_count", len(s.providers))

	for _, p := range s.providers {
		wg.Add(1)
		go func(p domain.Provider) {
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
