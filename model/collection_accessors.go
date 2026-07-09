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
			"parentId":     schema.String{Format: "objectId"},
			"type":         schema.String{Format: "text", MaxLength: 128},
			"read":         schema.Array{Items: schema.String{MaxLength: 256}},
			"write":        schema.Array{Items: schema.String{MaxLength: 256}},
		},
	}
}

/********************************
 * Getter/Setter Interfaces
 ********************************/

func (collection *Collection) GetPointer(name string) (any, bool) {

	switch name {

	case "type":
		return &collection.Type, true

	case "read":
		return &collection.Read, true

	case "write":
		return &collection.Write, true

	}

	return nil, false
}

func (collection Collection) GetStringOK(name string) (string, bool) {

	switch name {

	case "collectionId":
		return collection.CollectionID.Hex(), true

	case "userId":
		return collection.UserID.Hex(), true

	case "parentId":
		return collection.ParentID.Hex(), true
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

	case "parentId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			collection.ParentID = objectID
			return true
		}
	}

	return false
}
