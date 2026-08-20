package providers

import (
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/paypal"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/remote"
	"github.com/benpate/remote/options"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"golang.org/x/oauth2"
)

// PayPal connects a Domain to the PayPal payment integration
type PayPal struct{}

// NewPayPal returns a fully initialized PayPal provider
func NewPayPal() PayPal {
	return PayPal{}
}

// ManualConfig returns the form used to configure this Connection by hand. Implements the ManualProvider interface.
func (provider PayPal) ManualConfig() form.Form {

	options := []any{
		mapof.Any{"value": "SANDBOX", "label": "Sandbox. Test Transactions Only"},
		mapof.Any{"value": "LIVE", "label": "LIVE. Processing Real Payments"},
	}

	return form.Form{
		Schema: schema.Schema{
			Element: schema.Object{
				Properties: schema.ElementMap{
					"type":   schema.String{Enum: []string{model.ConnectionTypeUserPayment}},
					"active": schema.Boolean{},
					"data": schema.Object{
						Properties: schema.ElementMap{
							"bnCode":   schema.String{Required: true},
							"liveMode": schema.String{Enum: []string{"SANDBOX", "LIVE"}},
						},
					},
					"vault": schema.Object{
						Properties: schema.ElementMap{
							"clientId":  schema.String{Required: true},
							"secretKey": schema.String{Required: true},
						},
					},
				},
			},
		},
		Element: form.Element{
			Type:        "layout-vertical",
			Label:       "PayPal Marketplace",
			Description: "Allows users to accept payments from their own PayPal accounts.",
			Children: []form.Element{
				{
					Type:    "hidden",
					Path:    "type",
					Options: mapof.Any{"value": model.ConnectionTypeUserPayment},
				},
				{
					Type:        "text",
					Path:        "vault.clientId",
					Label:       "Client ID",
					Description: "Found in the PayPal Developer Dashboard, under 'Apps & Credentials'",
					Options:     mapof.Any{"autocomplete": "off", "autocorrect": "false", "spellcheck": "false"},
				},
				{
					Type:        "text",
					Path:        "vault.secretKey",
					Label:       "Secret Key",
					Description: "Found in the PayPal Developer Dashboard, under 'Apps & Credentials'",
					Options:     mapof.Any{"autocomplete": "off", "autocorrect": "false", "spellcheck": "false"},
				},
				{
					Type:        "text",
					Path:        "data.bnCode",
					Label:       "Build Notation (BN) Code",
					Description: "Provided by PayPal during Marketplace onboarding",
					Options:     mapof.Any{"autocomplete": "off", "autocorrect": "false", "spellcheck": "false"},
				},
				{
					Type:    "select",
					Path:    "data.liveMode",
					Label:   "Live Mode",
					Options: mapof.Any{"enum": options},
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

// BeforeSave applies any last-minute changes to this Connection before it is written to the database
func (adapter PayPal) BeforeSave(connection *model.Connection, vault mapof.String) error {
	return nil
}

// Connect applies any extra changes required when this Connection is first activated
func (provider PayPal) Connect(connection *model.Connection, vault mapof.String, host string) error {

	if err := provider.Refresh(connection, vault); err != nil {
		return derp.Wrap(err, "service.providers.PayPal", "Refreshing access token", derp.WithInternalError())
	}

	return nil
}

// Refresh updates this Connection if its credentials have expired
func (provider PayPal) Refresh(connection *model.Connection, vault mapof.String) error {

	const location = "service.providers.PayPal.Refresh"

	// If the token is still valid, then don't refresh it now.
	if connection.Token.Valid() {
		return nil
	}

	// Request a new access token
	liveMode := connection.Data.GetString("liveMode") == "LIVE"
	url := paypal.APIHost(liveMode) + "/v1/oauth2/token"
	token := oauth2.Token{}

	txn := remote.Post(url).
		ContentType("application/x-www-form-urlencoded").
		Header("User-Agent", "Emissary Social").
		Form("grant_type", "client_credentials").
		With(options.BasicAuth(vault.GetString("clientId"), vault.GetString("secretKey"))).
		Result(&token)

	if err := txn.Send(); err != nil {
		return derp.Wrap(err, location, "Requesting Access Token from PayPal")
	}

	// Calculate the Token expiry time.
	token.Expiry = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second).Add(-1 * time.Hour)

	// Apply the access token to the connection object
	connection.Token = &token

	return nil
}

// Disconnect applies any extra changes to the database when this Adapter is disconnected
func (adapter PayPal) Disconnect(connection *model.Connection, vault mapof.String) error {

	// TODO: Probably need to send an API call to PayPal to revoke this token.
	// if connection.Token != nil {
	// }

	connection.Token = nil
	return nil
}
