package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func GroupSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"groupId": schema.String{Format: "objectId"},
			"label":   schema.String{MaxLength: 64, Required: true},
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

	case "label":
		return group.Label, true
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

	case "label":
		group.Label = value
		return true
	}

	return false
}
