package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ObjectLinkSchema returns the rosetta schema that describes a ObjectLink
func ObjectLinkSchema() schema.Element {

	return schema.Object{
		Properties: schema.ElementMap{
			"objectLinkId": schema.String{Format: "objectId"},
			"context":      schema.String{Format: "url"},
			"inReplyTo":    schema.String{Format: "url"},
			"object":       schema.String{Format: "url"},
			"recipients":   schema.Array{Items: schema.String{Format: "url"}},
		},
	}
}

// GetPointer returns a pointer to the named property. Implements schema.PointerGetter.
func (objectLink *ObjectLink) GetPointer(name string) (any, bool) {

	switch name {

	case "context":
		return &objectLink.Context, true

	case "inReplyTo":
		return &objectLink.InReplyTo, true

	case "object":
		return &objectLink.Object, true

	case "recipients":
		return &objectLink.Recipients, true
	}

	return nil, false
}

// GetStringOK returns the named property. Implements schema.StringGetter.
func (objectLink ObjectLink) GetStringOK(name string) (string, bool) {
	switch name {

	case "objectLinkId":
		return objectLink.ObjectLinkID.Hex(), true

	}

	return "", false
}

// SetString writes the named property. Implements schema.StringSetter.
func (objectLink *ObjectLink) SetString(name string, value string) bool {

	switch name {

	case "objectLinkId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			objectLink.ObjectLinkID = objectID
			return true
		}

	}

	return false
}
