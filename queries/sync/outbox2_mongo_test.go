package sync

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// insertOutbox2 writes one raw OutboxItem document with the given url, omitting the field when url is nil
func insertOutbox2(ctx context.Context, database *mongo.Database, url any) error {

	document := bson.M{
		"_id":       primitive.NewObjectID(),
		"actorType": "User",
		"actorId":   primitive.NewObjectID(),
	}

	if url != nil {
		document["url"] = url
	}

	_, err := database.Collection("Outbox2").InsertOne(ctx, document)
	return err
}

// TestOutbox2Index_URLIsUnique verifies that one activity URL can be stored only once
func TestOutbox2Index_URLIsUnique(t *testing.T) {

	database := newResponseTestDatabase(t)
	ctx := context.Background()

	require.NoError(t, Outbox2(ctx, database))

	require.NoError(t, insertOutbox2(ctx, database, "https://example.test/@1/pub/outbox/1"))
	require.NoError(t, insertOutbox2(ctx, database, "https://example.test/@1/pub/outbox/2"))

	err := insertOutbox2(ctx, database, "https://example.test/@1/pub/outbox/1")
	require.True(t, mongo.IsDuplicateKeyError(err), "a second row with the same url must be rejected, got: %v", err)
}

// TestOutbox2Index_EmptyURLsDoNotCollide verifies that rows with no url stay out of the unique index
func TestOutbox2Index_EmptyURLsDoNotCollide(t *testing.T) {

	database := newResponseTestDatabase(t)
	ctx := context.Background()

	require.NoError(t, Outbox2(ctx, database))

	// ActivityURL returns "" for an unknown actor type, and a missing field indexes as null
	for index := 0; index < 3; index++ {
		require.NoError(t, insertOutbox2(ctx, database, ""), "empty url #%d", index+1)
		require.NoError(t, insertOutbox2(ctx, database, nil), "missing url #%d", index+1)
	}
}

// TestOutbox2Index_BuildsOverExistingRows verifies that the unique index builds on a collection that already holds rows
func TestOutbox2Index_BuildsOverExistingRows(t *testing.T) {

	database := newResponseTestDatabase(t)
	ctx := context.Background()

	require.NoError(t, insertOutbox2(ctx, database, "https://example.test/@1/pub/outbox/1"))
	require.NoError(t, insertOutbox2(ctx, database, ""))
	require.NoError(t, insertOutbox2(ctx, database, ""))
	require.NoError(t, insertOutbox2(ctx, database, nil))

	require.NoError(t, Outbox2(ctx, database))
	require.Equal(t, true, loadIndex(ctx, t, database.Collection("Outbox2"), "idx_Outbox2_URL")["unique"])
}

// TestOutbox2Index_SyncIsIdempotent verifies that a second sync keeps the unique index instead of rebuilding it
func TestOutbox2Index_SyncIsIdempotent(t *testing.T) {

	database := newResponseTestDatabase(t)
	ctx := context.Background()
	collection := database.Collection("Outbox2")

	require.NoError(t, Outbox2(ctx, database))
	before := indexCreatedAt(ctx, t, collection, "idx_Outbox2_URL")

	require.NoError(t, Outbox2(ctx, database))
	after := indexCreatedAt(ctx, t, collection, "idx_Outbox2_URL")

	require.Equal(t, before, after, "Sync rebuilt an unchanged index; the partial filter does not round-trip")

	// The declared filter must be the one MongoDB stored
	stored := loadIndex(ctx, t, collection, "idx_Outbox2_URL")
	require.Equal(t, bson.M{"url": bson.M{"$gt": ""}}, stored["partialFilterExpression"])
	require.Equal(t, true, stored["unique"])
}

// TestOutbox2Index_LookupByURLUsesTheIndex verifies that an equality match on url can use the partial index
func TestOutbox2Index_LookupByURLUsesTheIndex(t *testing.T) {

	database := newResponseTestDatabase(t)
	ctx := context.Background()

	require.NoError(t, Outbox2(ctx, database))
	require.NoError(t, insertOutbox2(ctx, database, "https://example.test/@1/pub/outbox/1"))

	// A partial index serves only queries that imply its filter, so prove that url == X implies url > ""
	var explain bson.M
	err := database.RunCommand(ctx, bson.D{
		{Key: "explain", Value: bson.D{
			{Key: "find", Value: "Outbox2"},
			{Key: "filter", Value: bson.M{"url": "https://example.test/@1/pub/outbox/1"}},
		}},
		{Key: "verbosity", Value: "queryPlanner"},
	}).Decode(&explain)
	require.NoError(t, err)

	planner, ok := explain["queryPlanner"].(bson.M)
	require.True(t, ok)
	require.Contains(t, findIndexNames(planner["winningPlan"]), "idx_Outbox2_URL")
}

// findIndexNames collects every indexName found anywhere in an explain plan
func findIndexNames(value any) []string {

	result := make([]string, 0)

	switch typed := value.(type) {

	case bson.M:
		for key, child := range typed {
			if name, ok := child.(string); ok && key == "indexName" {
				result = append(result, name)
				continue
			}
			result = append(result, findIndexNames(child)...)
		}

	case bson.A:
		for _, child := range typed {
			result = append(result, findIndexNames(child)...)
		}
	}

	return result
}
