package providers

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	dt "github.com/benpate/domain"
	"github.com/benpate/form"
	"github.com/benpate/remote"
	"github.com/benpate/remote/options"
	"github.com/benpate/rosetta/mapof"
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
							"liveMode": schema.String{Enum: []string{"false", "true"}},
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
					Path:  "data.liveMode",
					Label: "Live Mode?",
					Options: mapof.Any{
						"enum": []form.LookupCode{
							{Value: "SANDBOX", Label: "Sandbox (Use for Tests Only)"},
							{Value: "LIVE", Label: "Live. (Use for Real Payments)"},
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
func (adapter StripeConnect) Connect(connection *model.Connection, vault mapof.String, host string) error {

	const location = "providers.StripeConnect.Connect"

	// RULE: Cannot set webhooks for local domains
	if dt.IsLocalhost(host) {
		return nil
	}

	// RULE: If we already have a webhook for this MerchantAccount, then don't add another one.
	if connection.Data.GetString("webhook") != "" {
		return nil
	}

	// Configure a new Webhook in the Stripe API
	webhookResult := mapof.NewAny()
	txn := remote.Post("https://api.stripe.com/v1/webhook_endpoints").
		With(options.BearerAuth(vault.GetString("restrictedKey"))).
		// With(options.Debug()).
		Query("url", host+"/.stripe-connect/webhook/checkout").
		Query("description", dt.NameOnly(host)+" supscription updates").
		Query("enabled_events[]", "checkout.session.completed").
		Query("enabled_events[]", "customer.subscription.created").
		Query("enabled_events[]", "customer.subscription.deleted").
		Query("enabled_events[]", "customer.subscription.paused").
		Query("enabled_events[]", "customer.subscription.resumed").
		Query("enabled_events[]", "customer.subscription.updated").
		Query("connect", "true").
		Result(&webhookResult)

	if err := txn.Send(); err != nil {
		return derp.Wrap(err, location, "Error creating WebHook via Stripe API")
	}

	// Save the webhook data into the MerchantAccount
	connection.Data.SetString("webhook", webhookResult.GetString("id"))
	connection.Vault.SetString("webhookSecret", webhookResult.GetString("secret"))

	// Success!
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
