package providers

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
)

const ProviderTypeStripe = "STRIPE"

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
					"type":   schema.String{Enum: []string{"PAYMENT"}},
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
			Label:       "Stripe Setup",
			Description: "Sign into your Stripe account and create an API key.  Then, paste the API key into the field below.",
			Children: []form.Element{
				{
					Type:    "hidden",
					Path:    "type",
					Options: mapof.Any{"value": "PAYMENT"},
				},
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
func (adapter Stripe) AfterConnect(factory Factory, client *model.Connection, vault mapof.String) error {
	return nil
}

// AfterUpdate is called after a user has successfully updated their Twitter connection
func (adapter Stripe) AfterUpdate(factory Factory, client *model.Connection, vault mapof.String) error {
	return nil
}
