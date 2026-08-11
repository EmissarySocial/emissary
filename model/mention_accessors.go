package model

import (
	"github.com/benpate/rosetta/schema"
)

func MentionSchema() schema.Element {

	return schema.Object{
		Properties: schema.ElementMap{
			"handle": schema.String{Format: "text", MaxLength: 256},
			"href":   schema.String{Format: "url"},
		},
	}
}

/********************************
 * Getter/Setter Interfaces
 ********************************/

func (mention *Mention) GetPointer(name string) (any, bool) {
	switch name {

	case "handle":
		return &mention.Handle, true

	case "href":
		return &mention.Href, true
	}

	return nil, false
}
