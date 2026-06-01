package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func CollectionSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"collectionId": schema.String{Format: "objectId"},
			"userId":       schema.String{Format: "objectId"},
			"to":           schema.Array{Items: schema.String{}},
			"name":         schema.String{MaxLength: 128},
			"comment":      schema.String{MaxLength: 2048},
			"icon":         schema.String{MaxLength: 16},
		},
	}
}

/********************************
 * Getter/Setter Interfaces
 ********************************/

func (collection *Collection) GetPointer(name string) (any, bool) {

	switch name {

	case "to":
		return &collection.To, true

	case "name":
		return &collection.Name, true
	}

	return nil, false
}

func (collection Collection) GetStringOK(name string) (string, bool) {

	switch name {

	case "collectionId":
		return collection.CollectionID.Hex(), true

	case "userId":
		return collection.UserID.Hex(), true
	}

	return "", false
}

/*********************************
 * Setter Interfaces
 *********************************/

func (collection *Collection) SetString(name string, value string) bool {

	switch name {

	case "collectionId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			collection.CollectionID = objectID
			return true
		}

	case "userId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			collection.UserID = objectID
			return true
		}
	}

	return false
}
