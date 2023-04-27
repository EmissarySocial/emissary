package config

import "testing"

func TestConfig(t *testing.T) {

	c := NewConfig()
	s := Schema()

	table := []tableTestItem{
		{"providers.0.providerId", "PROVIDER_ID", nil},
		{"providers.0.clientId", "CLIENT_ID", nil},
		{"providers.0.clientSecret", "CLIENT_SECRET", nil},
		{"domains.0.label", "LABEL", nil},
		{"domains.0.hostname", "HOSTNAME", nil},
		{"domains.0.connectString", "CONNECT_STRING", nil},
		{"domains.0.databaseName", "DBNAME", nil},
		{"domains.0.smtp.hostname", "SMTP_SERVER", nil},
		{"domains.0.smtp.username", "SMTP_USERNAME", nil},
		{"domains.0.smtp.password", "SMTP_PASSWORD", nil},
		{"domains.0.smtp.port", "443", 443},
		{"domains.0.smtp.tls", "false", false},
		{"domains.0.owner.displayName", "DISPLAY_NAME", nil},
		{"domains.0.owner.username", "USERNAME", nil},
		{"domains.0.owner.emailAddress", "EMAIL_ADDRESS", nil},
		{"domains.0.owner.phoneNumber", "PHONE_NUMBER", nil},
		{"domains.0.owner.mailingAddress", "MAILING_ADDRESS", nil},
	}

	tableTest_Schema(t, &s, &c, table)
}
