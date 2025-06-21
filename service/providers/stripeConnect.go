package providers

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/null"
	"github.com/benpate/rosetta/schema"
)

const ProviderTypeStripeConnect = "STRIPE-CONNECT"

type StripeConnect struct{}

func NewStripeConnect() StripeConnect {
	return StripeConnect{}
}

/******************************************
 * Setup / Configuration Methods
 ******************************************/

func (adapter StripeConnect) ManualConfig() form.Form {

	return form.Form{
		Schema: schema.Schema{
			Element: schema.Object{
				Properties: schema.ElementMap{
					"type":   schema.String{Enum: []string{"PAYMENT"}},
					"active": schema.Boolean{},
					"data": schema.Object{
						Properties: schema.ElementMap{
							"clientId": schema.String{Required: true},
							"liveMode": schema.Boolean{Default: null.NewBool(false)},
						},
					},
					"vault": schema.Object{
						Properties: schema.ElementMap{
							"publishableKey": schema.String{Required: true, Pattern: "^(\\**)|(pk_(test|live)_[A-Za-z0-9]+)"},
							"restrictedKey":  schema.String{Required: true, Pattern: "^(\\**)|(rk_(test|live)_[A-Za-z0-9]+)"},
						},
					},
				},
			},
		},
		Element: form.Element{
			Type:        "layout-vertical",
			Label:       "Stripe Connect Setup",
			Description: "Allows users to use their own Stripe accounts via OAuth. This application must be registered with Stripe Connect.",
			Children: []form.Element{
				{
					Type:    "hidden",
					Path:    "type",
					Options: mapof.Any{"value": "USER-PAYMENT"},
				},
				{
					Type:        "text",
					Path:        "data.clientId",
					Label:       "Client ID",
					Description: "Found in the <a href='https://dashboard.stripe.com/test/settings/connect/onboarding-options/oauth' target='_blank'>Stripe Connect OAuth Settings &rarr;</a>.",
					Options: mapof.Any{
						"autocomplete": "off",
						"spellcheck":   false,
					},
				},
				{
					Type:        "text",
					Path:        "vault.publishableKey",
					Label:       "Publishable Key",
					Description: "Found in the <a href='https://dashboard.stripe.com/apikeys' target='_blank'>Stripe Dashboard &rarr;</a>.",
					Options: mapof.Any{
						"placeholder":  "pk_live_XXXXXXXXXXXXXXXXXXXXXXXXX",
						"autocomplete": "off",
						"spellcheck":   false,
					},
				},
				{
					Type:        "text",
					Path:        "vault.restrictedKey",
					Label:       "Restricted Key",
					Description: "Found in the <a href='https://dashboard.stripe.com/apikeys' target='_blank'>Stripe Dashboard &rarr;</a>.",
					Options: mapof.Any{
						"placeholder":  "rk_live_XXXXXXXXXXXXXXXXXXXXXXXXX",
						"autocomplete": "off",
						"spellcheck":   false,
					},
				},
				{
					Type:  "select",
					Path:  "liveMode",
					Label: "Live Mode?",
					Options: mapof.Any{
						"enum": []form.LookupCode{
							{Value: "false", Label: "Sandbox (Use for Tests Only)"},
							{Value: "true", Label: "Live. (Use for Real Payments)"},
						},
					},
				},
				{
					Type: "toggle",
					Path: "active",
					Options: mapof.Any{
						"true-text":  "Enabled. Users can connect their Stripe accounts",
						"false-text": "Enable?",
					},
				},
			},
		},
	}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Connect applies any extra changes to the database after this Adapter is activated.
func (adapter StripeConnect) Connect(connection *model.Connection, vault mapof.String) error {
	return nil
}

// Refresh updates this connection if it has changed or is out of date
func (adapter StripeConnect) Refresh(connection *model.Connection, vault mapof.String) error {
	return nil
}

// Disconnect applies any extra changes to the database when this Adapter is disconnected
func (adapter StripeConnect) Disconnect(connection *model.Connection, vault mapof.String) error {
	return nil
}
