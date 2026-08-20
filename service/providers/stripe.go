package providers

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
)

// Stripe connects a Domain to the Stripe payment integration
type Stripe struct{}

// NewStripe returns a fully initialized Stripe provider
func NewStripe() Stripe {
	return Stripe{}
}

/******************************************
 * Setup / Configuration Methods
 ******************************************/

// ManualConfig returns the form used to configure this Connection by hand. Implements the ManualProvider interface.
func (adapter Stripe) ManualConfig() form.Form {

	return form.Form{
		Schema: schema.Schema{
			Element: schema.Object{
				Properties: schema.ElementMap{
					"type":   schema.String{Enum: []string{model.ConnectionTypeUserPayment}},
					"active": schema.Boolean{},
					"data": schema.Object{
						Properties: schema.ElementMap{
							"apiKey": schema.String{Required: true},
						},
					},
				},
			},
		},
		Element: form.Element{
			Type:        "layout-vertical",
			Label:       "<i class='bi bi-stripe'></i> Stripe Setup",
			Description: "Allows users to accept payments by entering Stripe API keys directly.",
			Children: []form.Element{
				{
					Type:    "hidden",
					Path:    "type",
					Options: mapof.Any{"value": model.ConnectionTypeUserPayment},
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

// BeforeSave applies any last-minute changes to this Connection before it is written to the database
func (adapter Stripe) BeforeSave(connection *model.Connection, vault mapof.String) error {
	return nil
}

// Connect applies any extra changes to the database after this Adapter is activated.
func (adapter Stripe) Connect(connection *model.Connection, vault mapof.String, host string) error {
	return nil
}

// Refresh updates this connection if it has changed or is out of date
func (adapter Stripe) Refresh(connection *model.Connection, vault mapof.String) error {
	return nil
}

// Disconnect applies any extra changes to the database when this Adapter is disconnected
func (adapter Stripe) Disconnect(connection *model.Connection, vault mapof.String) error {
	return nil
}
