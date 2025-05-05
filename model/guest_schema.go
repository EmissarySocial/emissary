package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func GuestSchema() schema.Element {

	return schema.Object{
		Properties: schema.ElementMap{
			"guestId":         schema.String{Format: "objectId"},
			"name":            schema.String{},
			"emailAddress":    schema.String{},
			"fediverseHandle": schema.String{},
		},
	}
}

/*********************************
 * Getter/Setter Interfaces
 *********************************/

func (guest *Guest) GetPointer(name string) (interface{}, bool) {
	switch name {

	case "name":
		return &guest.Name, true

	case "emailAddress":
		return &guest.EmailAddress, true

	case "fediverseHandle":
		return &guest.FediverseHandle, true

	default:
		return nil, false
	}
}

func (guest *Guest) GetStringOK(name string) (string, bool) {
	switch name {

	case "guestId":
		return guest.GuestID.Hex(), true

	default:
		return "", false
	}
}

func (guest *Guest) SetString(name string, value string) bool {

	switch name {

	case "guestId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			guest.GuestID = objectID
			return true
		}
	}

	return false
}
