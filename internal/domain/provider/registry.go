package provider

import (
	"audiobookshelf-asmr-provider/internal/domain/provider/all"
	"audiobookshelf-asmr-provider/internal/domain/provider/dlsite"
	"audiobookshelf-asmr-provider/internal/domain/provider/void"
	"audiobookshelf-asmr-provider/internal/service"
)

// NewAll instantiates and returns all available providers.
func NewAll() []service.Provider {
	dlsiteProvider := dlsite.NewDLsiteFetcher()
	allProvider := all.NewProvider(dlsiteProvider)
	voidProvider := void.NewProvider()

	return []service.Provider{
		dlsiteProvider,
		allProvider,
		voidProvider,
	}
}
