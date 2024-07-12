package model

import "github.com/benpate/rosetta/schema"

func AttachmentRulesSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"extensions": schema.Array{Items: schema.String{}},
			"height":     schema.Integer{},
			"width":      schema.Integer{},
		},
	}
}

func (rules *AttachmentRules) GetPointer(name string) (any, bool) {

	switch name {

	case "extensions":
		return &rules.Extensions, true

	case "height":
		return &rules.Height, true

	case "width":
		return &rules.Width, true
	}

	return nil, false
}
