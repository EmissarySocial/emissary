package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func DomainSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"domainId":    schema.String{Format: "objectId", Required: true},
			"themeId":     schema.String{MaxLength: 128, Required: true},
			"label":       schema.String{MinLength: 1, MaxLength: 128, Required: true},
			"description": schema.String{MinLength: 1, MaxLength: 1024, Required: false},
			"forward":     schema.String{Format: "url", Required: false},
			"data":        schema.Object{Wildcard: schema.String{}},
			"colorMode":   schema.String{Enum: []string{DomainColorModeAuto, DomainColorModeLight, DomainColorModeDark}, Required: true},
			"signupForm":  SignupFormSchema(),
		},
	}
}

/********************************
 * Getter/Setter Interfaces
 ********************************/

func (domain *Domain) GetPointer(name string) (any, bool) {

	switch name {

	case "signupForm":
		return &domain.SignupForm, true

	case "themeId":
		return &domain.ThemeID, true

	case "label":
		return &domain.Label, true

	case "description":
		return &domain.Description, true

	case "forward":
		return &domain.Forward, true

	case "colorMode":
		return &domain.ColorMode, true

	case "data":
		return &domain.Data, true
	}

	return nil, false
}

func (domain Domain) GetStringOK(name string) (string, bool) {

	switch name {

	case "domainId":
		return domain.DomainID.Hex(), true
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
	}

	return false
}
