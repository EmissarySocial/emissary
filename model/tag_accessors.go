package model

import (
	"github.com/benpate/rosetta/schema"
)

// TagSchema returns the JSON-Schema that validates a Tag
func TagSchema() schema.Element {

	return schema.Object{
		Properties: schema.ElementMap{
			"type": schema.String{Format: "token", MaxLength: 32},
			"name": schema.String{Format: "text", MaxLength: 256},

			// Deliberately NOT Format:"url".  Href also carries TagHrefUnresolvable ("-"), and a
			// url-format rejection here would fail the whole Stream.Save.
			"href": schema.String{MaxLength: 1024},
		},
	}
}

/********************************
 * Getter/Setter Interfaces
 ********************************/

// GetPointer returns a pointer to the named field, and TRUE if the name is recognized
// It is part of the schema.PointerGetter interface
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
