package providers

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/domain"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/schema"
	"github.com/stripe/stripe-go/v72"
)

const ProviderTypeStripe = "STRIPE"

const Stripe_APIKey = "apiKey"
const Stripe_WebhookSecret = "webhookSecret"

type Stripe struct{}

func NewStripe() Stripe {
	return Stripe{}
}

/******************************************
 * Setup / Configuration Methods
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
							"webhookSecret": schema.String{Required: false},
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
					Type:  "toggle",
					Path:  "active",
					Label: "Enable?",
				},
			},
		},
	}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// AfterCoonnect applies any extra changes to the database after this Adapter is activated.
func (adapter Stripe) AfterConnect(factory Factory, client *model.Client) error {

	const location = "service.external.Stripe.Install"

	hostname := factory.Hostname()

	// Can't set up WebHooks on localhost
	if domain.IsLocalhost(hostname) {
		return nil
	}

	// Verify that webhooks have been set up on this domain
	if secret, _ := client.GetStringOK(Stripe_WebhookSecret); secret == "" {

		api, err := factory.StripeClient()

		if err != nil {
			return derp.Wrap(err, location, "Error getting Stripe client")
		}

		// Configure webhook
		webhookURL := "https://" + hostname + "/webhooks/stripe"
		params := stripe.WebhookEndpointParams{
			URL: stripe.String(webhookURL),
			EnabledEvents: []*string{
				stripe.String("checkout.session.completed"),
			},
		}

		// Create webhook
		webhook, err := api.WebhookEndpoints.New(&params)

		if err != nil {
			return derp.Wrap(err, location, "Error creating endpoint")
		}

		// Mark webhook as installed
		client.SetString(Stripe_WebhookSecret, webhook.Secret)
	}

	return nil
}

// AfterUpdate is called after a user has successfully updated their Twitter connection
func (adapter Stripe) AfterUpdate(factory Factory, client *model.Client) error {
	return nil
}

/******************************************
 * Adapter Methods
 ******************************************/

func (adapter Stripe) PollStreams(client *model.Client) <-chan model.Stream {
	return nil
}
