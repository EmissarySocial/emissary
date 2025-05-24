package providers

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/remote"
	"github.com/benpate/remote/options"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"golang.org/x/oauth2"
)

const ProviderTypePayPal = "PAYPAL"

type PayPal struct{}

func NewPayPal() PayPal {
	return PayPal{}
}

func (provider PayPal) ManualConfig() form.Form {

	options := []any{
		mapof.Any{"value": "SANDBOX", "label": "Sandbox. Test Transactions Only"},
		mapof.Any{"value": "LIVE", "label": "LIVE. Processing Real Payments"},
	}

	return form.Form{
		Schema: schema.Schema{
			Element: schema.Object{
				Properties: schema.ElementMap{
					"type":   schema.String{Enum: []string{"USER-PAYMENT"}},
					"active": schema.Boolean{},
					"data": schema.Object{
						Properties: schema.ElementMap{
							"clientId": schema.String{Required: true},
							"liveMode": schema.String{Enum: []string{"SANDBOX", "LIVE"}},
						},
					},
					"vault": schema.Object{
						Properties: schema.ElementMap{
							"secretKey": schema.String{Required: true},
						},
					},
				},
			},
		},
		Element: form.Element{
			Type:        "layout-vertical",
			Label:       "PayPal Partner Setup",
			Description: "Allows users to accept payments from their own PayPal accounts.",
			Children: []form.Element{
				{
					Type:    "hidden",
					Path:    "type",
					Options: mapof.Any{"value": "USER-PAYMENT"},
				},
				{
					Type:    "text",
					Path:    "data.clientId",
					Label:   "Client ID",
					Options: mapof.Any{"autocomplete": "off"},
				},
				{
					Type:    "text",
					Path:    "vault.secretKey",
					Label:   "Secret Key",
					Options: mapof.Any{"autocomplete": "off"},
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

func (provider PayPal) AfterConnect(factory Factory, connection *model.Connection, vault mapof.String) error {

	if err := provider.refreshAccessToken(connection, vault); err != nil {
		return derp.Wrap(err, "service.providers.PayPal", "Error refreshing access token")
	}

	return nil
}

func (provider PayPal) AfterUpdate(factory Factory, connection *model.Connection, vault mapof.String) error {

	if err := provider.refreshAccessToken(connection, vault); err != nil {
		return derp.Wrap(err, "service.providers.PayPal", "Error refreshing access token")
	}

	return nil
}

func (provider PayPal) refreshAccessToken(connection *model.Connection, vault mapof.String) error {

	const location = "service.providers.PayPal.refreshAccessToken"

	// Request a new access token
	token := oauth2.Token{}
	url := provider.serverName(connection) + "/v1/oauth2/token"

	txn := remote.Post(url).
		ContentType("application/x-www-form-urlencoded").
		Form("grant_type", "client_credentials").
		With(options.BasicAuth(connection.Data.GetString("clientId"), vault.GetString("secretKey"))).
		Result(&token)

	if err := txn.Send(); err != nil {
		return derp.Wrap(err, location, "Error requesting Access Token from PayPal")
	}

	// Apply the access token to the connection object
	connection.Token = &token

	return nil
}

func (provider PayPal) serverName(connection *model.Connection) string {

	if connection.Data.GetString("liveMode") == "LIVE" {
		return "https://api-m.paypal.com"
	}

	return "https://api-m.sandbox.paypal.com"
}
