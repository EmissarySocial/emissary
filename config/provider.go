package config

import (
	"github.com/EmissarySocial/emissary/tools/dataset"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/slice"
)

// Provider represents a single external service provider (typically OAuth2)
type Provider struct {
	ProviderID   string `path:"provider"`
	ClientID     string `path:"clientId"`
	ClientSecret string `path:"clientSecret"`
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

// ProviderSchema returns a schema that validates the Provider object
func ProviderSchema() schema.Schema {

	validProviders := slice.Map(dataset.Providers(), func(provider form.LookupCode) string {
		return provider.Value
	})

	return schema.Schema{
		Element: schema.Object{
			Properties: schema.ElementMap{
				"provider":     schema.String{Required: true, Enum: validProviders, MaxLength: 20},
				"clientId":     schema.String{Required: false, MaxLength: 255},
				"clientSecret": schema.String{Required: false, MaxLength: 255},
			},
		},
	}
}
