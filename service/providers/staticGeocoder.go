package providers

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
)

type StaticGeocoder struct{}

func NewStaticGeocoder() StaticGeocoder {
	return StaticGeocoder{}
}

/******************************************
 * Setup / Configuration Methods
 ******************************************/

func (adapter StaticGeocoder) ManualConfig() form.Form {

	return form.Form{
		Schema: schema.Schema{
			Element: schema.Object{
				Properties: schema.ElementMap{
					"type":   schema.String{Enum: []string{model.ConnectionTypeGeocoderIP}},
					"active": schema.Boolean{},
					"data": schema.Object{
						Properties: schema.ElementMap{
							"latitude":  schema.String{Required: true},
							"longitude": schema.String{Required: true},
						},
					},
				},
			},
		},
		Element: form.Element{
			Type:        "layout-vertical",
			Label:       "Static Geocoder Setup",
			Description: "Returns a fixed location for all IP-address geocoding requests.",
			Children: []form.Element{
				{
					Type:    "hidden",
					Path:    "type",
					Options: mapof.Any{"value": model.ConnectionTypeGeocoderIP},
				},
				{
					Type:    "text",
					Path:    "data.latitude",
					Label:   "Latitude",
					Options: mapof.Any{"autocomplete": "off"},
				},
				{
					Type:    "text",
					Path:    "data.longitude",
					Label:   "Longitude",
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
func (adapter StaticGeocoder) Connect(connection *model.Connection, vault mapof.String, host string) error {
	return nil
}

// Refresh updates this connection if it has changed or is out of date
func (adapter StaticGeocoder) Refresh(connection *model.Connection, vault mapof.String) error {
	return nil
}

// Disconnect applies any extra changes to the database when this Adapter is disconnected
func (adapter StaticGeocoder) Disconnect(connection *model.Connection, vault mapof.String) error {
	return nil
}
