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

	case providers.ProviderTypeGiphy:
		return service.GetGiphyProvider(), true
	}

	return providers.Null{}, false
}

func (service *Provider) GetGiphyProvider() providers.Giphy {
	return providers.NewGiphy()
}

/* REMOVED FOR NOW
// GetTwitterProvider returns a populated Twitter adapter
func (service *Provider) GetTwitterProvider() providers.Twitter {
	config, _ := service.config.Get(providers.ProviderTypeTwitter)
	return providers.NewTwitter(config)
}
*/
