package providers

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/schema"
)

const ProviderTypeGiphy = "GIPHY"

const Giphy_APIKey = "apiKey"

type Giphy struct{}

func NewGiphy() Giphy {
	return Giphy{}
}

/******************************************
 * Setup / Configuration Methods
 ******************************************/

func (adapter Giphy) ManualConfig() form.Form {

	return form.Form{
		Schema: schema.Schema{
			Element: schema.Object{
				Properties: schema.ElementMap{
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
			Label:       "Giphy Setup",
			Description: "Sign into your Giphy account and create an API key.  Then, paste the API key into the field below.",
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
func (adapter Giphy) AfterConnect(factory Factory, client *model.Connection) error {
	return nil
}

// AfterUpdate is called after a user has successfully updated their Twitter connection
func (adapter Giphy) AfterUpdate(factory Factory, client *model.Connection) error {
	return nil
}
