package config

import "testing"

func TestConfig(t *testing.T) {

	c := NewConfig()
	s := Schema()

	table := []tableTestItem{
		{"providers.0.providerId", "GIPHY", nil},
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
		{"domains.0.owner.emailAddress", "EMAIL@ADDRESS.COM", nil},
		{"domains.0.owner.phoneNumber", "PHONE_NUMBER", nil},
		{"domains.0.owner.mailingAddress", "MAILING_ADDRESS", nil},
		{"domains.0.keyEncryptingKey", "12345678901234567890123456789012", nil},

		{"templates.0.adapter", "S3", nil},
		{"templates.0.location", "LOCATION", nil},
		{"templates.0.accessKey", "ACCESS_KEY", nil},
		{"templates.0.secretKey", "SECRET_KEY", nil},
		{"templates.0.region", "REGION", nil},
		{"templates.0.token", "TOKEN", nil},
		{"templates.0.bucket", "BUCKET", nil},
		{"templates.0.path", "PATH...", nil},

		{"emails.0.adapter", "S3", nil},
		{"emails.0.location", "LOCATION", nil},
		{"emails.0.accessKey", "ACCESS_KEY", nil},
		{"emails.0.secretKey", "SECRET_KEY", nil},
		{"emails.0.region", "REGION", nil},
		{"emails.0.token", "TOKEN", nil},
		{"emails.0.bucket", "BUCKET", nil},
		{"emails.0.path", "PATH...", nil},

		{"certificates.adapter", "S3", nil},
		{"certificates.location", "LOCATION", nil},
		{"certificates.accessKey", "ACCESS_KEY", nil},
		{"certificates.secretKey", "SECRET_KEY", nil},
		{"certificates.region", "REGION", nil},
		{"certificates.token", "TOKEN", nil},
		{"certificates.bucket", "BUCKET", nil},
		{"certificates.path", "PATH...", nil},

		{"attachmentOriginals.adapter", "S3", nil},
		{"attachmentOriginals.location", "LOCATION", nil},
		{"attachmentOriginals.accessKey", "ACCESS_KEY", nil},
		{"attachmentOriginals.secretKey", "SECRET_KEY", nil},
		{"attachmentOriginals.region", "REGION", nil},
		{"attachmentOriginals.token", "TOKEN", nil},
		{"attachmentOriginals.bucket", "BUCKET", nil},
		{"attachmentOriginals.path", "PATH...", nil},

		{"attachmentCache.adapter", "S3", nil},
		{"attachmentCache.location", "LOCATION", nil},
		{"attachmentCache.accessKey", "ACCESS_KEY", nil},
		{"attachmentCache.secretKey", "SECRET_KEY", nil},
		{"attachmentCache.region", "REGION", nil},
		{"attachmentCache.token", "TOKEN", nil},
		{"attachmentCache.bucket", "BUCKET", nil},
		{"attachmentCache.path", "PATH...", nil},

		{"adminEmail", "ADMIN@EMAIL.COM", nil},
		{"debugLevel", "Trace", nil},
		{"httpPort", "8080", 8080},
		{"httpsPort", "8443", 8443},
		{"activityPubCache.connectString", "ACTIVITY_PUB_CACHE", nil},
		{"activityPubCache.database", "ACTIVITY_PUB_CACHE", nil},
	}

	tableTest_Schema(t, &s, &c, table)
}
