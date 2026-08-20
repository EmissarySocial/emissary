package providers

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
)

// Giphy_APIKey is the vault key that this provider stores its Giphy credential under
const Giphy_APIKey = "apiKey"

// Giphy connects a Domain to the Giphy image-search integration
type Giphy struct{}

// NewGiphy returns a fully initialized Giphy provider
func NewGiphy() Giphy {
	return Giphy{}
}

/******************************************
 * Setup / Configuration Methods
 ******************************************/

// ManualConfig returns the form used to configure this Connection by hand. Implements the ManualProvider interface.
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
			Label:       "<i class='bi bi-film'></i> Giphy Setup",
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

// BeforeSave applies any last-minute changes to this Connection before it is written to the database
func (adapter Giphy) BeforeSave(connection *model.Connection, vault mapof.String) error {
	return nil
}

// Connect applies any extra changes to the database after this Adapter is activated.
func (adapter Giphy) Connect(connection *model.Connection, vault mapof.String, host string) error {
	return nil
}

// Refresh updates this connection if it has changed or is out of date
func (adapter Giphy) Refresh(connection *model.Connection, vault mapof.String) error {
	return nil
}

// Disconnect applies any extra changes to the database when this Adapter is disconnected
func (adapter Giphy) Disconnect(connection *model.Connection, vault mapof.String) error {
	return nil
}
