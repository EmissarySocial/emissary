package config

import (
	"testing"

	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/require"
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
	}

	tableTest_Schema(t, &s, &d, table)
}

// TestDomainSchema_MasterKeyNotSettable proves that the masterKey cannot be read or
// written through the schema, so a crafted POST cannot overwrite key material (BUG-110).
func TestDomainSchema_MasterKeyNotSettable(t *testing.T) {

	d := NewDomain()
	s := schema.New(DomainSchema())

	original := d.MasterKey
	require.NotEmpty(t, original)

	// Setting masterKey through the schema must fail and must not change the stored key
	require.Error(t, s.Set(&d, "masterKey", "1234567890123456789012345678901234567890123456789012345678901234"))
	require.Equal(t, original, d.MasterKey)

	// Reading masterKey through the schema must fail, too
	_, err := s.Get(&d, "masterKey")
	require.Error(t, err)
}
