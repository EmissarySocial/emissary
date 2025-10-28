package providers

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
)

type Nominatim struct{}

func NewNominatim() Nominatim {
	return Nominatim{}
}

/******************************************
 * Setup / Configuration Methods
 ******************************************/

func (adapter Nominatim) ManualConfig() form.Form {

	return form.Form{
		Schema: schema.Schema{
			Element: schema.Object{
				Properties: schema.ElementMap{
					"type":   schema.String{Enum: []string{model.ConnectionTypeGeosearch}},
					"active": schema.Boolean{},
					"data": schema.Object{
						Properties: schema.ElementMap{
							"serverUrl": schema.String{Required: false},
						},
					},
				},
			},
		},
		Element: form.Element{
			Type:  "layout-vertical",
			Label: "Nominatum Server",
			Children: []form.Element{
				{
					Type:        "html",
					Description: "Geographic search engine that returns locations matching users' queries.  Use the community server, or self-host for large installations.",
				},
				{
					Type:    "hidden",
					Path:    "type",
					Options: mapof.Any{"value": model.ConnectionTypeGeosearch},
				},
				{
					Type:        "text",
					Path:        "data.serverUrl",
					Label:       "Custom Server URL",
					Description: "Leave blank to use the public server (nominatim.openstreetmap.org)",
					Options:     mapof.Any{"autocomplete": "off"},
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
func (adapter Nominatim) Connect(connection *model.Connection, vault mapof.String, host string) error {
	return nil
}

// Refresh updates this connection if it has changed or is out of date
func (adapter Nominatim) Refresh(connection *model.Connection, vault mapof.String) error {
	return nil
}

// Disconnect applies any extra changes to the database when this Adapter is disconnected
func (adapter Nominatim) Disconnect(connection *model.Connection, vault mapof.String) error {
	return nil
}
