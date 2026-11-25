package upgrades

import (
	"context"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// insertSyndication stores a record whose `syndication` property holds the given value, and
// returns its ID. A nil value leaves the property out.
func insertSyndication(t *testing.T, collection *mongo.Collection, syndication any) primitive.ObjectID {

	t.Helper()

	recordID := primitive.NewObjectID()
	record := bson.M{"_id": recordID, "label": "kept"}

	if syndication != nil {
		record["syndication"] = syndication
	}

	_, err := collection.InsertOne(context.Background(), record)
	require.NoError(t, err)

	return recordID
}

// loadRecord reads one record back as a plain map
func loadRecord(t *testing.T, collection *mongo.Collection, recordID primitive.ObjectID) bson.M {

	t.Helper()

	result := bson.M{}
	require.NoError(t, collection.FindOne(context.Background(), bson.M{"_id": recordID}).Decode(&result))

	return result
}

// TestVersion40_FlattensTheLegacyDocument verifies each stored shape of `syndication`, in
// both of the collections that hold a Stream
func TestVersion40_FlattensTheLegacyDocument(t *testing.T) {

	for _, name := range []string{"Stream", "StreamDraft"} {
		t.Run(name, func(t *testing.T) {

			database := newUpgradeTestDatabase(t)
			collection := database.Collection(name)

			// The legacy shapes that rosetta's delta.Slice stored
			legacy := insertSyndication(t, collection, bson.M{"values": bson.A{"mastodon", "bluesky"}})
			legacyOne := insertSyndication(t, collection, bson.M{"values": bson.A{"mastodon"}})
			legacyEmpty := insertSyndication(t, collection, bson.M{"values": bson.A{}})
			legacyNull := insertSyndication(t, collection, bson.M{"values": nil})
			legacyBare := insertSyndication(t, collection, bson.M{})

			// The shapes that must be left exactly as they are
			array := insertSyndication(t, collection, bson.A{"threads"})
			emptyArray := insertSyndication(t, collection, bson.A{})
			missing := insertSyndication(t, collection, nil)

			require.NoError(t, Version40(context.Background(), database))

			require.Equal(t, bson.A{"mastodon", "bluesky"}, loadRecord(t, collection, legacy)["syndication"])
			require.Equal(t, bson.A{"mastodon"}, loadRecord(t, collection, legacyOne)["syndication"])
			require.Equal(t, bson.A{}, loadRecord(t, collection, legacyEmpty)["syndication"])
			require.Equal(t, bson.A{}, loadRecord(t, collection, legacyNull)["syndication"])
			require.Equal(t, bson.A{}, loadRecord(t, collection, legacyBare)["syndication"])

			require.Equal(t, bson.A{"threads"}, loadRecord(t, collection, array)["syndication"])
			require.Equal(t, bson.A{}, loadRecord(t, collection, emptyArray)["syndication"])

			untouched := loadRecord(t, collection, missing)
			require.NotContains(t, untouched, "syndication")

			// Nothing else on a record is touched, and no record is added or lost
			documents := allDocuments(t, collection)
			require.Len(t, documents, 8)

			for _, document := range documents {
				require.Equal(t, "kept", document["label"])
			}

			// No record in the old shape is left
			remaining, err := collection.CountDocuments(context.Background(), bson.M{"syndication": bson.M{"$type": "object"}})
			require.NoError(t, err)
			require.Zero(t, remaining)
		})
	}
}

// TestVersion40_SecondRunChangesNothing verifies that the upgrade is safe to run again, as
// it is when two servers start together or a database's recorded version is wrong
func TestVersion40_SecondRunChangesNothing(t *testing.T) {

	database := newUpgradeTestDatabase(t)
	collection := database.Collection("Stream")
	ctx := context.Background()

	insertSyndication(t, collection, bson.M{"values": bson.A{"mastodon", "bluesky"}})
	insertSyndication(t, collection, bson.M{"values": nil})
	insertSyndication(t, collection, bson.A{"threads"})
	insertSyndication(t, collection, nil)

	// The first run converts the two legacy records
	modified, err := flattenSyndication(ctx, collection)
	require.NoError(t, err)
	require.Equal(t, int64(2), modified)

	first := allDocuments(t, collection)

	// The second run finds nothing to do
	modified, err = flattenSyndication(ctx, collection)
	require.NoError(t, err)
	require.Zero(t, modified)

	require.NoError(t, Version40(ctx, database))
	require.Equal(t, first, allDocuments(t, collection))
}

// TestVersion40_ArrayOfStringsIsNotMatched verifies that a converted list is never matched again
func TestVersion40_ArrayOfStringsIsNotMatched(t *testing.T) {

	// RULE: MongoDB's $type also matches an array that HOLDS the type, so the filter is
	// only safe because a syndication list holds strings and never documents.
	database := newUpgradeTestDatabase(t)
	collection := database.Collection("Stream")

	recordID := insertSyndication(t, collection, bson.A{"mastodon", "bluesky"})

	modified, err := flattenSyndication(context.Background(), collection)
	require.NoError(t, err)
	require.Zero(t, modified)
	require.Equal(t, bson.A{"mastodon", "bluesky"}, loadRecord(t, collection, recordID)["syndication"])
}

// TestVersion40_ConvertedStreamLoads verifies that a real Stream stored in the legacy shape
// loads with the same targets before and after the upgrade, and reads as unchanged
func TestVersion40_ConvertedStreamLoads(t *testing.T) {

	database := newUpgradeTestDatabase(t)
	collection := database.Collection("Stream")
	ctx := context.Background()

	// Store a whole Stream, then put its targets back into the legacy shape
	stream := model.NewStream()
	stream.Label = "Album"

	_, err := collection.InsertOne(ctx, stream)
	require.NoError(t, err)

	legacy := bson.M{"$set": bson.M{"syndication": bson.M{"values": bson.A{"mastodon", "bluesky"}}}}
	_, err = collection.UpdateOne(ctx, bson.M{"_id": stream.StreamID}, legacy)
	require.NoError(t, err)

	load := func() model.Stream {
		result := model.NewStream()
		require.NoError(t, collection.FindOne(ctx, bson.M{"_id": stream.StreamID}).Decode(&result))
		return result
	}

	before := load()
	require.Equal(t, []string{"mastodon", "bluesky"}, before.Syndication.Value())

	require.NoError(t, Version40(ctx, database))

	after := load()
	require.Equal(t, []string{"mastodon", "bluesky"}, after.Syndication.Value())
	require.False(t, after.Syndication.IsChanged())
	require.Equal(t, "Album", after.Label)
	require.Equal(t, before.Token.Value(), after.Token.Value())

	// A query on the plain property now finds the Stream, as the search templates need
	found, err := collection.CountDocuments(ctx, bson.M{"syndication": bson.M{"$in": bson.A{"bluesky", "ALL-PARTNERS"}}})
	require.NoError(t, err)
	require.Equal(t, int64(1), found)
}

// TestVersion40_EmptyDatabase verifies that a database with neither collection is not an error
func TestVersion40_EmptyDatabase(t *testing.T) {
	require.NoError(t, Version40(context.Background(), newUpgradeTestDatabase(t)))
}

// TestVersion40_DatabaseFailure verifies that a failed update is returned, so the slot runs again
func TestVersion40_DatabaseFailure(t *testing.T) {

	database := newUpgradeTestDatabase(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	require.Error(t, Version40(ctx, database))

	_, err := flattenSyndication(ctx, database.Collection("Stream"))
	require.Error(t, err)
}
