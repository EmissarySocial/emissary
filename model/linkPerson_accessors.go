package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func PersonLinkSchema() schema.Element {

	return schema.Object{
		Properties: schema.ElementMap{
			"internalId":   schema.String{Format: "objectId"},
			"name":         schema.String{MaxLength: 128},
			"profileUrl":   schema.String{Format: "url"},
			"inboxUrl":     schema.String{Format: "url"},
			"imageUrl":     schema.String{Format: "url"},
			"emailAddress": schema.String{Format: "email"},
		},
	}
}

/*********************************
 * Getter Interfaces
 *********************************/

func (link *PersonLink) GetString(name string) (string, bool) {
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
 * Setter Interfaces
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
