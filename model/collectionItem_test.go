package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
)

func TestCollectionItemSchema(t *testing.T) {

	collectionItem := NewCollectionItem()
	s := schema.New(CollectionItemSchema())

	table := []tableTestItem{
		{"collectionItemId", "123456781234567812345678", nil},
		{"collectionId", "123456781234567812345678", nil},
		{"userId", "123456781234567812345678", nil},
		{"uri", "https://test.com/test", nil},
		{"inReplyTo", "https://test.com/test/123", nil},
	}

	tableTest_Schema(t, &s, &collectionItem, table)
}
