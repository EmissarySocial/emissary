package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func GroupSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"groupId":     schema.String{Format: "objectId"},
			"token":       schema.String{MaxLength: 64},
			"label":       schema.String{MaxLength: 64, Required: true},
			"description": schema.String{MaxLength: 500, Required: false},
			"icon":        schema.String{MaxLength: 64, Required: false},
		},
	}
}

/******************************************
 * Getter Interfaces
 ******************************************/

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
