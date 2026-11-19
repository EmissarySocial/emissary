package sync

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// These tests drive the Stream token index against a REAL MongoDB, because what they guard is
// Mongo's own behavior: whether the unique index refuses a second writer, and what its partial filter excludes.

// insertStreamToken writes one raw Stream document.  An empty token is omitted, as model.Stream's
// `bson:",omitempty"` tag causes the real writer to omit it.
func insertStreamToken(t *testing.T, collection *mongo.Collection, streamID primitive.ObjectID, token string, createDate int64, deleteDate int64) {

	t.Helper()

	document := bson.M{"_id": streamID, "createDate": createDate, "deleteDate": deleteDate}

	if token != "" {
		document["token"] = token
	}

	_, err := collection.InsertOne(context.Background(), document)
	require.NoError(t, err)
}

// TestStreamTokenIndex_DeletedStreamsReleaseTheirToken confirms that a deleted Stream's token can be used
// again, and that two live Streams still cannot share one.
func TestStreamTokenIndex_DeletedStreamsReleaseTheirToken(t *testing.T) {

	database := newResponseTestDatabase(t)
	ctx := context.Background()
	collection := database.Collection("Stream")

	require.NoError(t, Stream(ctx, database))

	insertStreamToken(t, collection, primitive.NewObjectID(), "about", 100, 999)
	insertStreamToken(t, collection, primitive.NewObjectID(), "about", 200, 0)

	_, err := collection.InsertOne(ctx, bson.M{"_id": primitive.NewObjectID(), "token": "about", "createDate": 300, "deleteDate": 0})
	require.True(t, mongo.IsDuplicateKeyError(err), "a second live Stream must not take the token")
}

// TestStreamTokenIndex_TwoWritersShareOneToken races two writers that share nothing but the database,
// and confirms that exactly one of them takes each token.
func TestStreamTokenIndex_TwoWritersShareOneToken(t *testing.T) {

	database := newResponseTestDatabase(t)
	ctx := context.Background()

	require.NoError(t, Stream(ctx, database))

	// The second writer gets its own client, as a second server would
	connectCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	otherClient, err := mongo.Connect(connectCtx, options.Client().ApplyURI(responseTestConnection))
	require.NoError(t, err)
	t.Cleanup(func() { _ = otherClient.Disconnect(context.Background()) })

	writers := []*mongo.Collection{
		database.Collection("Stream"),
		otherClient.Database(database.Name()).Collection("Stream"),
	}

	for round := range 20 {

		token := fmt.Sprintf("race-%d", round)
		results := make(chan error, len(writers))

		for _, writer := range writers {
			go func() {
				_, err := writer.InsertOne(ctx, bson.M{"_id": primitive.NewObjectID(), "token": token, "createDate": 100, "deleteDate": 0})
				results <- err
			}()
		}

		successes := 0

		for range writers {

			err := <-results

			if err == nil {
				successes++
				continue
			}

			require.True(t, mongo.IsDuplicateKeyError(err), "the losing writer must see a duplicate key, not %v", err)
		}

		require.Equal(t, 1, successes, "exactly one writer takes %q", token)
	}
}
