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
		{"label", "LABEL", nil},
		{"description", "DESCRIPTION", nil},
		{"forward", "https://other.site", nil},
		{"signupForm.title", "SIGNUP TITLE", nil},
		{"signupForm.message", "SIGNUP MESSAGE", nil},
		{"signupForm.groupId", "123456781234567812345678", nil},
		{"signupForm.active", "true", true},
		{"data.custom", "CUSTOM", nil},
		{"data.value", "VALUE", nil},
		{"colorMode", "LIGHT", nil},
	}

	tableTest_Schema(t, &s, &domain, table)
}
