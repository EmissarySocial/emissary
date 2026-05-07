package providers

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
)

type Bluesky struct{}

func NewBluesky() Bluesky {
	return Bluesky{}
}

/******************************************
 * Setup / Configuration Methods
 ******************************************/

func (adapter Bluesky) ManualConfig() form.Form {

	return form.Form{
		Schema: schema.Schema{
			Element: schema.Object{
				Properties: schema.ElementMap{
					"active": schema.Boolean{},
					"data": schema.Object{
						Properties: schema.ElementMap{
							"allowType":   schema.String{Required: true, Enum: []string{"ALL", "GROUPS", "NONE"}},
							"shareGroups": schema.Array{Items: schema.String{Format: "objectId"}},
						},
					},
				},
			},
		},
		Element: form.Element{
			Type:        "layout-vertical",
			Label:       "<i class='bi bi-bluesky'></i> Bluesky Bridge",
			Description: "Allow users on this server to post and follow on the ATProto network (including BlueSky, BlackSky, EuroSky, and others) using Bridgy Fed. Individual users will have the choice to join the bridge or not.",
			Children: []form.Element{
				{
					Type:  "select",
					Path:  "data.allowType",
					Label: "Who can bridge to Bluesky?",
					Options: mapof.Any{
						"enum": []form.LookupCode{
							{Value: "NONE", Label: "Nobody(Disabled)"},
							{Value: "GROUPS", Label: "Selected Groups Only"},
							{Value: "ALL", Label: "All Users"},
						},
					},
				},
				{
					Type:  "multiselect",
					Path:  "data.shareGroups",
					Label: "Members of these groups only...",
					Options: mapof.Any{
						"show-if":  "data.allowType is GROUPS",
						"provider": "groups",
					},
				},
			},
		},
	}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

func (adapter Bluesky) BeforeSave(connection *model.Connection, vault mapof.String) error {
	return nil
}

// Connect applies any extra changes to the database after this Adapter is activated.
func (adapter Bluesky) Connect(connection *model.Connection, vault mapof.String, host string) error {
	return nil
}

// Refresh updates this connection if it has changed or is out of date
func (adapter Bluesky) Refresh(connection *model.Connection, vault mapof.String) error {
	return nil
}

// Disconnect applies any extra changes to the database when this Adapter is disconnected
func (adapter Bluesky) Disconnect(connection *model.Connection, vault mapof.String) error {
	return nil
}
