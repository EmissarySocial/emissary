package providers

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
)

const ProviderTypeOpenStreetMap = "OPENSTREETMAP"

type OpenStreetMap struct{}

func NewOpenStreetMap() OpenStreetMap {
	return OpenStreetMap{}
}

/******************************************
 * Setup / Configuration Methods
 ******************************************/

func (adapter OpenStreetMap) ManualConfig() form.Form {

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
			Label:       "OpenStreetMap Setup",
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

// AfterCoonnect applies any extra changes to the database after this Adapter is activated.
func (adapter OpenStreetMap) AfterConnect(factory Factory, client *model.Connection) error {
	return nil
}

// AfterUpdate is called after a user has successfully updated their Twitter connection
func (adapter OpenStreetMap) AfterUpdate(factory Factory, client *model.Connection) error {
	return nil
}
