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

	case model.ConnectionProviderArcGIS:
		return providers.NewArcGIS(), true

	case model.ConnectionProviderFREEIPAPICOM:
		return providers.NewFREEIPAPICOM(), true

	case model.ConnectionProviderGeoapify:
		return providers.NewGeoapify(), true

	case model.ConnectionProviderGiphy:
		return providers.NewGiphy(), true

	case model.ConnectionProviderGoogleMaps:
		return providers.NewGoogleMaps(), true

	case model.ConnectionProviderNominatim:
		return providers.NewNominatim(), true

	case model.ConnectionProviderIPAPICO:
		return providers.NewIPAPICO(), true

	case model.ConnectionProviderIPAPICOM:
		return providers.NewIPAPICOM(), true

	case model.ConnectionProviderOpenStreetMap:
		return providers.NewOpenStreetMap(), true

	case model.ConnectionProviderStaticGeocoderIP:
		return providers.NewStaticGeocoder(), true

	case model.ConnectionProviderStripe:
		return providers.NewStripe(), true

	case model.ConnectionProviderStripeConnect:
		return providers.NewStripeConnect(), true

	case model.ConnectionProviderUnsplash:
		return providers.NewUnsplash(), true
	}

	return providers.Null{}, false
}
