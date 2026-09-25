package providers

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/remote"
	"github.com/benpate/remote/options"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/uri"
)

// stripeAPIBase is the root of every Stripe API call this provider makes
const stripeAPIBase = "https://api.stripe.com"

// StripeConnect connects a Domain to the Stripe Connect payment integration, which processes payments on behalf of a merchant
type StripeConnect struct {
	apiBase         string // root of the Stripe API; only tests change it
	allowPrivateIPs bool   // lets a test reach a loopback server; never set in production
}

// NewStripeConnect returns a fully initialized StripeConnect provider
func NewStripeConnect() StripeConnect {
	return StripeConnect{apiBase: stripeAPIBase}
}

// endpoint returns the full URL of a Stripe API path
func (adapter StripeConnect) endpoint(path string) string {

	// A zero-value StripeConnect still calls the real Stripe API
	if adapter.apiBase == "" {
		return stripeAPIBase + path
	}

	return adapter.apiBase + path
}

/******************************************
 * Setup / Configuration Methods
 ******************************************/

// ManualConfig returns the form used to configure this Connection by hand. Implements the ManualProvider interface.
func (adapter StripeConnect) ManualConfig() form.Form {

	return form.Form{
		Schema: schema.Schema{
			Element: schema.Object{
				Properties: schema.ElementMap{
					"type":   schema.String{Enum: []string{model.ConnectionTypeUserPayment}},
					"active": schema.Boolean{},
					"data": schema.Object{
						Properties: schema.ElementMap{
							"clientId": schema.String{Required: true},
							"liveMode": schema.String{Enum: []string{"SANDBOX", "LIVE"}},
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
			Label:       "<i class='bi bi-stripe'></i> Stripe Connect Setup",
			Description: "Allows users to use their own Stripe accounts via OAuth. This application must be registered with Stripe Connect.",
			Children: []form.Element{
				{
					Type:    "hidden",
					Path:    "type",
					Options: mapof.Any{"value": model.ConnectionTypeUserPayment},
				},
				{
					Type:        "text",
					Path:        "data.clientId",
					Label:       "Client ID",
					Description: "Found in the <a href='https://dashboard.stripe.com/test/settings/connect/onboarding-options/oauth' target='_blank' rel='noopener noreferrer'>Stripe Connect OAuth Settings &rarr;</a>.",
					Options: mapof.Any{
						"autocomplete": "off",
						"spellcheck":   false,
					},
				},
				{
					Type:        "text",
					Path:        "vault.publishableKey",
					Label:       "Publishable Key",
					Description: "Found in the <a href='https://dashboard.stripe.com/apikeys' target='_blank' rel='noopener noreferrer'>Stripe Dashboard &rarr;</a>.",
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
					Description: "Found in the <a href='https://dashboard.stripe.com/apikeys' target='_blank' rel='noopener noreferrer'>Stripe Dashboard &rarr;</a>.",
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

// BeforeSave applies any last-minute changes to this Connection before it is written to the database
func (adapter StripeConnect) BeforeSave(connection *model.Connection, vault mapof.String) error {
	return nil
}

// Connect registers this Domain's webhook endpoint with Stripe, replacing one whose signing
// secret was never stored.
func (adapter StripeConnect) Connect(connection *model.Connection, vault mapof.String, host string) error {

	const location = "providers.StripeConnect.Connect"

	// RULE: Cannot set webhooks for local domains
	if uri.IsLocalHostname(host) {
		return nil
	}

	// RULE: A webhook whose signing secret is stored needs nothing more.  One without its secret
	// can verify nothing, and Stripe reveals a secret only once, so it is replaced below.
	previousWebhookID := connection.Data.GetString("webhook")

	if (previousWebhookID != "") && (vault.GetString("webhookSecret") != "") {
		return nil
	}

	restrictedKey := vault.GetString("restrictedKey")

	// Configure a new Webhook in the Stripe API
	webhookResult := mapof.NewAny()
	txn := remote.Post(adapter.endpoint("/v1/webhook_endpoints")).
		AllowPrivateIPs(adapter.allowPrivateIPs).
		With(options.BearerAuth(restrictedKey)).
		// With(options.Debug()).
		Query("url", host+"/.stripe-connect/webhook/checkout").
		Query("description", uri.Hostname(host)+" supscription updates").
		Query("enabled_events[]", "checkout.session.completed").
		Query("enabled_events[]", "customer.subscription.created").
		Query("enabled_events[]", "customer.subscription.deleted").
		Query("enabled_events[]", "customer.subscription.paused").
		Query("enabled_events[]", "customer.subscription.resumed").
		Query("enabled_events[]", "customer.subscription.updated").
		Query("connect", "true").
		Result(&webhookResult)

	if err := txn.Send(); err != nil {
		return derp.Wrap(err, location, "Creating WebHook via Stripe API")
	}

	// RULE: An endpoint without its secret can verify nothing, so never store one
	webhookSecret := webhookResult.GetString("secret")

	if webhookSecret == "" {
		return derp.Internal(location, "Stripe did not return a webhook signing secret", uri.Hostname(host))
	}

	// Save the webhook data into the Connection.  Connection.Save seals the secret afterward.
	connection.Data.SetString("webhook", webhookResult.GetString("id"))
	connection.Vault.SetString("webhookSecret", webhookSecret)

	// Remove the endpoint this one replaces, which Stripe would otherwise keep sending events to
	if previousWebhookID != "" {
		adapter.deleteWebhook(restrictedKey, previousWebhookID, host)
	}

	// Success!
	return nil
}

// deleteWebhook removes a webhook endpoint from Stripe.  A failure is reported rather than returned,
// because the replacement is already registered and its secret must still be saved.
func (adapter StripeConnect) deleteWebhook(restrictedKey string, webhookID string, host string) {

	const location = "providers.StripeConnect.deleteWebhook"

	txn := remote.Delete(adapter.endpoint("/v1/webhook_endpoints/" + webhookID)).
		AllowPrivateIPs(adapter.allowPrivateIPs).
		With(options.BearerAuth(restrictedKey))

	if err := txn.Send(); err != nil {

		// Stripe answers 404 for an endpoint that is already gone, which is the outcome we wanted
		if derp.IsNotFound(err) {
			return
		}

		derp.Report(derp.Wrap(err, location, "Deleting replaced webhook endpoint", uri.Hostname(host), webhookID))
	}
}

// Refresh updates this connection if it has changed or is out of date
func (adapter StripeConnect) Refresh(connection *model.Connection, vault mapof.String) error {
	return nil
}

// Disconnect applies any extra changes to the database when this Adapter is disconnected
func (adapter StripeConnect) Disconnect(connection *model.Connection, vault mapof.String) error {
	return nil
}
