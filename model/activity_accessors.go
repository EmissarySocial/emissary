package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ActivitySchema returns a validating schema for Activity objects.
func ActivitySchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"activityId": schema.String{Format: "objectId"},
			"actorId":    schema.String{Format: "objectId"},
			"actorType":  schema.String{Enum: []string{ActorTypeUser, ActorTypeStream, ActorTypeSearchQuery, ActorTypeSearchDomain, ActorTypeApplication}},
			"recipients": schema.Array{Items: schema.String{}},
			"url":        schema.String{Format: "url"},
			"object":     schema.Object{Wildcard: schema.Any{}},
		},
	}
}

/******************************************
 * Getter Interfaces
 ******************************************/

// GetPointer implements the schema.PointerGetter interface, and
// allows read/write access to (most) fields of the Activity object.
func (activity *Activity) GetPointer(name string) (any, bool) {

	switch name {

	case "url":
		return &activity.URL, true

	case "actorType":
		return &activity.ActorType, true

	case "recipients":
		return &activity.Recipients, true

	case "object":
		return &activity.Object, true
	}

	return "", false
}

// GetStringOK implements the schema.StringGetter interface, and
// returns string values for several fields of the Activity object.
func (activity *Activity) GetStringOK(name string) (string, bool) {

	switch name {

	case "activityId":
		return activity.ActivityID.Hex(), true

	case "actorId":
		return activity.ActorID.Hex(), true
	}

	return "", false
}

/******************************************
 * Setter Interfaces
 ******************************************/

// SetString implemments the schema.StringSetter interface, and
// allows setting string values for several fields of the Activity object.
func (activity *Activity) SetString(name string, value string) bool {

	switch name {

	case "activityId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			activity.ActivityID = objectID
			return true
		}

	case "actorId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			activity.ActorID = objectID
			return true
		}
	}

	return false
}
