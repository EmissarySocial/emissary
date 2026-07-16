package sync

import (
	"context"
	"testing"
	"time"

	"github.com/benpate/hannibal/vocab"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// These tests drive deduplicateResponses against a REAL MongoDB, because the parts it gets wrong
// are the parts a fake cannot check: the aggregation pipeline, the $push/$group shape, and whether
// the unique index will actually build afterwards. They skip when no database is reachable, so
// `go test ./...` still passes on a machine without one.

// directConnection bypasses replica-set discovery. The local demo's replica set advertises its
// member as "host.docker.internal", which does not resolve from outside the container, so a
// discovering client would hang on server selection instead of talking to the port in front of it.
const responseTestConnection = "mongodb://localhost:27017/?directConnection=true"

// newResponseTestDatabase connects to a local MongoDB and returns a throwaway database that is
// dropped when the test ends. The test is skipped when no database is reachable.
func newResponseTestDatabase(t *testing.T) *mongo.Database {

	t.Helper()

	if testing.Short() {
		t.Skip("Skipping MongoDB integration test in -short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(responseTestConnection))

	if err != nil {
		t.Skip("Skipping: no MongoDB at " + responseTestConnection)
	}

	if err := client.Ping(ctx, nil); err != nil {
		t.Skip("Skipping: no MongoDB at " + responseTestConnection)
	}

	// A per-test database name keeps parallel runs (and the real demo data) well clear of this.
	database := client.Database("emissary_synctest_" + primitive.NewObjectID().Hex())

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = database.Drop(ctx)
		_ = client.Disconnect(ctx)
	})

	return database
}

// insertTestResponse writes one raw Response document, bypassing the service layer entirely --
// exactly as the pre-fix handlers did.
func insertTestResponse(t *testing.T, database *mongo.Database, userID primitive.ObjectID, object string, responseType string, createDate int64) primitive.ObjectID {

	t.Helper()

	responseID := primitive.NewObjectID()

	_, err := database.Collection("Response").InsertOne(context.Background(), bson.M{
		"_id":        responseID,
		"userId":     userID,
		"actor":      "https://example.test/@user",
		"object":     object,
		"type":       responseType,
		"createDate": createDate,
		"deleteDate": 0,
	})

	require.Nil(t, err)
	return responseID
}

// countTestResponses returns the number of live Responses matching a User, Object, and type.
func countTestResponses(t *testing.T, database *mongo.Database, userID primitive.ObjectID, object string, responseType string) int64 {

	t.Helper()

	count, err := database.Collection("Response").CountDocuments(context.Background(), bson.M{
		"userId": userID,
		"object": object,
		"type":   responseType,
	})

	require.Nil(t, err)
	return count
}

// The exact reported reproduction: 3 Likes and 1 Dislike on one post by one User. After the
// migration exactly one reaction survives -- the newest -- and the unique index builds.
func TestResponse_Migration_ReportedReproduction(t *testing.T) {

	database := newResponseTestDatabase(t)
	ctx := context.Background()

	userID := primitive.NewObjectID()
	const object = "https://example.test/post"

	insertTestResponse(t, database, userID, object, vocab.ActivityTypeLike, 100)
	insertTestResponse(t, database, userID, object, vocab.ActivityTypeLike, 200)
	insertTestResponse(t, database, userID, object, vocab.ActivityTypeLike, 300)
	insertTestResponse(t, database, userID, object, vocab.ActivityTypeDislike, 400)

	require.Nil(t, Response(ctx, database))

	// The Dislike is newest, so it displaces all three Likes
	require.Equal(t, int64(0), countTestResponses(t, database, userID, object, vocab.ActivityTypeLike))
	require.Equal(t, int64(1), countTestResponses(t, database, userID, object, vocab.ActivityTypeDislike))

	// The unique index must exist once the collection is clean
	indexed, err := indexExists(ctx, database.Collection("Response"), responseUniqueIndex)
	require.Nil(t, err)
	require.True(t, indexed)
}

// An Announce survives alongside a surviving Like: they are independent reactions.
func TestResponse_Migration_AnnounceCoexists(t *testing.T) {

	database := newResponseTestDatabase(t)
	ctx := context.Background()

	userID := primitive.NewObjectID()
	const object = "https://example.test/post"

	insertTestResponse(t, database, userID, object, vocab.ActivityTypeLike, 100)
	insertTestResponse(t, database, userID, object, vocab.ActivityTypeLike, 200)
	insertTestResponse(t, database, userID, object, vocab.ActivityTypeAnnounce, 300)

	require.Nil(t, Response(ctx, database))

	require.Equal(t, int64(1), countTestResponses(t, database, userID, object, vocab.ActivityTypeLike))
	require.Equal(t, int64(1), countTestResponses(t, database, userID, object, vocab.ActivityTypeAnnounce))
}

// Reactions from other Users, and on other Objects, are never touched.
func TestResponse_Migration_LeavesOtherRecordsAlone(t *testing.T) {

	database := newResponseTestDatabase(t)
	ctx := context.Background()

	userID := primitive.NewObjectID()
	otherUserID := primitive.NewObjectID()
	const object = "https://example.test/post"
	const otherObject = "https://example.test/other"

	insertTestResponse(t, database, userID, object, vocab.ActivityTypeLike, 100)
	insertTestResponse(t, database, userID, object, vocab.ActivityTypeLike, 200)
	insertTestResponse(t, database, otherUserID, object, vocab.ActivityTypeLike, 100)
	insertTestResponse(t, database, userID, otherObject, vocab.ActivityTypeLike, 100)

	require.Nil(t, Response(ctx, database))

	require.Equal(t, int64(1), countTestResponses(t, database, userID, object, vocab.ActivityTypeLike))
	require.Equal(t, int64(1), countTestResponses(t, database, otherUserID, object, vocab.ActivityTypeLike))
	require.Equal(t, int64(1), countTestResponses(t, database, userID, otherObject, vocab.ActivityTypeLike))
}

// A clean collection is left exactly as it was found.
func TestResponse_Migration_NoDuplicates(t *testing.T) {

	database := newResponseTestDatabase(t)
	ctx := context.Background()

	userID := primitive.NewObjectID()
	const object = "https://example.test/post"

	insertTestResponse(t, database, userID, object, vocab.ActivityTypeLike, 100)

	require.Nil(t, Response(ctx, database))

	require.Equal(t, int64(1), countTestResponses(t, database, userID, object, vocab.ActivityTypeLike))
}

// Once the migration has run, the unique index rejects the duplicate insert that BUG-003249
// reported -- the backstop behind SetResponse's Conflict handling.
func TestResponse_Migration_UniqueIndexRejectsDuplicate(t *testing.T) {

	database := newResponseTestDatabase(t)
	ctx := context.Background()

	userID := primitive.NewObjectID()
	const object = "https://example.test/post"

	insertTestResponse(t, database, userID, object, vocab.ActivityTypeLike, 100)
	require.Nil(t, Response(ctx, database))

	// A second Like for the same (userId, object, type) must now be refused
	_, err := database.Collection("Response").InsertOne(ctx, bson.M{
		"_id":        primitive.NewObjectID(),
		"userId":     userID,
		"object":     object,
		"type":       vocab.ActivityTypeLike,
		"createDate": 200,
		"deleteDate": 0,
	})

	require.NotNil(t, err)
	require.True(t, mongo.IsDuplicateKeyError(err))

	// ...but a Dislike is a different `type`, so the index alone cannot stop it. This is exactly
	// why mutual exclusivity has to live in service.SetResponse.
	_, err = database.Collection("Response").InsertOne(ctx, bson.M{
		"_id":        primitive.NewObjectID(),
		"userId":     userID,
		"object":     object,
		"type":       vocab.ActivityTypeDislike,
		"createDate": 300,
		"deleteDate": 0,
	})

	require.Nil(t, err)
}

// The migration is one-time: with the unique index already in place, the scan is skipped rather
// than repeated on every startup.
func TestResponse_Migration_SkippedOnceIndexed(t *testing.T) {

	database := newResponseTestDatabase(t)
	ctx := context.Background()

	require.Nil(t, Response(ctx, database))

	indexed, err := indexExists(ctx, database.Collection("Response"), responseUniqueIndex)
	require.Nil(t, err)
	require.True(t, indexed)

	// A second sync is a no-op that must not error
	require.Nil(t, Response(ctx, database))
}

// indexExists reports FALSE for an index that was never created.
func TestIndexExists_Missing(t *testing.T) {

	database := newResponseTestDatabase(t)

	indexed, err := indexExists(context.Background(), database.Collection("Response"), responseUniqueIndex)

	require.Nil(t, err)
	require.False(t, indexed)
}
