package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
)

func TestDomainSchema(t *testing.T) {

	domain := NewDomain()
	s := schema.New(DomainSchema())

	table := []tableTestItem{
		{"domainId", "123456781234567812345678", nil},
		{"iconId", "aaa4bbb8ddd4ddd812345678", nil},
		{"themeId", "123456516253413243716253", nil},
		{"registrationId", "none", nil},
		{"inboxId", "user-inbox", nil},
		{"outboxId", "user-outbox", nil},
		{"registrationData.customA", "CUSTOM", nil},
		{"registrationData.customB", "CUSTOM", nil},
		{"registrationData.customC", "CUSTOM", nil},
		{"label", "LABEL", nil},
		{"description", "DESCRIPTION", nil},
		{"forward", "https://other.site", nil},
		{"data.custom", "CUSTOM", nil},
		{"data.value", "VALUE", nil},
		{"data.sso_active", "true", nil},
		{"data.sso_secret", "123456789-10-11-12", nil},
		{"colorMode", "LIGHT", nil},
		{"registrationData.custom", "CUSTOM", nil},
		{"registrationData.value", "VALUE", nil},
		{"syndication.0.value", "VALUE", nil},
		{"syndication.0.label", "LABEL", nil},
		{"syndication.1.description", "DESCRIPTION", nil},
		{"syndication.1.href", "https://syndication.site", nil},
	}

	tableTest_Schema(t, &s, &domain, table)
}
