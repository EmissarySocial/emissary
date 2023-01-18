package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/*********************************
 * Getter  Methods
 *********************************/

func (link *PersonLink) GetStringOK(name string) (string, bool) {
	switch name {

	case "internalId":
		return link.InternalID.Hex(), true

	case "name":
		return link.Name, true

	case "profileUrl":
		return link.ProfileURL, true

	case "inboxUrl":
		return link.InboxURL, true

	case "emailAddress":
		return link.EmailAddress, true

	case "imageUrl":
		return link.ImageURL, true

	}

	return "", false
}

/*********************************
 * Setter Methods
 *********************************/

func (link *PersonLink) SetStringOK(name string, value string) bool {
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
