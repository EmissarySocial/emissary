package service

import (
	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/service/providers"
	"github.com/EmissarySocial/emissary/tools/set"
)

// Provider service manages all access to external services
type Provider struct {
	config set.Slice[config.Provider]
}

// NewProvider returns a fully initialized Provider service
func NewProvider(providers []config.Provider) Provider {
	result := Provider{}
	result.Refresh(providers)
	return result
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates the list of clients
func (service *Provider) Refresh(providers []config.Provider) {
	service.config = providers
}

/******************************************
 * Other Methods
 ******************************************/

// GetProvider returns a populated adapter for the given provider
func (service *Provider) GetProvider(providerID string) (providers.Provider, bool) {

	// Create an adapter for known providers
	switch providerID {

	case providers.ProviderTypeArcGIS:
		return providers.NewArcGIS(), true

	case providers.ProviderTypeGoogleMaps:
		return providers.NewGoogleMaps(), true

	case providers.ProviderTypeOpenStreetMap:
		return providers.NewOpenStreetMap(), true

	case providers.ProviderTypeGiphy:
		return providers.NewGiphy(), true

	case providers.ProviderTypeStripe:
		return providers.NewStripe(), true

	case providers.ProviderTypeUnsplash:
		return providers.NewUnsplash(), true
	}

	return providers.Null{}, false
}
