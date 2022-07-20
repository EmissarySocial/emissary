package config

import (
	"github.com/benpate/rosetta/null"
	"github.com/benpate/rosetta/schema"
)

// Schema returns the data schema for the configuration file.
func Schema() schema.Schema {
	return schema.Schema{
		ID:      "emissary.Server",
		Comment: "Validating schema for a server configuration",
		Element: schema.Object{
			Properties: schema.ElementMap{
				"domains": schema.Array{
					Items: DomainSchema().Element,
				},
				"certificates": schema.Object{
					Properties: schema.ElementMap{
						"adapter":  schema.String{Enum: []string{"FILE"}, Default: "FILE"},
						"location": schema.String{},
						"sync":     schema.Boolean{Default: null.NewBool(false)},
					},
				},
				"templates": schema.Object{
					Properties: schema.ElementMap{
						"adapter":  schema.String{Enum: []string{"FILE"}, Default: "FILE"},
						"location": schema.String{},
						"sync":     schema.Boolean{Default: null.NewBool(false)},
					},
				},
				"layouts": schema.Object{
					Properties: schema.ElementMap{
						"adapter":  schema.String{Enum: []string{"FILE"}, Default: "FILE"},
						"location": schema.String{},
						"sync":     schema.Boolean{Default: null.NewBool(false)},
					},
				},
				"static": schema.Object{
					Properties: schema.ElementMap{
						"adapter":  schema.String{Enum: []string{"FILE"}, Default: "FILE"},
						"location": schema.String{},
						"sync":     schema.Boolean{Default: null.NewBool(false)},
					},
				},
				"attachmentOriginals": schema.Object{
					Properties: schema.ElementMap{
						"adapter":  schema.String{Enum: []string{"FILE"}, Default: "FILE"},
						"location": schema.String{},
						"sync":     schema.Boolean{Default: null.NewBool(false)},
					},
				},
				"attachmentCache": schema.Object{
					Properties: schema.ElementMap{
						"adapter":  schema.String{Enum: []string{"FILE"}, Default: "FILE"},
						"location": schema.String{},
						"sync":     schema.Boolean{Default: null.NewBool(false)},
					},
				},
				"adminEmail": schema.String{},
			},
		},
	}
}

func DomainSchema() schema.Schema {

	return schema.Schema{
		ID:      "emissary.Domain",
		Comment: "Validating schema for a domain configuration",
		Element: schema.Object{
			Properties: map[string]schema.Element{
				"label":         schema.String{MaxLength: null.NewInt(100), Required: true},
				"hostname":      schema.String{MaxLength: null.NewInt(255), Required: true},
				"connectString": schema.String{MaxLength: null.NewInt(1000)},
				"databaseName":  schema.String{Pattern: `[a-zA-Z0-9]+`},
			},
		},
	}
}
