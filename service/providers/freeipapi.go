package providers

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
)

const ProviderTypeFREEIPAPICOM = "FREEIPAPI.COM"

type FREEIPAPICOM struct{}

func NewFREEIPAPICOM() FREEIPAPICOM {
	return FREEIPAPICOM{}
}

/******************************************
 * Setup / Configuration Methods
 ******************************************/

func (adapter FREEIPAPICOM) ManualConfig() form.Form {

	return form.Form{
		Schema: schema.Schema{
			Element: schema.Object{
				Properties: schema.ElementMap{
					"type":   schema.String{Enum: []string{"GEOCODER-IP"}},
					"active": schema.Boolean{},
					"data": schema.Object{
						Properties: schema.ElementMap{
							"apiKey":    schema.String{Required: false},
							"latitude":  schema.String{Required: false},
							"longitude": schema.String{Required: false},
						},
					},
				},
			},
		},
		Element: form.Element{
			Type:  "layout-tabs",
			Label: "freeipapi.com Geocoder",
			Children: []form.Element{
				{
					Type:  "layout-vertical",
					Label: "API Keys",
					Children: []form.Element{
						{
							Type:        "html",
							Description: "Returns geocoded location data for each IP address. No signup required to use free tier.  Or pay for increased rate limits at <a href='https://freeipapi.com/'>freeipapi.com</a>.",
						},
						{
							Type:    "hidden",
							Path:    "type",
							Options: mapof.Any{"value": "GEOCODER-IP"},
						},
						{
							Type:        "text",
							Path:        "data.apiKey",
							Label:       "API Key",
							Description: "Leave blank to use the free tier (60 requests/day)",
							Options:     mapof.Any{"autocomplete": "off"},
						},
						{
							Type:  "toggle",
							Path:  "active",
							Label: "Enable?",
						},
					},
				},
				{
					Type:        "layout-vertical",
					Label:       "Default Location",
					Description: "Enter default coordinates to use in case there is an error locating an IP address",
					Children: []form.Element{
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
func (adapter FREEIPAPICOM) Connect(connection *model.Connection, vault mapof.String, host string) error {
	return nil
}

// Refresh updates this connection if it has changed or is out of date
func (adapter FREEIPAPICOM) Refresh(connection *model.Connection, vault mapof.String) error {
	return nil
}

// Disconnect applies any extra changes to the database when this Adapter is disconnected
func (adapter FREEIPAPICOM) Disconnect(connection *model.Connection, vault mapof.String) error {
	return nil
}
