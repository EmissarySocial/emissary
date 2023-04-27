package config

// Provider represents a single external service provider (typically OAuth2)
type Provider struct {
	ProviderID   string `json:"providerId"   bson:"providerId"`   // Unique identifier for this provider
	ClientID     string `json:"clientId"     bson:"clientId"`     // Client ID for this provider
	ClientSecret string `json:"clientSecret" bson:"clientSecret"` // Client Secret for this provider
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
