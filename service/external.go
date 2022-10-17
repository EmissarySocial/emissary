package service

import (
	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/service/external"
	"github.com/EmissarySocial/emissary/tools/set"
)

type External struct {
	clients set.Slice[config.Provider]
}

func NewExternal(clients []config.Provider) External {
	result := External{}
	result.Refresh(clients)
	return result
}

func (service *External) Refresh(clients []config.Provider) {
	service.clients = clients
}

func (service *External) GetAdapter(providerID string) (external.Adapter, bool) {

	// Create an adapter for known providers
	switch providerID {

	case external.ProviderTypeStripe:
		return external.NewStripe(), true

	case external.ProviderTypeTwitter:
		return external.NewTwitter(), true
	}

	return external.Null{}, false
}
