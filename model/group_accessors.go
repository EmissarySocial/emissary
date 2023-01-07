package model

import "go.mongodb.org/mongo-driver/bson/primitive"

/*******************************************
 * Getters
 *******************************************/

func (group *Group) GetString(name string) string {
	switch name {
	case "groupId":
		return group.GroupID.Hex()
	case "label":
		return group.Label
	}
	return ""
}

/*******************************************
 * Setters
 *******************************************/

func (group *Group) SetString(name string, value string) bool {
	switch name {

	case "groupId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			group.GroupID = objectID
			return true
		}

	case "label":
		group.Label = value
	}

	return false
}
