package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/require"
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

// ActivityPubURL returns the item's public URI (used when serving reply collections).
func TestCollectionItem_ActivityPubURL(t *testing.T) {

	collectionItem := NewCollectionItem()
	require.Equal(t, "", collectionItem.ActivityPubURL())

	collectionItem.URI = "https://example.test/reply/1"
	require.Equal(t, "https://example.test/reply/1", collectionItem.ActivityPubURL())
}
