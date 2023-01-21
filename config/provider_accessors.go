package config

import (
	"github.com/EmissarySocial/emissary/tools/dataset"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/slice"
)

// ProviderSchema returns a schema that validates the Provider object
func ProviderSchema() schema.Element {

	validProviders := slice.Map(dataset.Providers(), func(provider form.LookupCode) string {
		return provider.Value
	})

	return schema.Object{
		Properties: schema.ElementMap{
			"provider":     schema.String{Required: true, Enum: validProviders, MaxLength: 20},
			"clientId":     schema.String{Required: false, MaxLength: 255},
			"clientSecret": schema.String{Required: false, MaxLength: 255},
		},
	}
}

func (provider Provider) GetStringOK(key string) (string, bool) {

	switch key {

	case "providerId":
		return provider.ProviderID, true

	case "clientId":
		return provider.ClientID, true

	case "clientSecret":
		return provider.ClientSecret, true

	}

	return "", false
}

func (provider *Provider) SetStringOK(key string, value string) bool {

	switch key {

	case "providerId":
		provider.ProviderID = value
		return true

	case "clientId":
		provider.ClientID = value
		return true

	case "clientSecret":
		provider.ClientSecret = value
		return true

	}

	return false
}
