package config

import "github.com/benpate/rosetta/schema"

func DatabaseConnectInfo() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"connectString": schema.String{Required: true},
			"database":      schema.String{Required: true},
		},
	}
}
