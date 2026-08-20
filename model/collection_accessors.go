package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CollectionSchema returns the rosetta schema that describes a Collection
func CollectionSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"collectionId":   schema.String{Format: "objectId"},
			"userId":         schema.String{Format: "objectId"},
			"parentId":       schema.String{Format: "objectId"},
			"parentType":     schema.String{Enum: []string{CollectionParentTypeStream, CollectionParentTypeUser}},
			"collectionType": schema.String{Enum: []string{CollectionTypeContext, CollectionTypeReplies, CollectionTypeLikes, CollectionTypeDislikes, CollectionTypeShares}},
			"read":           schema.Array{Items: schema.String{MaxLength: 256}},
			"write":          schema.Array{Items: schema.String{MaxLength: 256}},
			"totalItems":     schema.Integer{},
		},
	}
}

/********************************
 * Getter/Setter Interfaces
 ********************************/

// GetPointer returns a pointer to the named property. Implements schema.PointerGetter.
func (collection *Collection) GetPointer(name string) (any, bool) {

	switch name {

	case "parentType":
		return &collection.ParentType, true

	case "collectionType":
		return &collection.CollectionType, true

	case "read":
		return &collection.Read, true

	case "write":
		return &collection.Write, true

	case "totalItems":
		return &collection.TotalItems, true

	}

	return nil, false
}

// GetStringOK returns the named property. Implements schema.StringGetter.
func (collection Collection) GetStringOK(name string) (string, bool) {

	switch name {

	case "collectionId":
		return collection.CollectionID.Hex(), true

	case "userId":
		return collection.UserID.Hex(), true

	case "parentId":
		return collection.ParentID.Hex(), true

	case "parentType":
		return collection.ParentType, true

	case "collectionType":
		return collection.CollectionType, true
	}

	return "", false
}

/*********************************
 * Setter Interfaces
 *********************************/

// SetString writes the named property. Implements schema.StringSetter.
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
