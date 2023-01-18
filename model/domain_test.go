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
		{"label", "LABEL", nil},
		{"headerHtml", "HEADER", nil},
		{"footerHtml", "FOOTER", nil},
		{"customCss", "CSS", nil},
		{"bannerUrl", "http://banner.url", nil},
		{"forward", "https://other.site", nil},
		{"signupForm.title", "SIGNUP TITLE", nil},
		{"signupForm.message", "SIGNUP MESSAGE", nil},
		{"signupForm.groupId", "123456781234567812345678", nil},
		{"signupForm.active", "true", true},
		{"socialLinks", "true", true},
	}

	tableTest_Schema(t, &s, &domain, table)
}
