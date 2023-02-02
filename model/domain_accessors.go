package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func DomainSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"domainId":   schema.String{Format: "objectId"},
			"themeId":    schema.String{MaxLength: 100},
			"label":      schema.String{Required: true, MinLength: 1, MaxLength: 100},
			"forward":    schema.String{Format: "url"},
			"signupForm": SignupFormSchema(),
			// "clients":    ClientSchema(),
		},
	}
}

/*********************************
 * Getter Interfaces
 *********************************/

func (domain *Domain) GetObject(name string) (any, bool) {

	switch name {
	case "signupForm":
		return &domain.SignupForm, true
	}

	return nil, false
}

func (domain Domain) GetStringOK(name string) (string, bool) {

	switch name {

	case "domainId":
		return domain.DomainID.Hex(), true

	case "themeId":
		return domain.ThemeID, true

	case "label":
		return domain.Label, true

	case "forward":
		return domain.Forward, true
	}

	return "", false
}

/*********************************
 * Setter Interfaces
 *********************************/

func (domain *Domain) SetString(name string, value string) bool {

	switch name {

	case "domainId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			domain.DomainID = objectID
			return true
		}

	case "themeId":
		domain.ThemeID = value
		return true

	case "label":
		domain.Label = value
		return true

	case "forward":
		domain.Forward = value
		return true
	}

	return false
}
