package model

import (
	"testing"

	"github.com/benpate/rosetta/schema"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

// TestCollectionItemSchema_CollectionTypes pins that a CollectionItem validates for every collection
// type, Quotes included, and is refused for an unknown one
func TestCollectionItemSchema_CollectionTypes(t *testing.T) {

	s := schema.New(CollectionItemSchema())

	for _, collectionType := range []string{CollectionTypeContext, CollectionTypeReplies, CollectionTypeLikes, CollectionTypeDislikes, CollectionTypeShares, CollectionTypeQuotes} {
		collectionItem := NewCollectionItem()
		collectionItem.CollectionID = primitive.NewObjectID()
		collectionItem.UserID = primitive.NewObjectID()
		collectionItem.ParentID = primitive.NewObjectID()
		collectionItem.CollectionType = collectionType
		collectionItem.URI = "https://remote.example/notes/1"

		_, err := s.Validate(&collectionItem)
		require.NoError(t, err, collectionType)
	}

	require.NotNil(t, s.Set(&CollectionItem{}, "collectionType", "Bookmarks"), "an unknown collection type is refused")
}

// TestCollectionItem_AccessLister pins that only the User who owns the item's Collection passes as its
// author and as "self", which is what lets a template action guard the item with either role
func TestCollectionItem_AccessLister(t *testing.T) {

	ownerID := primitive.NewObjectID()

	collectionItem := NewCollectionItem()
	collectionItem.UserID = ownerID

	var accessLister AccessLister = &collectionItem

	require.Equal(t, "default", accessLister.State())
	require.True(t, accessLister.IsAuthor(ownerID))
	require.True(t, accessLister.IsMyself(ownerID))

	otherID := primitive.NewObjectID()
	require.False(t, accessLister.IsAuthor(otherID), "another User")
	require.False(t, accessLister.IsMyself(otherID), "another User")

	// An item with no owner belongs to nobody, not to the anonymous visitor
	unowned := NewCollectionItem()
	require.False(t, unowned.IsAuthor(primitive.NilObjectID))
	require.False(t, unowned.IsMyself(primitive.NilObjectID))

	require.Empty(t, collectionItem.RolesToPrivilegeIDs("author"))
}
