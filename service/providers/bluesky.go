package providers

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
)

type Bluesky struct{}

func NewBluesky() Bluesky {
	return Bluesky{}
}

/******************************************
 * Setup / Configuration Methods
 ******************************************/

func (adapter Bluesky) ManualConfig() form.Form {

	return form.Form{
		Schema: schema.Schema{
			Element: schema.Object{
				Properties: schema.ElementMap{
					"active": schema.Boolean{},
					"data": schema.Object{
						Properties: schema.ElementMap{
							"serverUrl":   schema.String{Required: true, Format: "url"},
							"bridgeActor": schema.String{Required: true},
						},
					},
				},
			},
		},
		Element: form.Element{
			Type:        "layout-vertical",
			Label:       "<i class='bi bi-bluesky'></i> Bluesky Bridge",
			Description: "Use Bridgy Fed to connect to Bluesky.",
			Children: []form.Element{
				{
					Type:        "text",
					Path:        "data.serverUrl",
					Label:       "Bridgy Fed Server URL",
					Description: "(https://fed.brid.gy) The URL of your Bridgy Fed server.",
					Options: mapof.Any{
						"placeholder": "https://fed.brid.gy",
					},
				},
				{
					Type:        "text",
					Path:        "data.bridgeActor",
					Label:       "Bridgy Fed Actor",
					Description: "(@ap.brid.gy) The actor ID to follow or block to join the bridge.",
					Options: mapof.Any{
						"placeholder": "@ap.brid.gy",
					},
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

// Connect applies any extra changes to the database after this Adapter is activated.
func (adapter Bluesky) Connect(connection *model.Connection, vault mapof.String, host string) error {
	return nil
}

// Refresh updates this connection if it has changed or is out of date
func (adapter Bluesky) Refresh(connection *model.Connection, vault mapof.String) error {
	return nil
}

// Disconnect applies any extra changes to the database when this Adapter is disconnected
func (adapter Bluesky) Disconnect(connection *model.Connection, vault mapof.String) error {
	return nil
}
