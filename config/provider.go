package config

// Provider represents a single external service provider (typically OAuth2)
type Provider struct {
	ProviderID   string
	ClientID     string
	ClientSecret string
}

// NewProvider returns a fully initialized Provider object
func NewProvider(providerID string) Provider {
	return Provider{
		ProviderID: providerID,
	}
}

// ID implements the set.Value interface
func (provider Provider) ID() string {
	return provider.ProviderID
}

// IsEmpty returns TRUE if the provider is empty
func (provider Provider) IsEmpty() bool {
	return (provider.ClientID == "") && (provider.ClientSecret == "")
}
