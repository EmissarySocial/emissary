package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
)

func TestNewsItemSchema(t *testing.T) {

	activity := NewNewsItem()
	s := schema.New(NewsItemSchema())

	table := []tableTestItem{
		{"newsItemId", "123456781234567812345678", nil},
		{"userId", "876543218765432187654321", nil},
		{"followingId", "abcdef218765432187654321", nil},
		{"folderId", "fedcba218765432187654321", nil},
		{"socialRole", "Article", nil},
		{"origin.url", "https://origin.url", nil},
		{"references.0.url", "https://first.reference.url", nil},
		{"references.1.url", "https://another.reference.url", nil},
		{"url", "https://message.url", nil},
		{"inReplyTo", "https://url.com", nil},
		{"stateId", "UNREAD", nil},
		{"publishDate", "123", int64(123)},
		{"readDate", 456, int64(456)},
		{"rank", "123", int64(123)},
	}

	tableTest_Schema(t, &s, &activity, table)
}
