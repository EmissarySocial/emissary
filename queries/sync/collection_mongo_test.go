package sync

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// These tests drive the real Collection index set against a REAL MongoDB, because what they guard
// is Mongo's own behavior: how a partial filter treats an ABSENT field, and whether a unique index
// admits or rejects a given insert. A fake would only re-assert the query literal. They skip when
// no database is reachable, so `go test ./...` still passes without one.

// insertCollection writes one raw Collection document. Fields are omitted when zero-valued, exactly
// as model.Collection's `bson:",omitempty"` tags cause the real writer to omit them.
func insertCollection(ctx context.Context, database *mongo.Database, parentID primitive.ObjectID, collectionType string) error {

	document := bson.M{
		"_id":        primitive.NewObjectID(),
		"userId":     primitive.NewObjectID(),
		"deleteDate": 0,
	}

	if !parentID.IsZero() {
		document["parentId"] = parentID
	}

	if collectionType != "" {
		document["collectionType"] = collectionType
	}

	_, err := database.Collection("Collection").InsertOne(ctx, document)
	return err
}

// TestCollectionIndex_ConversationsAreUnconstrained is the regression test for the bug where a
// private message could not be sent because the unique index capped the whole DOMAIN at one
// conversation.
//
// A conversation collection (outbox_CreateOrderedCollection) has no parentId and no collectionType,
// and `omitempty` drops both from the document -- so every one of them indexed at the same
// {null, null} key. The index has no userId component, so the FIRST user to start a conversation
// blocked every other user on the domain, permanently, with a 409 Conflict.
func TestCollectionIndex_ConversationsAreUnconstrained(t *testing.T) {

	database := newResponseTestDatabase(t)
	ctx := context.Background()

	require.NoError(t, Collection(ctx, database))

	// Several users each start several conversations. Before the fix, the second insert here
	// failed with E11000 on { parentId: null, collectionType: null }.
	for index := 0; index < 5; index++ {
		require.NoError(t, insertCollection(ctx, database, primitive.NilObjectID, ""),
			"unparented conversation collection #%d must not collide", index+1)
	}
}

// TestCollectionIndex_SystemCollectionsStayUnique pins the invariant the index exists for: exactly
// one system collection per (parent, type). loadOrCreateByParent depends on this rejection to
// resolve create races, so widening the partial filter must NOT weaken it.
func TestCollectionIndex_SystemCollectionsStayUnique(t *testing.T) {

	database := newResponseTestDatabase(t)
	ctx := context.Background()

	require.NoError(t, Collection(ctx, database))

	parentID := primitive.NewObjectID()

	require.NoError(t, insertCollection(ctx, database, parentID, "Replies"))

	// The same (parent, type) is still a duplicate -- this is the create-race rejection.
	require.Error(t, insertCollection(ctx, database, parentID, "Replies"),
		"a duplicate (parentId, collectionType) must still be rejected")

	// A different type under the same parent is a different collection.
	require.NoError(t, insertCollection(ctx, database, parentID, "Likes"))

	// The same type under a different parent is a different collection.
	require.NoError(t, insertCollection(ctx, database, primitive.NewObjectID(), "Replies"))
}

// TestCollectionIndex_SoftDeleteDoesNotBlock pins the other half of the partial filter: a
// soft-deleted collection must not block re-creating a live one with the same key.
func TestCollectionIndex_SoftDeleteDoesNotBlock(t *testing.T) {

	database := newResponseTestDatabase(t)
	ctx := context.Background()

	require.NoError(t, Collection(ctx, database))

	parentID := primitive.NewObjectID()

	_, err := database.Collection("Collection").InsertOne(ctx, bson.M{
		"_id":            primitive.NewObjectID(),
		"parentId":       parentID,
		"collectionType": "Replies",
		"deleteDate":     1234,
	})
	require.NoError(t, err)

	require.NoError(t, insertCollection(ctx, database, parentID, "Replies"),
		"a soft-deleted collection must not block a fresh one with the same key")
}

// TestCollectionIndex_SyncIsIdempotent guards against index churn. tools/indexer.Sync drops and
// recreates any index whose stored definition does not compare equal to the declared one, so a
// partial filter that fails to round-trip drops and rebuilds this UNIQUE index on every single
// boot -- wasteful, and each rebuild opens a window with no uniqueness enforcement at all. This is
// not hypothetical: writing the nested `$exists` operator as a bson.D instead of a bson.M does
// exactly that, because indexer.convertMapValue normalizes maps but Mongo returns the stored
// operator as one.
//
// Comparing the two DEFINITIONS cannot detect this -- a rebuilt index has an identical definition.
// $indexStats.accesses.since is the observable that distinguishes them: it is the moment stats
// collection began for that index, and a drop-and-recreate resets it.
func TestCollectionIndex_SyncIsIdempotent(t *testing.T) {

	database := newResponseTestDatabase(t)
	ctx := context.Background()

	require.NoError(t, Collection(ctx, database))
	before := indexCreatedAt(ctx, t, database, "idx_Collection_Parent_Type")

	require.NoError(t, Collection(ctx, database))
	after := indexCreatedAt(ctx, t, database, "idx_Collection_Parent_Type")

	require.Equal(t, before, after,
		"Sync rebuilt an unchanged index -- the declared partial filter does not round-trip through indexer.compareModel")

	// The declared filter must be what Mongo actually stored. A silently-dropped condition would
	// leave the domain-wide conversation cap in place while the tests above still passed against
	// a freshly-created index.
	stored := loadIndex(ctx, t, database, "idx_Collection_Parent_Type")

	require.Equal(t,
		bson.M{"deleteDate": int32(0), "parentId": bson.M{"$exists": true}},
		stored["partialFilterExpression"],
	)
	require.Equal(t, true, stored["unique"])
}

// indexCreatedAt returns the moment MongoDB began collecting stats for the named index, which
// resets whenever the index is dropped and recreated.
func indexCreatedAt(ctx context.Context, t *testing.T, database *mongo.Database, name string) time.Time {

	t.Helper()

	cursor, err := database.Collection("Collection").Aggregate(ctx, mongo.Pipeline{
		bson.D{{Key: "$indexStats", Value: bson.D{}}},
	})
	require.NoError(t, err)

	var stats []bson.M
	require.NoError(t, cursor.All(ctx, &stats))

	for _, stat := range stats {
		if stat["name"] != name {
			continue
		}

		accesses, ok := stat["accesses"].(bson.M)
		require.True(t, ok, "$indexStats returned no accesses for "+name)

		since, ok := accesses["since"].(primitive.DateTime)
		require.True(t, ok, "$indexStats returned no accesses.since for "+name)

		return since.Time()
	}

	require.Fail(t, "index not found in $indexStats: "+name)
	return time.Time{}
}

// loadIndex returns one index's stored definition by name.
func loadIndex(ctx context.Context, t *testing.T, database *mongo.Database, name string) bson.M {

	t.Helper()

	cursor, err := database.Collection("Collection").Indexes().List(ctx)
	require.NoError(t, err)

	var indexes []bson.M
	require.NoError(t, cursor.All(ctx, &indexes))

	for _, index := range indexes {
		if index["name"] == name {
			return index
		}
	}

	require.Fail(t, "index not found: "+name)
	return nil
}
