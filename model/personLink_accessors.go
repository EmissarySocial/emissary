package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// PersonLinkSchema returns the rosetta schema that describes a PersonLink
func PersonLinkSchema() schema.Element {

	return schema.Object{
		Properties: schema.ElementMap{
			"userId":       schema.String{Format: "objectId"},
			"name":         schema.String{Format: "text", MaxLength: 128},
			"username":     schema.String{Format: "text", MaxLength: 128},
			"profileUrl":   schema.String{Format: "url"},
			"inboxUrl":     schema.String{Format: "url"},
			"iconUrl":      schema.String{Format: "url"},
			"emailAddress": schema.String{Format: "email", MaxLength: 128},
		},
	}
}

/********************************
 * Getter/Setter Interfaces
 ********************************/

// GetPointer returns a pointer to the named property. Implements schema.PointerGetter.
func (link *PersonLink) GetPointer(name string) (any, bool) {
	switch name {

	case "name":
		return &link.Name, true

	case "username":
		return &link.Username, true

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

// GetStringOK returns the named property. Implements schema.StringGetter.
func (link *PersonLink) GetStringOK(name string) (string, bool) {
	switch name {

	case "userId":
		return link.UserID.Hex(), true

	}

	return "", false
}

// SetString writes the named property. Implements schema.StringSetter.
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
