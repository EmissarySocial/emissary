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
		{"colorMode", "LIGHT", nil},
		{"registrationData.custom", "CUSTOM", nil},
		{"registrationData.value", "VALUE", nil},
	}

	tableTest_Schema(t, &s, &domain, table)
}
