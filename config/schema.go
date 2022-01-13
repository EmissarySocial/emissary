package config

import (
	"github.com/benpate/null"
	"github.com/benpate/schema"
)

func Schema() schema.Schema {

	result := schema.Schema{
		ID:      "ghost.Domain",
		Comment: "Validating schema for a domain configuration",
		Element: schema.Object{
			Properties: map[string]schema.Element{
				"label":         schema.String{Required: true},
				"hostname":      schema.String{Required: true},
				"connectString": schema.String{},
				"databaseName":  schema.String{},
				"smtp": schema.Object{
					Properties: map[string]schema.Element{
						"hostname": schema.String{},
						"username": schema.String{},
						"password": schema.String{},
						"tls":      schema.Boolean{Default: null.NewBool(false)},
					},
				},
				"layoutPath": schema.String{},
				"showAdmin":  schema.Boolean{Default: null.NewBool(false)},
			},
		},
	}
	return result
}
