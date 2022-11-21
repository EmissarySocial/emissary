package model

import "github.com/benpate/rosetta/schema"

type Content struct {
	Format string `json:"format" bson:"format" path:"format"`
	Raw    string `json:"raw"    bson:"raw"    path:"raw"`
	HTML   string `json:"html"   bson:"html"   path:"html"`
}

func ContentSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"format": schema.String{},
			"raw":    schema.String{Format: "unsafe-any"},
			"html":   schema.String{Format: "html"},
		},
	}
}
