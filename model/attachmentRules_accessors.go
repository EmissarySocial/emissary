package model

import "github.com/benpate/rosetta/schema"

// AttachmentRulesSchema returns the rosetta schema that describes a AttachmentRules
func AttachmentRulesSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"extensions": schema.Array{Items: schema.String{}},
			"height":     schema.Integer{},
			"width":      schema.Integer{},
		},
	}
}

// GetPointer returns a pointer to the named property. Implements schema.PointerGetter.
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
