package service

import (
	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/service/external"
	"github.com/EmissarySocial/emissary/tools/set"
)

// External service manages all access to external services
type External struct {
	clients set.Slice[config.Provider]
}

// NewExternal returns a fully initialized External service
func NewExternal(clients []config.Provider) External {
	result := External{}
	result.Refresh(clients)
	return result
}

/*******************************************
 * Lifecycle Methods
 *******************************************/

// Refresh updates the list of clients
func (service *External) Refresh(clients []config.Provider) {
	service.clients = clients
}

/*******************************************
 * Other Methods
 *******************************************/

// Provider returns the provider for the given ID
func (service *External) Provider(providerID string) config.Provider {

	// If the provider exists in the configuration, then return it
	if provider, ok := service.clients.Get(providerID); ok {
		return provider
	}

	// Otherwise, make a new provider with default values
	return config.NewProvider(providerID)
}

// GetAdapter returns a populated adapter for the given provider
func (service *External) GetAdapter(providerID string) (external.Adapter, bool) {

	// Create an adapter for known providers
	switch providerID {

	case external.ProviderTypeStripe:
		return service.GetStripeAdapter(), true

	case external.ProviderTypeTwitter:
		return service.GetTwitterAdapter(), true
	}

	return external.Null{}, false
}

// GetStripeAdapter returns a populated Stripe adapter
func (service *External) GetStripeAdapter() external.Stripe {
	return external.NewStripe()
}

// GetTwitterAdapter returns a populated Twitter adapter
func (service *External) GetTwitterAdapter() external.Twitter {
	provider, _ := service.clients.Get(external.ProviderTypeTwitter)
	return external.NewTwitter(provider)
}
