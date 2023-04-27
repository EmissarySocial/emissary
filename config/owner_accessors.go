package config

import (
	"github.com/benpate/rosetta/schema"
)

func OwnerSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"displayName":    schema.String{MaxLength: 100, Required: true},
			"username":       schema.String{MaxLength: 255, Required: true},
			"emailAddress":   schema.String{MaxLength: 255, Format: "email", Required: true},
			"phoneNumber":    schema.String{MaxLength: 20},
			"mailingAddress": schema.String{MaxLength: 1000},
		},
	}
}

func (owner Owner) GetStringOK(name string) (string, bool) {

	switch name {

	case "displayName":
		return owner.DisplayName, true

	case "username":
		return owner.Username, true

	case "emailAddress":
		return owner.EmailAddress, true

	case "phoneNumber":
		return owner.PhoneNumber, true

	case "mailingAddress":
		return owner.MailingAddress, true
	}

	return "", false
}

func (owner *Owner) SetString(name string, value string) bool {

	switch name {

	case "displayName":
		owner.DisplayName = value
		return true

	case "username":
		owner.Username = value
		return true

	case "emailAddress":
		owner.EmailAddress = value
		return true

	case "phoneNumber":
		owner.PhoneNumber = value
		return true

	case "mailingAddress":
		owner.MailingAddress = value
		return true
	}

	return false
}
