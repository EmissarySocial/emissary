package providers

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
)

const ProviderTypeGoogleMaps = "GOOGLE-MAPS"

type GoogleMaps struct{}

func NewGoogleMaps() GoogleMaps {
	return GoogleMaps{}
}

/******************************************
 * Setup / Configuration Methods
 ******************************************/

func (adapter GoogleMaps) ManualConfig() form.Form {

	return form.Form{
		Schema: schema.Schema{
			Element: schema.Object{
				Properties: schema.ElementMap{
					"type":   schema.String{Enum: []string{"GEOCODER"}},
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
			Label:       "GoogleMaps Setup",
			Description: "Choose a geocoding service to look up addresses",
			Children: []form.Element{
				{
					Type:    "hidden",
					Path:    "type",
					Options: mapof.Any{"value": "GEOCODER"},
				},
				{
					Type:    "text",
					Path:    "data.apiKey",
					Label:   "API Key",
					Options: mapof.Any{"autocomplete": "off"},
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
func (adapter GoogleMaps) Connect(connection *model.Connection, vault mapof.String, host string) error {
	return nil
}

// Refresh updates this connection if it has changed or is out of date
func (adapter GoogleMaps) Refresh(connection *model.Connection, vault mapof.String) error {
	return nil
}

// Disconnect applies any extra changes to the database when this Adapter is disconnected
func (adapter GoogleMaps) Disconnect(connection *model.Connection, vault mapof.String) error {
	return nil
}
