package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/*********************************
 * Getter Methods
 *********************************/

func (origin *OriginLink) GetStringOK(name string) (string, bool) {
	switch name {

	case "internalId":
		return origin.InternalID.Hex(), true

	case "type":
		return origin.Type, true

	case "url":
		return origin.URL, true

	case "label":
		return origin.Label, true

	case "imageUrl":
		return origin.ImageURL, true

	}

	return "", false
}

/*********************************
 * Setter Methods
 *********************************/

func (origin *OriginLink) SetStringOK(name string, value string) bool {
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
