package void

import (
	"context"
	"time"

	"audiobookshelf-asmr-provider/internal/service"
)

// Provider is a fallback provider that returns no results.
type Provider struct{}

// NewProvider creates a new void provider instance.
func NewProvider() *Provider {
	return &Provider{}
}

// ID returns the unique identifier for this provider.
func (p *Provider) ID() string {
	return "void"
}

// Search returns an empty slice of metadata and no error.
func (p *Provider) Search(_ context.Context, _ string) ([]service.AbsBookMetadata, error) {
	return []service.AbsBookMetadata{}, nil
}

// CacheTTL returns the duration for which results should be cached.
func (p *Provider) CacheTTL() time.Duration {
	return 24 * time.Hour
}
