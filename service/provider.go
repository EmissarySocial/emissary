package service

import (
	"github.com/EmissarySocial/emissary/model"
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

	case model.ConnectionProviderGeocodeAddress:
		return providers.NewGeocodeAddress(), true

	case model.ConnectionProviderGeocodeAutocomplete:
		return providers.NewGeocodeAutocomplete(), true

	case model.ConnectionProviderGeocodeNetwork:
		return providers.NewGeocodeNetwork(), true

	case model.ConnectionProviderGeocodeTiles:
		return providers.NewGeocodeTiles(), true

	case model.ConnectionProviderGeocodeTimezone:
		return providers.NewGeocodeTimezone(), true

	case model.ConnectionProviderGiphy:
		return providers.NewGiphy(), true

	// case model.ConnectionProviderStripe:
	//	return providers.NewStripe(), true

	case model.ConnectionProviderStripeConnect:
		return providers.NewStripeConnect(), true

	case model.ConnectionProviderUnsplash:
		return providers.NewUnsplash(), true
	}

	return providers.Null{}, false
}
