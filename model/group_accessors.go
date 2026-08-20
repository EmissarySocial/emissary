package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GroupSchema returns the rosetta schema that describes a Group
func GroupSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"groupId":     schema.String{Format: "objectId"},
			"token":       schema.String{Format: "token", MaxLength: 64},
			"label":       schema.String{Format: "text", MaxLength: 64, Required: true},
			"description": schema.String{Format: "text", MaxLength: 1024, Required: false},
			"icon":        schema.String{Format: "token", MaxLength: 64, Required: false},
		},
	}
}

/******************************************
 * Getter Interfaces
 ******************************************/

// GetStringOK returns the named property. Implements schema.StringGetter.
func (group *Group) GetStringOK(name string) (string, bool) {

	switch name {

	case "groupId":
		return group.GroupID.Hex(), true
	}

	return "", false
}

/******************************************
 * Setter Interfaces
 ******************************************/

// SetString writes the named property. Implements schema.StringSetter.
func (group *Group) SetString(name string, value string) bool {

	switch name {

	case "groupId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			group.GroupID = objectID
			return true
		}
	}

	return false
}

// GetPointer returns a pointer to the named property. Implements schema.PointerGetter.
func (group *Group) GetPointer(name string) (any, bool) {

	switch name {

	case "token":
		return &group.Token, true

	case "label":
		return &group.Label, true

	case "description":
		return &group.Description, true

	case "icon":
		return &group.Icon, true
	}

	return nil, false
}
