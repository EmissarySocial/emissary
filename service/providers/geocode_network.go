package providers

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
)

type GeocodeNetwork struct{}

func NewGeocodeNetwork() GeocodeNetwork {
	return GeocodeNetwork{}
}

/******************************************
 * Setup / Configuration Methods
 ******************************************/

func (adapter GeocodeNetwork) ManualConfig() form.Form {

	return form.Form{
		Schema: schema.Schema{
			Element: schema.Object{
				Properties: schema.ElementMap{
					"type":   schema.String{Enum: []string{model.ConnectionTypeGeocodeNetwork}},
					"active": schema.Boolean{},
					"data": schema.Object{
						Properties: schema.ElementMap{
							"provider":  schema.String{Required: true},
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
			Label: "<i class='bi bi-diagram-2'></i> Network Geocoder",
			Children: []form.Element{
				{
					Type:        "html",
					Description: "Configure this service to look up map coordinates using clients' IP addresses. <a href=https://emissary.social/geocode-network target=_blank>Learn More &rarr;</a>",
				},
				{
					Type:    "hidden",
					Path:    "type",
					Options: mapof.Any{"value": model.ConnectionTypeGeocodeNetwork},
				},
				{
					Type:  "select",
					Path:  "data.provider",
					Label: "Service Provider",
					Options: mapof.Any{"enum": []form.LookupCode{
						{Group: "Recommended", Value: "GEOAPIFY", Label: "Geoapify"},
						{Group: "Supported", Value: "FREEIPAPI", Label: "FreeIPAPI.com"},
						{Group: "Supported", Value: "IPAPICOM", Label: "IP-API.COM"},
						{Group: "Supported", Value: "STATIC", Label: "Static Location"},
					}},
				},
				{
					Type: "layout-vertical",
					Options: mapof.Any{
						"show-if": "data.provider != (null)",
					},
					Children: []form.Element{
						{
							Type:  "text",
							Path:  "data.apiKey",
							Label: "API Key",
							Options: mapof.Any{
								"autocomplete": "off",
								"show-if":      "data.provider != STATIC",
							},
						},
						{
							Type:  "text",
							Path:  "data.latitude",
							Label: "Latitude",
							Options: mapof.Any{
								"autocomplete": "off",
								"show-if":      "data.provider == STATIC",
							},
						},
						{
							Type:  "text",
							Path:  "data.longitude",
							Label: "Longitude",
							Options: mapof.Any{
								"autocomplete": "off",
								"show-if":      "data.provider == STATIC",
							},
						},
						{
							Type:  "toggle",
							Path:  "active",
							Label: "Enable?",
						},
					},
				},
			},
		},
	}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Connect applies any extra changes to the database after this Adapter is activated.
func (adapter GeocodeNetwork) Connect(connection *model.Connection, vault mapof.String, host string) error {
	return nil
}

// Refresh updates this connection if it has changed or is out of date
func (adapter GeocodeNetwork) Refresh(connection *model.Connection, vault mapof.String) error {
	return nil
}

// Disconnect applies any extra changes to the database when this Adapter is disconnected
func (adapter GeocodeNetwork) Disconnect(connection *model.Connection, vault mapof.String) error {
	return nil
}
