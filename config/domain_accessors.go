package config

import (
	"github.com/EmissarySocial/emissary/tools/random"
	"github.com/benpate/rosetta/schema"
)

func DomainSchema() schema.Element {

	// Default KEK is a slice of 64 random bytes
	keyEncryptingKey, _ := random.GenerateString(32)

	return schema.Object{
		Properties: schema.ElementMap{
			"label":            schema.String{MaxLength: 100, Required: true},
			"hostname":         schema.String{MaxLength: 255, Required: true},
			"connectString":    schema.String{MaxLength: 1000},
			"databaseName":     schema.String{Pattern: `[a-zA-Z0-9-_]+`},
			"smtp":             SMTPConnectionSchema(),
			"owner":            OwnerSchema(),
			"keyEncryptingKey": schema.String{MinLength: 32, MaxLength: 32, Default: keyEncryptingKey},
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

	case "keyEncryptingKey":
		return &domain.KeyEncryptingKey, true
	}

	return nil, false
}
