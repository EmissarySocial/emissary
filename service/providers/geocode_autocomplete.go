package providers

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
)

type GeocodeAutocomplete struct{}

func NewGeocodeAutocomplete() GeocodeAutocomplete {
	return GeocodeAutocomplete{}
}

/******************************************
 * Setup / Configuration Methods
 ******************************************/

func (adapter GeocodeAutocomplete) ManualConfig() form.Form {

	return form.Form{
		Schema: schema.Schema{
			Element: schema.Object{
				Properties: schema.ElementMap{
					"type":   schema.String{Enum: []string{model.ConnectionTypeGeocodeAutocomplete}},
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
			Label: "Address Search",
			Children: []form.Element{
				{
					Type:        "html",
					Description: "Configure this service to show autocomplete search hits when entering addresses. <a href=https://emissary.social/geocode-autocomplete target=_blank>Learn More &rarr;</a>",
				},
				{
					Type:    "hidden",
					Path:    "type",
					Options: mapof.Any{"value": model.ConnectionTypeGeocodeAutocomplete},
				},
				{
					Type:  "select",
					Path:  "data.provider",
					Label: "Service Provider",
					Options: mapof.Any{
						"enum": []form.LookupCode{
							{Value: "", Label: "(select one)"},
							{Value: "GEOAPIFY", Label: "Geoapify"},
							{Value: "MAPTILER", Label: "Maptiler"},
							{Value: "NOMINATIM", Label: "Nominatim"},
							{Value: "GOOGLE-MAPS", Label: "Google Maps"},
						},
					},
				},
				{
					Type:  "text",
					Path:  "data.apiKey",
					Label: "API Key",
					Options: mapof.Any{
						"autocomplete": "off",
						"show-if":      "data.provider != (null)",
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
func (adapter GeocodeAutocomplete) Connect(connection *model.Connection, vault mapof.String, host string) error {
	return nil
}

// Refresh updates this connection if it has changed or is out of date
func (adapter GeocodeAutocomplete) Refresh(connection *model.Connection, vault mapof.String) error {
	return nil
}

// Disconnect applies any extra changes to the database when this Adapter is disconnected
func (adapter GeocodeAutocomplete) Disconnect(connection *model.Connection, vault mapof.String) error {
	return nil
}
