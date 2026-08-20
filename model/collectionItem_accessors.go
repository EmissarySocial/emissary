package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CollectionItemSchema returns the rosetta schema that describes a CollectionItem
func CollectionItemSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"collectionItemId": schema.String{Required: true, Format: "objectId"},
			"collectionId":     schema.String{Required: true, Format: "objectId"},
			"userId":           schema.String{Required: true, Format: "objectId"},
			"parentId":         schema.String{Required: true, Format: "objectId"},
			"collectionType":   schema.String{Required: true, Enum: []string{CollectionTypeContext, CollectionTypeDislikes, CollectionTypeLikes, CollectionTypeReplies, CollectionTypeShares}},
			"uri":              schema.String{Required: true, Format: "uri"},
		},
	}
}

/********************************
 * Getter/Setter Interfaces
 ********************************/

// GetStringOK returns the named property. Implements schema.StringGetter.
func (collectionItem CollectionItem) GetStringOK(name string) (string, bool) {

	switch name {

	case "collectionItemId":
		return collectionItem.CollectionItemID.Hex(), true

	case "collectionId":
		return collectionItem.CollectionID.Hex(), true

	case "userId":
		return collectionItem.UserID.Hex(), true

	case "parentId":
		return collectionItem.ParentID.Hex(), true

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

// SetString writes the named property. Implements schema.StringSetter.
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

	case "parentId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			collectionItem.ParentID = objectID
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
