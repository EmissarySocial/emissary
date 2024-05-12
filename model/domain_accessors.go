package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func DomainSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"domainId":    schema.String{Format: "objectId"},
			"themeId":     schema.String{MaxLength: 128},
			"signupId":    schema.String{MaxLength: 128},
			"inboxId":     schema.String{MaxLength: 128},
			"outboxId":    schema.String{MaxLength: 128},
			"signupData":  schema.Object{Wildcard: schema.String{}},
			"label":       schema.String{MaxLength: 128},
			"description": schema.String{MaxLength: 1024},
			"forward":     schema.String{Format: "url", Required: false},
			"data":        schema.Object{Wildcard: schema.String{}},
			"colorMode":   schema.String{Enum: []string{DomainColorModeAuto, DomainColorModeLight, DomainColorModeDark}},
			"signupForm":  SignupFormSchema(),
		},
	}
}

/********************************
 * Getter/Setter Interfaces
 ********************************/

func (domain *Domain) GetPointer(name string) (any, bool) {

	switch name {

	case "signupId":
		return &domain.SignupID, true

	case "inboxId":
		return &domain.InboxID, true

	case "outboxId":
		return &domain.OutboxID, true

	case "signupForm":
		return &domain.SignupForm, true

	case "signupData":
		return &domain.SignupData, true

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
