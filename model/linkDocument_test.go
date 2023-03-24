package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
)

func TestDocumentLink(t *testing.T) {

	origin := NewDocumentLink()

	s := schema.New(DocumentLinkSchema())

	table := []tableTestItem{
		{"url", "https://test.url", nil},
		{"label", "TEST-LABEL", nil},
		{"summary", "TEST-SUMMARY", nil},
		{"imageUrl", "https://test.image.url", nil},
		{"attributedTo.0.name", "TEST-AUTHOR-NAME", nil},
		{"attributedTo.0.profileUrl", "https://test.author.url", nil},
		{"attributedTo.1.name", "TEST-AUTHOR-NAME-2", nil},
	}

	tableTest_Schema(t, &s, &origin, table)
}
