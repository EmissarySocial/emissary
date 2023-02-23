package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
)

func TestDocumentLink(t *testing.T) {

	origin := NewDocumentLink()

	s := schema.New(DocumentLinkSchema())

	table := []tableTestItem{
		{"internalId", "123412341234123412341234", nil},
		{"author.name", "TEST-AUTHOR", nil},
		{"author.profileUrl", "https://test.author.url", nil},
		{"author.imageUrl", "https://test.author.image.url", nil},
		{"url", "https://test.url", nil},
		{"type", "TEST-TYPE", nil},
		{"label", "TEST-LABEL", nil},
		{"summary", "TEST-SUMMARY", nil},
		{"imageUrl", "https://test.image.url", nil},
		{"publishDate", int64(1234567890), nil},
		{"unpublishDate", int64(1234567890), nil},
		{"updateDate", int64(1234567890), nil},
	}

	tableTest_Schema(t, &s, &origin, table)
}
