package model

import (
	"testing"

	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/schema"
)

func TestMessageSchema(t *testing.T) {

	activity := NewMessage()
	s := schema.New(MessageSchema())

	table := []tableTestItem{
		{"messageId", "123456781234567812345678", nil},
		{"userId", "876543218765432187654321", nil},
		{"socialRole", "Article", nil},
		{"origin.url", "https://origin.url", nil},
		{"references.0.url", "https://first.reference.url", nil},
		{"references.1.url", "https://another.reference.url", nil},
		{"folderId", "123456123456123456123456", nil},
		{"publishDate", "123", int64(123)},
		{"readDate", 456, int64(456)},
		{"rank", "123", int64(123)},
		{"myResponse", vocab.ActivityTypeLike, nil},
	}

	tableTest_Schema(t, &s, &activity, table)
}
