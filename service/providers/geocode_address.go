package providers

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
)

type GeocodeAddress struct{}

func NewGeocodeAddress() GeocodeAddress {
	return GeocodeAddress{}
}

/******************************************
 * Setup / Configuration Methods
 ******************************************/

func (adapter GeocodeAddress) ManualConfig() form.Form {

	return form.Form{
		Schema: schema.Schema{
			Element: schema.Object{
				Properties: schema.ElementMap{
					"type":   schema.String{Enum: []string{model.ConnectionTypeGeocodeAddress}},
					"active": schema.Boolean{},
					"data": schema.Object{
						Properties: schema.ElementMap{
							"provider":  schema.String{Required: true},
							"apiID":     schema.String{Required: false},
							"apiKey":    schema.String{Required: false},
							"latitude":  schema.String{Required: false},
							"longitude": schema.String{Required: false},
						},
					},
				},
			},
		},
		Element: form.Element{
			Type:  "layout-vertical",
			Label: "<i class='bi bi-pin-map'></i> Address Geocoder",
			Children: []form.Element{
				{
					Type:        "html",
					Description: "Configure this service to look up the map coordinates of specific addresses. <a href=https://emissary.social/geocode-address target=_blank>Learn More &rarr;</a>",
				},
				{
					Type:    "hidden",
					Path:    "type",
					Options: mapof.Any{"value": model.ConnectionTypeGeocodeAddress},
				},
				{
					Type:  "select",
					Path:  "data.provider",
					Label: "Service Provider",
					Options: mapof.Any{"enum": []form.LookupCode{
						{Group: "Recommended", Value: "GEOAPIFY", Label: "Geoapify"},
						{Group: "Recommended", Value: "HERE", Label: "Here"},
						{Group: "Supported", Value: "GOOGLE-MAPS", Label: "Google Maps"},
						{Group: "Supported", Value: "MAPTILER", Label: "Maptiler"},
						{Group: "Supported", Value: "OPEN-STREET-MAP", Label: "Open Street Map"},
					}},
				},
				{
					Type:  "text",
					Path:  "data.apiID",
					Label: "API ID",
					Options: mapof.Any{
						"show-if":      "data.provider == HERE",
						"autocomplete": "off",
					},
				},
				{
					Type:  "text",
					Path:  "data.apiKey",
					Label: "API Key",
					Options: mapof.Any{
						"show-if":      "data.provider != (null)",
						"autocomplete": "off",
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
func (adapter GeocodeAddress) Connect(connection *model.Connection, vault mapof.String, host string) error {
	return nil
}

// Refresh updates this connection if it has changed or is out of date
func (adapter GeocodeAddress) Refresh(connection *model.Connection, vault mapof.String) error {
	return nil
}

// Disconnect applies any extra changes to the database when this Adapter is disconnected
func (adapter GeocodeAddress) Disconnect(connection *model.Connection, vault mapof.String) error {
	return nil
}
