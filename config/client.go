package config

import (
	"github.com/EmissarySocial/emissary/tools/dataset"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/null"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/slice"
)

type Provider struct {
	ProviderID   string `path:"provider"`
	ClientID     string `path:"clientId"`
	ClientSecret string `path:"clientSecret"`
}

// ID implements the set.Value interface
func (provider Provider) ID() string {
	return provider.ProviderID
}

func (provider Provider) IsEmpty() bool {
	return (provider.ClientID == "") && (provider.ClientSecret == "")
}

func ConnectionSchema() schema.Schema {

	validProviders := slice.Map(dataset.Providers(), func(provider form.LookupCode) string {
		return provider.Value
	})

	return schema.Schema{
		Element: schema.Object{
			Properties: schema.ElementMap{
				"provider":     schema.String{Required: true, Enum: validProviders, MaxLength: null.NewInt(20)},
				"clientId":     schema.String{Required: false, MaxLength: null.NewInt(255)},
				"clientSecret": schema.String{Required: false, MaxLength: null.NewInt(255)},
			},
		},
	}
}
