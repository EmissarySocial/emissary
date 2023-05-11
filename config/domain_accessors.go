package config

import "github.com/benpate/rosetta/schema"

func DomainSchema() schema.Element {

	return schema.Object{
		Properties: schema.ElementMap{
			"label":         schema.String{MaxLength: 100, Required: true},
			"hostname":      schema.String{MaxLength: 255, Required: true},
			"connectString": schema.String{MaxLength: 1000},
			"databaseName":  schema.String{Pattern: `[a-zA-Z0-9-_]+`},
			"smtp":          SMTPConnectionSchema(),
			"owner":         OwnerSchema(),
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
	}

	return nil, false
}

func (domain Domain) GetStringOK(name string) (string, bool) {

	switch name {

	case "label":
		return domain.Label, true

	case "hostname":
		return domain.Hostname, true

	case "connectString":
		return domain.ConnectString, true

	case "databaseName":
		return domain.DatabaseName, true
	}

	return "", false
}

func (domain *Domain) SetString(name string, value string) bool {

	switch name {

	case "label":
		domain.Label = value
		return true

	case "hostname":
		domain.Hostname = value
		return true

	case "connectString":
		domain.ConnectString = value
		return true

	case "databaseName":
		domain.DatabaseName = value
		return true
	}

	return false
}
