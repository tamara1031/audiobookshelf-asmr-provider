package provider

import (
	"audiobookshelf-asmr-provider/internal/adapter/provider/dlsite"
	"audiobookshelf-asmr-provider/internal/domain"
)

// NewAll instantiates and returns all available providers.
// Add new providers here when implementing them.
func NewAll() []domain.Provider {
	return []domain.Provider{
		dlsite.NewDLsiteFetcher(),
	}
}
