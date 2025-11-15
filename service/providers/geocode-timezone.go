package providers

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
)

type GeocodeTimezone struct{}

func NewGeocodeTimezone() GeocodeTimezone {
	return GeocodeTimezone{}
}

/******************************************
 * Setup / Configuration Methods
 ******************************************/

func (adapter GeocodeTimezone) ManualConfig() form.Form {

	return form.Form{
		Schema: schema.Schema{
			Element: schema.Object{
				Properties: schema.ElementMap{
					"type":   schema.String{Enum: []string{model.ConnectionProviderGeocodeTimezone}},
					"active": schema.Boolean{},
					"data": schema.Object{
						Properties: schema.ElementMap{
							"provider": schema.String{Required: true},
							"apiID":    schema.String{Required: false},
							"apiKey":   schema.String{Required: false},
						},
					},
				},
			},
		},
		Element: form.Element{
			Type:  "layout-vertical",
			Label: "<i class='bi bi-clock'></i> Timezone Geocoder",
			Children: []form.Element{
				{
					Type:        "html",
					Description: "Discover timezone information from an address or map coordinates. Required for accurate event times. <a href=https://emissary.social/geocode-timezone target=_blank>Learn More &rarr;</a>",
				},
				{
					Type:    "hidden",
					Path:    "type",
					Options: mapof.Any{"value": model.ConnectionTypeGeocodeTimezone},
				},
				{
					Type:  "select",
					Path:  "data.provider",
					Label: "Service Provider",
					Options: mapof.Any{
						"enum": []form.LookupCode{
							{Group: "Recommended", Value: "GEOAPIFY", Label: "Geoapify"},
							{Group: "Recommended", Value: "HERE", Label: "Here"},
							{Group: "Supported", Value: "GEOCODIO", Label: "Geocodio (US Only)"},
							{Group: "Supported", Value: "GOOGLE", Label: "Google Maps"},
						},
					},
				},
				{
					Type:  "text",
					Label: "API ID",
					Path:  "data.apiID",
					Options: mapof.Any{
						"show-if":     "data.provider == HERE",
						"autocorrect": "false",
						"spellcheck":  "false",
					},
				},
				{
					Type:  "text",
					Label: "API Key",
					Path:  "data.apiKey",
					Options: mapof.Any{
						"autocorrect": "false",
						"spellcheck":  "false",
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
func (adapter GeocodeTimezone) Connect(connection *model.Connection, vault mapof.String, host string) error {
	return nil
}

// Refresh updates this connection if it has changed or is out of date
func (adapter GeocodeTimezone) Refresh(connection *model.Connection, vault mapof.String) error {
	return nil
}

// Disconnect applies any extra changes to the database when this Adapter is disconnected
func (adapter GeocodeTimezone) Disconnect(connection *model.Connection, vault mapof.String) error {
	return nil
}
