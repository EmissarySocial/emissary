package external

import (
	"github.com/benpate/form"
	"github.com/benpate/rosetta/schema"
)

const ProviderTypeStripe = "STRIPE"

const StripeData_APIKey = "apiKey"
const StripeData_WebhookSecret = "webhookSecret"

type Stripe struct{}

func NewStripe() Stripe {
	return Stripe{}
}

/******************************************
 * Manual API Methods
 ******************************************/

func (adapter Stripe) ManualConfig() form.Form {

	return form.Form{
		Schema: schema.Schema{
			Element: schema.Object{
				Properties: schema.ElementMap{
					"active": schema.Boolean{},
					"data": schema.Object{
						Properties: schema.ElementMap{
							"apiKey":        schema.String{Required: true},
							"webhookSecret": schema.String{Required: true},
						},
					},
				},
			},
		},
		Element: form.Element{
			Type:        "layout-vertical",
			Label:       "Stripe Setup",
			Description: "Sign into your Stripe account and create an API key.  Then, paste the API key into the field below.",
			Children: []form.Element{
				{
					Type:  "text",
					Path:  "data.apiKey",
					Label: "API Key",
				},
				{
					Type:  "text",
					Path:  "data.webhookSecret",
					Label: "Webhook Secret",
				},
				{
					Type:  "toggle",
					Path:  "active",
					Label: "Enable?",
				},
			},
		},
	}
}

/* OAuth (removed)

func (adapter Stripe) OAuthConfig() oauth2.Config {

	return oauth2.Config{
		ClientID:     adapter.configuration.ClientID,
		ClientSecret: adapter.configuration.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:   "https://connect.stripe.com/oauth/authorize",
			TokenURL:  "https://connect.stripe.com/oauth/token",
			AuthStyle: oauth2.AuthStyleInParams,
		},
		Scopes: []string{},
	}
}

******************************************/

/******************************************
 * Adapter Methods
 ******************************************/

func (adapter Stripe) Install() {

}

func (adapter Stripe) PollStreams() {
}

func (adapter Stripe) PostStream() {

}
