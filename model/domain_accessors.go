package model

import (
	"github.com/benpate/form"
	"github.com/benpate/rosetta/schema"
)

// DomainSchema returns the rosetta schema that describes a Domain
func DomainSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"domainId":             schema.String{Format: "objectId"},
			"iconId":               schema.String{Format: "objectId"},
			"imageId":              schema.String{Format: "objectId"},
			"iconUrl":              schema.String{}, // virtual field
			"imageUrl":             schema.String{}, // virtual field
			"themeId":              schema.String{MaxLength: 128},
			"registrationId":       schema.String{MaxLength: 128},
			"inboxId":              schema.String{MaxLength: 128},
			"outboxId":             schema.String{MaxLength: 128},
			"label":                schema.String{MaxLength: 128},
			"description":          schema.String{MaxLength: 1024},
			"forward":              schema.String{Format: "url", Required: false},
			"data":                 schema.Object{Wildcard: schema.String{MaxLength: 4096}},
			"themeData":            schema.Object{Wildcard: schema.String{MaxLength: 1048576}},
			"colorMode":            schema.String{Enum: []string{DomainColorModeAuto, DomainColorModeLight, DomainColorModeDark}},
			"mlsMode":              schema.String{Enum: []string{DomainMLSModeAll, DomainMLSModeGroups, DomainMLSModeNone}},
			"defaultAnonymous":     schema.String{MaxLength: 128},
			"defaultAuthenticated": schema.String{MaxLength: 128},
			"defaultOwner":         schema.String{MaxLength: 128},
			"mlsGroupIds":          schema.String{MaxLength: 2048},
			"syndication":          schema.Array{Items: form.LookupCodeSchema()},
			"registrationData":     schema.Object{Wildcard: schema.String{MaxLength: 8192}},
			"startupTasks":         schema.Array{Items: schema.String{MaxLength: 32}, MaxLength: 16},
			"stateId":              schema.String{Enum: []string{DomainStateStartup, DomainStateLive}},
		},
	}
}

/********************************
 * Getter/Setter Interfaces
 ********************************/

// GetStringOK returns the named property. Implements schema.StringGetter.
func (domain Domain) GetStringOK(name string) (string, bool) {

	switch name {

	case "domainId":
		return domain.DomainID.Hex(), true

	case "iconId":
		return domain.IconID.Hex(), true

	case "imageId":
		return domain.ImageID.Hex(), true

	case "iconUrl":
		return domain.IconURL(), true

	case "imageUrl":
		return domain.ImageURL(), true

	case "mlsGroupIds":
		return domain.MLSGroupIDs.Join(","), true
	}

	return "", false
}
