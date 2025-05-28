package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func IdentitySchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"identityId":            schema.String{Format: "objectId", Required: true},
			"name":                  schema.String{Required: true, MinLength: 1, MaxLength: 100},
			"iconUrl":               schema.String{Format: "url", Required: false, MaxLength: 500},
			"emailAddress":          schema.String{Format: "email", Required: false, MaxLength: 200},
			"webfingerHandle":       schema.String{Format: "webfinger", Required: false, MaxLength: 200},
			"emailVerifiedDate":     schema.Integer{Required: false},
			"webfingerVerifiedDate": schema.Integer{Required: false},
		},
	}
}

/********************************
 * Getter/Setter Interfaces
 ********************************/

func (identity *Identity) GetPointer(name string) (any, bool) {

	switch name {

	case "name":
		return &identity.Name, true

	case "iconUrl":
		return &identity.IconURL, true

	case "emailAddress":
		return &identity.EmailAddress, true

	case "webfingerHandle":
		return &identity.WebFingerHandle, true

	}

	return nil, false
}

func (identity Identity) GetStringOK(name string) (string, bool) {

	switch name {

	case "identityId":
		return identity.IdentityID.Hex(), true
	}

	return "", false
}

func (identity *Identity) SetString(name string, value string) bool {

	switch name {

	case "identityId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			identity.IdentityID = objectID
			return true
		}
	}

	return false
}
