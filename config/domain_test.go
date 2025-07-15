package config

import (
	"testing"

	"github.com/benpate/rosetta/schema"
)

func TestDomainSchema(t *testing.T) {

	d := NewDomain()
	s := schema.New(DomainSchema())

	table := []tableTestItem{
		{"label", "LABEL", nil},
		{"hostname", "HOSTNAME", nil},
		{"connectString", "CONNECT_STRING", nil},
		{"databaseName", "DBNAME", nil},
		{"smtp.hostname", "SMTP_HOSTNAME", nil},
		{"smtp.username", "SMTP_USERNAME", nil},
		{"smtp.password", "SMTP_PASSWORD", nil},
		{"smtp.port", "443", 443},
		{"smtp.tls", "false", false},
		{"owner.displayName", "OWNER NAME", nil},
		{"owner.username", "OWNER USERNAME", nil},
		{"owner.emailAddress", "owner@email.address", nil},
		{"owner.phoneNumber", "123-456-7890", nil},
		{"owner.mailingAddress", "1234 Owner Street, Ownerville, OW 00000", nil},
		{"masterKey", "1234567890123456789012345678901234567890123456789012345678901234", nil},
	}

	tableTest_Schema(t, &s, &d, table)
}
