package model

import "go.mongodb.org/mongo-driver/bson/primitive"

/*******************************************
 * Getters
 *******************************************/

func (group *Group) GetStringOK(name string) (string, bool) {

	switch name {

	case "groupId":
		return group.GroupID.Hex(), true

	case "label":
		return group.Label, true
	}

	return "", false
}

/*******************************************
 * Setters
 *******************************************/

func (group *Group) SetStringOK(name string, value string) bool {

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
