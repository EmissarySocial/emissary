package service

import (
	"github.com/EmissarySocial/emissary/service/providers"
)

// Provider service manages all access to external services
type Provider struct {
}

// NewProvider returns a fully initialized Provider service
func NewProvider() Provider {
	return Provider{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates the list of clients
func (service *Provider) Refresh() {
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

	case providers.ProviderTypeGiphy:
		return providers.NewGiphy(), true

	case providers.ProviderTypeGoogleMaps:
		return providers.NewGoogleMaps(), true

	case providers.ProviderTypeOpenStreetMap:
		return providers.NewOpenStreetMap(), true

	case providers.ProviderTypePayPal:
		return providers.NewPayPal(), true

	case providers.ProviderTypeStripe:
		return providers.NewStripe(), true

	case providers.ProviderTypeUnsplash:
		return providers.NewUnsplash(), true
	}

	return providers.Null{}, false
}
