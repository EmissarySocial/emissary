package sync

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// insertOutboxItem writes one raw OutboxItem document with the given url, omitting the field when url is nil
func insertOutboxItem(ctx context.Context, database *mongo.Database, url any) error {

	document := bson.M{
		"_id":       primitive.NewObjectID(),
		"actorType": "User",
		"actorId":   primitive.NewObjectID(),
	}

	if url != nil {
		document["url"] = url
	}

	_, err := database.Collection("OutboxItem").InsertOne(ctx, document)
	return err
}

// TestOutboxItemIndex_URLIsUnique verifies that one activity URL can be stored only once
func TestOutboxItemIndex_URLIsUnique(t *testing.T) {

	database := newResponseTestDatabase(t)
	ctx := context.Background()

	require.NoError(t, OutboxItem(ctx, database))

	require.NoError(t, insertOutboxItem(ctx, database, "https://example.test/@1/pub/outbox/1"))
	require.NoError(t, insertOutboxItem(ctx, database, "https://example.test/@1/pub/outbox/2"))

	err := insertOutboxItem(ctx, database, "https://example.test/@1/pub/outbox/1")
	require.True(t, mongo.IsDuplicateKeyError(err), "a second row with the same url must be rejected, got: %v", err)
}

// TestOutboxItemIndex_EmptyURLsDoNotCollide verifies that rows with no url stay out of the unique index
func TestOutboxItemIndex_EmptyURLsDoNotCollide(t *testing.T) {

	database := newResponseTestDatabase(t)
	ctx := context.Background()

	require.NoError(t, OutboxItem(ctx, database))

	// ActivityURL returns "" for an unknown actor type, and a missing field indexes as null
	for index := 0; index < 3; index++ {
		require.NoError(t, insertOutboxItem(ctx, database, ""), "empty url #%d", index+1)
		require.NoError(t, insertOutboxItem(ctx, database, nil), "missing url #%d", index+1)
	}
}

// TestOutboxItemIndex_BuildsOverExistingRows verifies that the unique index builds on a collection that already holds rows
func TestOutboxItemIndex_BuildsOverExistingRows(t *testing.T) {

	database := newResponseTestDatabase(t)
	ctx := context.Background()

	require.NoError(t, insertOutboxItem(ctx, database, "https://example.test/@1/pub/outbox/1"))
	require.NoError(t, insertOutboxItem(ctx, database, ""))
	require.NoError(t, insertOutboxItem(ctx, database, ""))
	require.NoError(t, insertOutboxItem(ctx, database, nil))

	require.NoError(t, OutboxItem(ctx, database))
	require.Equal(t, true, loadIndex(ctx, t, database.Collection("OutboxItem"), "idx_OutboxItem_URL")["unique"])
}

// TestOutboxItemIndex_SyncIsIdempotent verifies that a second sync keeps the unique index instead of rebuilding it
func TestOutboxItemIndex_SyncIsIdempotent(t *testing.T) {

	database := newResponseTestDatabase(t)
	ctx := context.Background()
	collection := database.Collection("OutboxItem")

	require.NoError(t, OutboxItem(ctx, database))
	before := indexCreatedAt(ctx, t, collection, "idx_OutboxItem_URL")

	require.NoError(t, OutboxItem(ctx, database))
	after := indexCreatedAt(ctx, t, collection, "idx_OutboxItem_URL")

	require.Equal(t, before, after, "Sync rebuilt an unchanged index; the partial filter does not round-trip")

	// The declared filter must be the one MongoDB stored
	stored := loadIndex(ctx, t, collection, "idx_OutboxItem_URL")
	require.Equal(t, bson.M{"url": bson.M{"$gt": ""}}, stored["partialFilterExpression"])
	require.Equal(t, true, stored["unique"])
}

