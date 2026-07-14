package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// OutboxItemSchema returns a validating schema for OutboxItem objects.
func OutboxItemSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"activityId": schema.String{Format: "objectId"},
			"actorId":    schema.String{Format: "objectId"},
			"actorType":  schema.String{Enum: []string{ActorTypeUser, ActorTypeStream, ActorTypeSearchQuery, ActorTypeSearchDomain, ActorTypeApplication}},
			"recipients": schema.Array{Items: schema.String{}},
			"url":        schema.String{Format: "url"},
			"activity":   schema.Object{Wildcard: schema.Any{}},
		},
	}
}

/******************************************
 * Getter Interfaces
 ******************************************/

// GetPointer implements the schema.PointerGetter interface, and
// allows read/write access to (most) fields of the OutboxItem object.
func (item *OutboxItem) GetPointer(name string) (any, bool) {

	switch name {

	case "url":
		return &item.URL, true

	case "actorType":
		return &item.ActorType, true

	case "recipients":
		return &item.Recipients, true

	case "activity":
		return &item.Activity, true
	}

	return "", false
}

// GetStringOK implements the schema.StringGetter interface, and
// returns string values for several fields of the OutboxItem object.
func (item *OutboxItem) GetStringOK(name string) (string, bool) {

	switch name {

	case "activityId":
		return item.ActivityID.Hex(), true

	case "actorId":
		return item.ActorID.Hex(), true
	}

	return "", false
}

/******************************************
 * Setter Interfaces
 ******************************************/

// SetString implemments the schema.StringSetter interface, and
// allows setting string values for several fields of the OutboxItem object.
func (item *OutboxItem) SetString(name string, value string) bool {

	switch name {

	case "activityId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			item.ActivityID = objectID
			return true
		}

	case "actorId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			item.ActorID = objectID
			return true
		}
	}

	return false
}
