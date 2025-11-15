package providers

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
)

type Unsplash struct{}

func NewUnsplash() Unsplash {
	return Unsplash{}
}

/******************************************
 * Setup / Configuration Methods
 ******************************************/

func (adapter Unsplash) ManualConfig() form.Form {

	return form.Form{
		Schema: schema.Schema{
			Element: schema.Object{
				Properties: schema.ElementMap{
					"type":   schema.String{Enum: []string{model.ConnectionTypeImage}},
					"active": schema.Boolean{},
					"data": schema.Object{
						Properties: schema.ElementMap{
							"applicationId":   schema.String{Required: true},
							"applicationName": schema.String{Required: true},
							"accessKey":       schema.String{Required: true},
							"secretKey":       schema.String{Required: true},
						},
					},
				},
			},
		},
		Element: form.Element{
			Type:        "layout-vertical",
			Label:       "<i class='bi bi-image'></i> Unsplash Setup",
			Description: "Sign into your Unsplash account and create an API key.  Then, paste the API key into the field below.",
			Children: []form.Element{
				{
					Type:    "hidden",
					Path:    "type",
					Options: mapof.Any{"value": model.ConnectionTypeImage},
				},
				{
					Type:    "text",
					Path:    "data.applicationId",
					Label:   "Application ID",
					Options: mapof.Any{"autocomplete": "off"},
				},
				{
					Type:    "text",
					Path:    "data.applicationName",
					Label:   "Application Name",
					Options: mapof.Any{"autocomplete": "off"},
				},
				{
					Type:    "text",
					Path:    "data.accessKey",
					Label:   "Access Key",
					Options: mapof.Any{"autocomplete": "off"},
				},
				{
					Type:    "text",
					Path:    "data.secretKey",
					Label:   "Secret Key",
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
func (adapter Unsplash) Connect(connection *model.Connection, vault mapof.String, host string) error {
	return nil
}

// Refresh updates this connection if it has changed or is out of date
func (adapter Unsplash) Refresh(connection *model.Connection, vault mapof.String) error {
	return nil
}

// Disconnect applies any extra changes to the database when this Adapter is disconnected
func (adapter Unsplash) Disconnect(connection *model.Connection, vault mapof.String) error {
	return nil
}
