package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/*********************************
 * Getter  Methods
 *********************************/

func (link *PersonLink) GetObjectID(name string) primitive.ObjectID {
	switch name {
	case "internalId":
		return link.InternalID
	default:
		return primitive.NilObjectID
	}
}

func (link *PersonLink) GetString(name string) string {
	switch name {
	case "name":
		return link.Name
	case "profileUrl":
		return link.ProfileURL
	case "inboxUrl":
		return link.InboxURL
	case "emailAddress":
		return link.EmailAddress
	case "imageUrl":
		return link.ImageURL
	default:
		return ""
	}
}

/*********************************
 * Setter Methods
 *********************************/

func (link *PersonLink) SetObjectID(name string, value primitive.ObjectID) bool {
	switch name {
	case "internalId":
		link.InternalID = value
		return true
	default:
		return false
	}
}

func (link *PersonLink) SetString(name string, value string) bool {
	switch name {
	case "name":
		link.Name = value
		return true
	case "profileUrl":
		link.ProfileURL = value
		return true
	case "inboxUrl":
		link.InboxURL = value
		return true
	case "emailAddress":
		link.EmailAddress = value
		return true
	case "imageUrl":
		link.ImageURL = value
		return true
	default:
		return false
	}
}
