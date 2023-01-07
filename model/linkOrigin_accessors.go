package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/*********************************
 * Getter Methods
 *********************************/

func (origin *OriginLink) GetString(name string) string {
	switch name {
	case "internalId":
		return origin.InternalID.Hex()
	case "type":
		return origin.Type
	case "url":
		return origin.URL
	case "label":
		return origin.Label
	case "imageUrl":
		return origin.ImageURL
	default:
		return ""
	}
}

/*********************************
 * Setter Methods
 *********************************/

func (origin *OriginLink) SetString(name string, value string) bool {
	switch name {
	case "internalId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			origin.InternalID = objectID
			return true
		}
	case "type":
		origin.Type = value
		return true
	case "url":
		origin.URL = value
		return true
	case "label":
		origin.Label = value
		return true
	case "imageUrl":
		origin.ImageURL = value
		return true
	}

	return false
}
