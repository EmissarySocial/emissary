package providers

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
)

type Geoapify struct{}

func NewGeoapify() Geoapify {
	return Geoapify{}
}

/******************************************
 * Setup / Configuration Methods
 ******************************************/

func (adapter Geoapify) ManualConfig() form.Form {

	return form.Form{
		Schema: schema.Schema{
			Element: schema.Object{
				Properties: schema.ElementMap{
					"type":   schema.String{Enum: []string{model.ConnectionTypeGeosearch}},
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
			Type:  "layout-vertical",
			Label: "Nominatum Server",
			Children: []form.Element{
				{
					Type:        "html",
					Description: "Sign up for Geoapify at <a href='geoapify.com'>Geoapify.com</a> and greate an API key for their `Autocomplete` API.",
				},
				{
					Type:    "hidden",
					Path:    "type",
					Options: mapof.Any{"value": model.ConnectionTypeGeosearch},
				},
				{
					Type:        "text",
					Path:        "data.apiKey",
					Label:       "API Key",
					Description: "Enter your `Autocomplete` API key ",
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
func (adapter Geoapify) Connect(connection *model.Connection, vault mapof.String, host string) error {
	return nil
}

// Refresh updates this connection if it has changed or is out of date
func (adapter Geoapify) Refresh(connection *model.Connection, vault mapof.String) error {
	return nil
}

// Disconnect applies any extra changes to the database when this Adapter is disconnected
func (adapter Geoapify) Disconnect(connection *model.Connection, vault mapof.String) error {
	return nil
}
