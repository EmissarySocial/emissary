package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/*********************************
 * Getter  Methods
 *********************************/

func (link *PersonLink) GetString(name string) string {
	switch name {
	case "internalId":
		return link.InternalID.Hex()
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

func (link *PersonLink) SetString(name string, value string) bool {
	switch name {

	case "internalId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			link.InternalID = objectID
			return true
		}
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
	}

	return false
}
