package sync

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// These tests drive the client_id index against a REAL MongoDB, because what they guard is Mongo's own
// behavior: whether the unique index refuses a second client, and what its partial filter excludes.

// TestOAuthClientURLIndex_OneLiveClientPerURL confirms that a URL client_id names one live client, while
// registered clients, which have no clientUrl, and deleted clients are left out
func TestOAuthClientURLIndex_OneLiveClientPerURL(t *testing.T) {

	database := newResponseTestDatabase(t)
	collection := database.Collection("OAuthClient")

	require.NoError(t, OAuthClient(context.Background(), database))

	_, err := insertOAuthClient(collection, "https://fed.brid.gy/", 0)
	require.NoError(t, err)

	// RULE: A second live client for the same client_id is refused
	_, err = insertOAuthClient(collection, "https://fed.brid.gy/", 0)
	require.True(t, mongo.IsDuplicateKeyError(err), "a second live client must be refused, not %v", err)

	// Registered clients have no clientUrl, and there may be any number of them
	for range 3 {
		_, err := insertOAuthClient(collection, "", 0)
		require.NoError(t, err)
	}

	// A deleted client releases its client_id
	_, err = insertOAuthClient(collection, "https://app.example.com/client.json", 999)
	require.NoError(t, err)

	_, err = insertOAuthClient(collection, "https://app.example.com/client.json", 0)
	require.NoError(t, err)
}

// TestOAuthClient_RemovesDuplicatesBeforeIndexing confirms that clients sharing a client_id from before the
// index existed are reduced to the oldest, so the index builds, and that a later start changes nothing
func TestOAuthClient_RemovesDuplicatesBeforeIndexing(t *testing.T) {

	database := newResponseTestDatabase(t)
	collection := database.Collection("OAuthClient")
	ctx := context.Background()

	// Three clients for one client_id, and one for another, written before any index
	oldest, err := insertOAuthClient(collection, "https://fed.brid.gy/", 0)
	require.NoError(t, err)

	newer, err := insertOAuthClient(collection, "https://fed.brid.gy/", 0)
	require.NoError(t, err)

	newest, err := insertOAuthClient(collection, "https://fed.brid.gy/", 0)
	require.NoError(t, err)

	other, err := insertOAuthClient(collection, "https://app.example.com/client.json", 0)
	require.NoError(t, err)

	require.NoError(t, OAuthClient(ctx, database))

	// The oldest stays live, and the others are deleted, not removed
	require.Equal(t, int64(0), deleteDateOf(t, collection, oldest))
	require.Positive(t, deleteDateOf(t, collection, newer))
	require.Positive(t, deleteDateOf(t, collection, newest))
	require.Equal(t, int64(0), deleteDateOf(t, collection, other))

	indexed, err := indexExists(ctx, collection, oauthClientURLIndex)
	require.NoError(t, err)
	require.True(t, indexed)

	// A later start finds the index, and touches nothing
	require.NoError(t, OAuthClient(ctx, database))
	require.Equal(t, int64(0), deleteDateOf(t, collection, oldest))
}

// insertOAuthClient writes one raw OAuthClient document
func insertOAuthClient(collection *mongo.Collection, clientURL string, deleteDate int64) (primitive.ObjectID, error) {

	clientID := primitive.NewObjectID()

	_, err := collection.InsertOne(context.Background(), bson.M{
		"_id":        clientID,
		"clientUrl":  clientURL,
		"name":       "Test Client",
		"deleteDate": deleteDate,
	})

	return clientID, err
}

// deleteDateOf returns the deleteDate stored on one client
func deleteDateOf(t *testing.T, collection *mongo.Collection, clientID primitive.ObjectID) int64 {

	t.Helper()

	result := struct {
		DeleteDate int64 `bson:"deleteDate"`
	}{}

	require.NoError(t, collection.FindOne(context.Background(), bson.M{"_id": clientID}).Decode(&result))
	return result.DeleteDate
}
