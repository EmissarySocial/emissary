package sync

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// These tests ask a REAL MongoDB which index it chooses for the Mastodon API's lists, because
// a partial index is used only when the query names the index's own filter terms.

// insertOutboxStreams stores 300 Streams by one author, of which every tenth is in the outbox
func insertOutboxStreams(t *testing.T, collection *mongo.Collection, authorID primitive.ObjectID) {

	t.Helper()

	documents := make([]any, 0, 300)

	for index := range 300 {
		streamID := primitive.NewObjectID()

		// Every live Stream needs a token of its own, which idx_Stream_Token enforces
		documents = append(documents, bson.M{
			"_id":          streamID,
			"token":        streamID.Hex(),
			"attributedTo": bson.M{"userId": authorID},
			"templateId":   "outbox-message",
			"isOutbox":     index%10 == 0,
			"createDate":   int64(index),
			"publishDate":  int64(index),
			"deleteDate":   0,
		})
	}

	_, err := collection.InsertMany(context.Background(), documents)
	require.NoError(t, err)
}

// explainStreamFind returns the winning plan for a sorted, limited query on the Stream collection
func explainStreamFind(t *testing.T, database *mongo.Database, filter bson.M, sort bson.D) string {

	t.Helper()

	command := bson.D{
		{Key: "explain", Value: bson.D{
			{Key: "find", Value: "Stream"},
			{Key: "filter", Value: filter},
			{Key: "sort", Value: sort},
			{Key: "limit", Value: 20},
		}},
		{Key: "verbosity", Value: "queryPlanner"},
	}

	plan := bson.M{}
	require.NoError(t, database.RunCommand(context.Background(), command).Decode(&plan))

	planner, isMap := plan["queryPlanner"].(bson.M)
	require.True(t, isMap)

	winningPlan, err := bson.MarshalExtJSON(planner["winningPlan"], false, false)
	require.NoError(t, err)

	return string(winningPlan)
}

// TestStreamIndex_OutboxByAuthor verifies that one User's posts are listed from the partial
// index, with no sort in memory
func TestStreamIndex_OutboxByAuthor(t *testing.T) {

	database := newResponseTestDatabase(t)
	ctx := context.Background()
	authorID := primitive.NewObjectID()

	require.NoError(t, Stream(ctx, database))
	insertOutboxStreams(t, database.Collection("Stream"), authorID)

	// The same terms that Stream.QueryByUser sends for the owner
	filter := bson.M{"attributedTo.userId": authorID, "isOutbox": true, "deleteDate": 0}
	plan := explainStreamFind(t, database, filter, bson.D{{Key: "createDate", Value: -1}})

	require.Contains(t, plan, "idx_Stream_OutboxByAuthor", plan)
	require.NotContains(t, plan, "COLLSCAN", plan)
	require.NotContains(t, plan, "SORT", plan)
}

// TestStreamIndex_OutboxByDate verifies that the local timeline is listed from the partial
// index, with no sort in memory
func TestStreamIndex_OutboxByDate(t *testing.T) {

	database := newResponseTestDatabase(t)
	ctx := context.Background()

	require.NoError(t, Stream(ctx, database))
	insertOutboxStreams(t, database.Collection("Stream"), primitive.NewObjectID())

	// The same terms that Stream.QueryPublic sends for an anonymous caller
	filter := bson.M{
		"templateId":   "outbox-message",
		"isOutbox":     true,
		"defaultAllow": bson.M{"$in": bson.A{primitive.NewObjectID()}},
		"deleteDate":   0,
	}

	plan := explainStreamFind(t, database, filter, bson.D{{Key: "createDate", Value: -1}})

	require.Contains(t, plan, "idx_Stream_OutboxByDate", plan)
	require.NotContains(t, plan, "COLLSCAN", plan)
	require.NotContains(t, plan, "SORT", plan)
}

// TestStreamIndex_OutboxNeedsTheFlag verifies that a query which does not name the flag cannot
// use either partial index, so that nobody removes the term as redundant
func TestStreamIndex_OutboxNeedsTheFlag(t *testing.T) {

	database := newResponseTestDatabase(t)
	ctx := context.Background()

	require.NoError(t, Stream(ctx, database))
	insertOutboxStreams(t, database.Collection("Stream"), primitive.NewObjectID())

	filter := bson.M{"templateId": "outbox-message", "deleteDate": 0}
	plan := explainStreamFind(t, database, filter, bson.D{{Key: "createDate", Value: -1}})

	require.NotContains(t, plan, "idx_Stream_Outbox", plan)
}
