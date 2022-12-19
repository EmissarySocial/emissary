package model

import "go.mongodb.org/mongo-driver/bson/primitive"

func (group *Group) GetObjectID(name string) primitive.ObjectID {
	switch name {
	case "groupId":
		return group.GroupID
	}
	return primitive.NilObjectID
}

func (group *Group) GetString(name string) string {
	switch name {
	case "label":
		return group.Label
	}
	return ""
}
