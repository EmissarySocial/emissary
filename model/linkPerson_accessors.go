package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func PersonLinkSchema() schema.Element {

	return schema.Object{
		Properties: schema.ElementMap{
			"userId":       schema.String{Format: "objectId"},
			"name":         schema.String{MaxLength: 128},
			"profileUrl":   schema.String{Format: "url", MaxLength: 1024},
			"inboxUrl":     schema.String{Format: "url", MaxLength: 1024},
			"iconUrl":      schema.String{Format: "url", MaxLength: 1024},
			"emailAddress": schema.String{Format: "email", MaxLength: 128},
		},
	}
}

/********************************
 * Getter/Setter Interfaces
 ********************************/

func (link *PersonLink) GetPointer(name string) (any, bool) {
	switch name {

	case "name":
		return &link.Name, true

	case "profileUrl":
		return &link.ProfileURL, true

	case "inboxUrl":
		return &link.InboxURL, true

	case "emailAddress":
		return &link.EmailAddress, true

	case "iconUrl":
		return &link.IconURL, true

	}

	return nil, false
}

func (link *PersonLink) GetStringOK(name string) (string, bool) {
	switch name {

	case "userId":
		return link.UserID.Hex(), true

	}

	return "", false
}

func (link *PersonLink) SetString(name string, value string) bool {
	switch name {

	case "userId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			link.UserID = objectID
			return true
		}
	}

	return false
}