// TestOutboxItemIndex_LookupByURLUsesTheIndex verifies that an equality match on url can use the partial index
func TestOutboxItemIndex_LookupByURLUsesTheIndex(t *testing.T) {

	database := newResponseTestDatabase(t)
	ctx := context.Background()

	require.NoError(t, OutboxItem(ctx, database))
	require.NoError(t, insertOutboxItem(ctx, database, "https://example.test/@1/pub/outbox/1"))

	// A partial index serves only queries that imply its filter, so prove that url == X implies url > ""
	var explain bson.M
	err := database.RunCommand(ctx, bson.D{
		{Key: "explain", Value: bson.D{
			{Key: "find", Value: "OutboxItem"},
			{Key: "filter", Value: bson.M{"url": "https://example.test/@1/pub/outbox/1"}},
		}},
		{Key: "verbosity", Value: "queryPlanner"},
	}).Decode(&explain)
	require.NoError(t, err)

	planner, ok := explain["queryPlanner"].(bson.M)
	require.True(t, ok)
	require.Contains(t, findIndexNames(planner["winningPlan"]), "idx_OutboxItem_URL")
}

// insertRetiredOutboxItem writes one raw row into the retired Outbox2 collection, and returns its _id
func insertRetiredOutboxItem(ctx context.Context, t *testing.T, database *mongo.Database, url string) primitive.ObjectID {

	t.Helper()

	rowID := primitive.NewObjectID()

	_, err := database.Collection("Outbox2").InsertOne(ctx, bson.M{
		"_id":         rowID,
		"actorType":   "User",
		"actorId":     primitive.NewObjectID(),
		"url":         url,
		"permissions": bson.A{primitive.NilObjectID},
		"activity":    bson.M{"type": "Create", "id": url},
		"createDate":  int64(1700000000000),
	})

	require.NoError(t, err)
	return rowID
}

// loadRawRow returns one row of a collection by _id, or nil when it is not there
func loadRawRow(ctx context.Context, t *testing.T, collection *mongo.Collection, rowID primitive.ObjectID) bson.M {

	t.Helper()

	result := bson.M{}
	err := collection.FindOne(ctx, bson.M{"_id": rowID}).Decode(&result)

	if err == mongo.ErrNoDocuments {
		return nil
	}

	require.NoError(t, err)
	return result
}

// countRows returns the number of rows in a collection
func countRows(ctx context.Context, t *testing.T, collection *mongo.Collection) int64 {

	t.Helper()

	count, err := collection.CountDocuments(ctx, bson.M{})
	require.NoError(t, err)
	return count
}

// TestOutboxItem_MovesRetiredRows verifies that every row in Outbox2 moves to OutboxItem with its _id and fields unchanged
func TestOutboxItem_MovesRetiredRows(t *testing.T) {

	database := newResponseTestDatabase(t)
	ctx := context.Background()

	firstID := insertRetiredOutboxItem(ctx, t, database, "https://example.test/@1/pub/outbox/1")
	secondID := insertRetiredOutboxItem(ctx, t, database, "https://example.test/@1/pub/outbox/2")
	retired := loadRawRow(ctx, t, database.Collection("Outbox2"), firstID)

	require.NoError(t, OutboxItem(ctx, database))

	require.Equal(t, retired, loadRawRow(ctx, t, database.Collection("OutboxItem"), firstID), "the moved row must keep every field")
	require.NotNil(t, loadRawRow(ctx, t, database.Collection("OutboxItem"), secondID))
	require.Zero(t, countRows(ctx, t, database.Collection("Outbox2")), "every moved row leaves Outbox2")
	require.Equal(t, true, loadIndex(ctx, t, database.Collection("OutboxItem"), "idx_OutboxItem_URL")["unique"])
}

