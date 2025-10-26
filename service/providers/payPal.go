package providers

import (
	"net/http"
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
					Options: mapof.Any{"value": "USER-PAYMENT"},
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

func (provider PayPal) Connect(connection *model.Connection, vault mapof.String, host string) error {

	if err := provider.Refresh(connection, vault); err != nil {
		return derp.Wrap(err, "service.providers.PayPal", "Error refreshing access token", derp.WithCode(http.StatusInternalServerError))
	}

	return nil
}

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
		return derp.Wrap(err, location, "Error requesting Access Token from PayPal")
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
