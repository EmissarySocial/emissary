package model

import "github.com/benpate/rosetta/schema"

func TagSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"type": schema.String{MaxLength: 128},
			"name": schema.String{MaxLength: 128},
			"href": schema.String{Format: "url"},
		},
	}
}

func (tag *Tag) GetPointer(name string) (any, bool) {

	switch name {

	case "type":
		return &tag.Type, true

	case "name":
		return &tag.Name, true

	case "href":
		return &tag.Href, true
	}

	return nil, false
}
