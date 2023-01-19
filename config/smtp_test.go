package config

import (
	"testing"

	"github.com/benpate/rosetta/schema"
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
