package provider

import (
	"audiobookshelf-asmr-provider/internal/domain/provider/dlsite"
	"audiobookshelf-asmr-provider/internal/service"
)

// NewAll instantiates and returns all available providers.
// Add new providers here when implementing them.
func NewAll() []service.Provider {
	return []service.Provider{
		dlsite.NewDLsiteFetcher(),
	}
}
