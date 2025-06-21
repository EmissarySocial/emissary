package config

import (
	"github.com/benpate/rosetta/schema"
)

func DomainSchema() schema.Element {

	return schema.Object{
		Properties: schema.ElementMap{
			"label":         schema.String{MaxLength: 100, Required: true},
			"hostname":      schema.String{MaxLength: 255, Required: true},
			"connectString": schema.String{MaxLength: 1000, Required: true},
			"databaseName":  schema.String{Pattern: `[a-zA-Z0-9-_]+`, Required: true},
			"smtp":          SMTPConnectionSchema(),
			"owner":         OwnerSchema(),
			"masterKey":     schema.String{MinLength: 64, MaxLength: 64, Pattern: "^[0-9A-Fa-f]{64}$"},
		},
	}
}

/******************************************
 * Getter/Setter Interfaces
 ******************************************/

func (domain *Domain) GetPointer(name string) (any, bool) {

	switch name {

	case "smtp":
		return &domain.SMTPConnection, true

	case "owner":
		return &domain.Owner, true

	case "label":
		return &domain.Label, true

	case "hostname":
		return &domain.Hostname, true

	case "connectString":
		return &domain.ConnectString, true

	case "databaseName":
		return &domain.DatabaseName, true

	case "masterKey":
		return &domain.MasterKey, true
	}

	return nil, false
}
