package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/require"
)

// TestCollectionItemSchema returns the rosetta schema that describes a TestCollectionItem
func TestCollectionItemSchema(t *testing.T) {

	collectionItem := NewCollectionItem()
	s := schema.New(CollectionItemSchema())

	table := []tableTestItem{
		{"collectionItemId", "123456781234567812345678", nil},
		{"collectionId", "123456781234567812345678", nil},
		{"userId", "123456781234567812345678", nil},
		{"parentId", "123456781234567812345678", nil},
		{"collectionType", "Context", nil},
		{"uri", "https://test.com/test", nil},
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
