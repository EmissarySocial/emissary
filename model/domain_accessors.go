package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func DomainSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"domainId":         schema.String{Format: "objectId"},
			"iconId":           schema.String{Format: "objectId"},
			"iconUrl":          schema.String{Format: "url"}, // virtual field
			"themeId":          schema.String{MaxLength: 128},
			"registrationId":   schema.String{MaxLength: 128},
			"inboxId":          schema.String{MaxLength: 128},
			"outboxId":         schema.String{MaxLength: 128},
			"label":            schema.String{MaxLength: 128},
			"description":      schema.String{MaxLength: 1024},
			"forward":          schema.String{Format: "url", Required: false},
			"data":             schema.Object{Wildcard: schema.String{}},
			"colorMode":        schema.String{Enum: []string{DomainColorModeAuto, DomainColorModeLight, DomainColorModeDark}},
			"registrationData": schema.Object{Wildcard: schema.String{}},
		},
	}
}

/********************************
 * Getter/Setter Interfaces
 ********************************/

func (domain *Domain) GetPointer(name string) (any, bool) {

	switch name {

	case "registrationId":
		return &domain.RegistrationID, true

	case "inboxId":
		return &domain.InboxID, true

	case "outboxId":
		return &domain.OutboxID, true

	case "registrationData":
		return &domain.RegistrationData, true

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

	case "iconId":
		return domain.IconID.Hex(), true

	case "iconUrl":
		return domain.IconURL(), true
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

	case "iconId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			domain.IconID = objectID
			return true
		}

	case "iconUrl":
		return true
	}

	return false
}
