package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func CollectionItemSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"collectionItemId": schema.String{Format: "objectId"},
			"collectionId":     schema.String{Format: "objectId"},
			"userId":           schema.String{Format: "objectId"},
			"collectionType":   schema.String{Enum: []string{CollectionTypeContext, CollectionTypeDislikes, CollectionTypeLikes, CollectionTypeReplies, CollectionTypeShares}},
			"uri":              schema.String{Format: "url"},
		},
	}
}

/********************************
 * Getter/Setter Interfaces
 ********************************/

func (collectionItem CollectionItem) GetStringOK(name string) (string, bool) {

	switch name {

	case "collectionItemId":
		return collectionItem.CollectionItemID.Hex(), true

	case "collectionId":
		return collectionItem.CollectionID.Hex(), true

	case "userId":
		return collectionItem.UserID.Hex(), true

	case "collectionType":
		return collectionItem.CollectionType, true

	case "uri":
		return collectionItem.URI, true
	}

	return "", false
}

/*********************************
 * Setter Interfaces
 *********************************/

func (collectionItem *CollectionItem) SetString(name string, value string) bool {

	switch name {

	case "collectionItemId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			collectionItem.CollectionItemID = objectID
			return true
		}

	case "collectionId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			collectionItem.CollectionID = objectID
			return true
		}

	case "userId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			collectionItem.UserID = objectID
			return true
		}

	case "collectionType":
		collectionItem.CollectionType = value
		return true

	case "uri":
		collectionItem.URI = value
		return true
	}

	return false
}
