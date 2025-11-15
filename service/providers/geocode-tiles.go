package providers

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
)

type GeocodeTiles struct{}

func NewGeocodeTiles() GeocodeTiles {
	return GeocodeTiles{}
}

/******************************************
 * Setup / Configuration Methods
 ******************************************/

func (adapter GeocodeTiles) ManualConfig() form.Form {

	return form.Form{
		Schema: schema.Schema{
			Element: schema.Object{
				Properties: schema.ElementMap{
					"type":   schema.String{Enum: []string{model.ConnectionProviderGeocodeTiles}},
					"active": schema.Boolean{},
					"data": schema.Object{
						Properties: schema.ElementMap{
							"provider": schema.String{Required: true},
							"style":    schema.String{Required: true},
							"apiKey":   schema.String{Required: false},
						},
					},
				},
			},
		},
		Element: form.Element{
			Type:  "layout-vertical",
			Label: "<i class='bi bi-map'></i> Map Tiles",
			Children: []form.Element{
				{
					Type:        "html",
					Description: "Configure maps to use custom tiles from both free and commercial sources. <a href=https://emissary.social/geocode-tiles target=_blank>Learn More &rarr;</a>",
				},
				{
					Type:    "hidden",
					Path:    "type",
					Options: mapof.Any{"value": model.ConnectionTypeGeocodeTiles},
				},
				{
					Type:  "select-group",
					Path:  "data.provider",
					Label: "Service Provider",
					Options: mapof.Any{
						"provider": "geocode-tiles",
						"children": "data.style",
					},
				},
				{
					Type:    "layout-vertical",
					Options: mapof.Any{"show-if": "data.provider != Custom"},
					Children: []form.Element{
						{
							ID:    "geocode-tiles-style",
							Type:  "select",
							Label: "Map Style",
							Path:  "data.style",
						},
						{
							Type:  "text",
							Label: "API Key",
							Path:  "data.apiKey",
							Options: mapof.Any{
								"show-if":     "data.provider != Open Street Map",
								"autocorrect": "false",
								"spellcheck":  "false",
							},
						},
					},
				},
				{
					Type:        "text",
					Label:       "ZXY Tile URL",
					Description: "Looks like: https://tile.openstreetmap.org/{z}/{x}/{y}.png",
					Path:        "data.href",
					Options: mapof.Any{
						"show-if": "data.provider == Custom",
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
func (adapter GeocodeTiles) Connect(connection *model.Connection, vault mapof.String, host string) error {
	return nil
}

// Refresh updates this connection if it has changed or is out of date
func (adapter GeocodeTiles) Refresh(connection *model.Connection, vault mapof.String) error {
	return nil
}

// Disconnect applies any extra changes to the database when this Adapter is disconnected
func (adapter GeocodeTiles) Disconnect(connection *model.Connection, vault mapof.String) error {
	return nil
}
