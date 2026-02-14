package all

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"audiobookshelf-asmr-provider/internal/service"
)

// Provider implements the service.Provider interface by aggregating results from multiple providers.
type Provider struct {
	providers []service.Provider
}

// NewProvider creates a new aggregation provider with the given sub-providers.
func NewProvider(providers ...service.Provider) *Provider {
	return &Provider{
		providers: providers,
	}
}

// ID returns the unique identifier for this provider.
func (p *Provider) ID() string {
	return "all"
}

// Search queries all registered providers in parallel and aggregates their results.
func (p *Provider) Search(ctx context.Context, query string) ([]service.AbsBookMetadata, error) {
	var (
		wg         sync.WaitGroup
		mu         sync.Mutex
		allMatches []service.AbsBookMetadata
	)

	slog.Info("Starting aggregated search in AllProvider", "query", query, "providers_count", len(p.providers))

	for _, provider := range p.providers {
		wg.Add(1)
		go func(pr service.Provider) {
			defer wg.Done()
			matches, err := pr.Search(ctx, query)
			if err != nil {
				slog.Error("Provider search failed in AllProvider", "provider", pr.ID(), "error", err)
				return
			}

			mu.Lock()
			allMatches = append(allMatches, matches...)
			mu.Unlock()
		}(provider)
	}

	wg.Wait()

	return allMatches, nil
}

// CacheTTL returns the duration for which results should be cached.
func (p *Provider) CacheTTL() time.Duration {
	return 1 * time.Hour
}