// TestOutboxItem_KeepsCurrentRows verifies that a row already in OutboxItem is never overwritten by its retired copy
func TestOutboxItem_KeepsCurrentRows(t *testing.T) {

	database := newResponseTestDatabase(t)
	ctx := context.Background()

	rowID := insertRetiredOutboxItem(ctx, t, database, "https://example.test/@1/pub/outbox/old")

	_, err := database.Collection("OutboxItem").InsertOne(ctx, bson.M{"_id": rowID, "url": "https://example.test/@1/pub/outbox/new"})
	require.NoError(t, err)

	require.NoError(t, OutboxItem(ctx, database))

	current := loadRawRow(ctx, t, database.Collection("OutboxItem"), rowID)
	require.Equal(t, "https://example.test/@1/pub/outbox/new", current["url"])
	require.NotContains(t, current, "activity", "no field may be copied onto a current row")
	require.Zero(t, countRows(ctx, t, database.Collection("Outbox2")))
}

// TestOutboxItem_MoveIsIdempotent verifies that a second start moves nothing and changes nothing
func TestOutboxItem_MoveIsIdempotent(t *testing.T) {

	database := newResponseTestDatabase(t)
	ctx := context.Background()

	rowID := insertRetiredOutboxItem(ctx, t, database, "https://example.test/@1/pub/outbox/1")

	require.NoError(t, OutboxItem(ctx, database))
	first := loadRawRow(ctx, t, database.Collection("OutboxItem"), rowID)

	require.NoError(t, OutboxItem(ctx, database))
	require.Equal(t, first, loadRawRow(ctx, t, database.Collection("OutboxItem"), rowID))
	require.Equal(t, int64(1), countRows(ctx, t, database.Collection("OutboxItem")))
}

// TestOutboxItem_LateRetiredRowMovesOnNextStart verifies that a row an older server writes after the move is moved by the next start
func TestOutboxItem_LateRetiredRowMovesOnNextStart(t *testing.T) {

	database := newResponseTestDatabase(t)
	ctx := context.Background()

	insertRetiredOutboxItem(ctx, t, database, "https://example.test/@1/pub/outbox/1")
	require.NoError(t, OutboxItem(ctx, database))

	lateID := insertRetiredOutboxItem(ctx, t, database, "https://example.test/@1/pub/outbox/2")
	require.NoError(t, OutboxItem(ctx, database))

	require.NotNil(t, loadRawRow(ctx, t, database.Collection("OutboxItem"), lateID))
	require.Zero(t, countRows(ctx, t, database.Collection("Outbox2")))
}

// TestOutboxItem_ConflictingRowStaysRetired verifies that a retired row whose url is taken stays in Outbox2, and the others still move
func TestOutboxItem_ConflictingRowStaysRetired(t *testing.T) {

	database := newResponseTestDatabase(t)
	ctx := context.Background()

	// Build the unique index, then hold its url with a current row
	require.NoError(t, OutboxItem(ctx, database))
	require.NoError(t, insertOutboxItem(ctx, database, "https://example.test/@1/pub/outbox/taken"))

	conflictID := insertRetiredOutboxItem(ctx, t, database, "https://example.test/@1/pub/outbox/taken")
	movableID := insertRetiredOutboxItem(ctx, t, database, "https://example.test/@1/pub/outbox/free")

	require.NoError(t, OutboxItem(ctx, database), "a row that cannot move is reported, not returned")

	require.NotNil(t, loadRawRow(ctx, t, database.Collection("Outbox2"), conflictID), "the conflicting row must not be deleted")
	require.Nil(t, loadRawRow(ctx, t, database.Collection("OutboxItem"), conflictID))
	require.NotNil(t, loadRawRow(ctx, t, database.Collection("OutboxItem"), movableID))
}

// TestOutboxItem_NoRetiredCollection verifies that a Domain without Outbox2 is left without one
func TestOutboxItem_NoRetiredCollection(t *testing.T) {

	database := newResponseTestDatabase(t)
	ctx := context.Background()

	require.NoError(t, OutboxItem(ctx, database))

	names, err := database.ListCollectionNames(ctx, bson.M{"name": "Outbox2"})
	require.NoError(t, err)
	require.Empty(t, names)
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
