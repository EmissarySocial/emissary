package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/*********************************
 * Getter Methods
 *********************************/

func (origin *OriginLink) GetObjectID(name string) primitive.ObjectID {
	switch name {
	case "internalId":
		return origin.InternalID
	default:
		return primitive.NilObjectID
	}
}

func (origin *OriginLink) GetString(name string) string {
	switch name {
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

func (origin *OriginLink) SetObjectID(name string, value primitive.ObjectID) bool {
	switch name {
	case "internalId":
		origin.InternalID = value
		return true
	default:
		return false
	}
}

func (origin *OriginLink) SetString(name string, value string) bool {
	switch name {
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
	default:
		return false
	}
}
