package config

import (
	"testing"

	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/require"
)

func TestSMTPSchema(t *testing.T) {

	d := NewSMTPConnection()
	s := schema.New(SMTPConnectionSchema())

	table := []tableTestItem{
		{"hostname", "SMTP_HOSTNAME", nil},
		{"username", "SMTP_USERNAME", nil},
		{"password", "SMTP_PASSWORD", nil},
		{"port", "443", 443},
		{"tls", "false", false},
	}

	tableTest_Schema(t, &s, &d, table)
}

// TestSMTPValidate guards against a regression where Validate() passed the
// connection by value. The schema resolves properties through GetPointer (a
// pointer-receiver method), so a value copy fails the PointerGetter interface
// and every populated connection was rejected -- silently disabling all email.
func TestSMTPValidate(t *testing.T) {

	// A fully-populated connection must validate cleanly.
	smtp := SMTPConnection{
		Hostname: "mailhog",
		Username: "u",
		Password: "p",
		Port:     1025,
	}

	require.Nil(t, smtp.Validate())

	// Server() gates on Validate(), so it must now return a usable client.
	server, ok := smtp.Server()
	require.True(t, ok)
	require.NotNil(t, server)

	// An empty connection is caught by IsNil() (not Validate), so it too passes
	// the schema -- callers short-circuit on IsNil before ever calling Server().
	empty := NewSMTPConnection()
	require.Nil(t, empty.Validate())
	require.True(t, empty.IsNil())
}
