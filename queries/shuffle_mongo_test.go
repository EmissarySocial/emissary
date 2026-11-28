package queries

import (
	"context"
	"math"
	"testing"

	mockdb "github.com/benpate/data-mock"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestShuffle verifies the pipeline update against a real MongoDB, because $rand and $toLong
// are evaluated by the database
func TestShuffle(t *testing.T) {

	session := newFollowersTestSession(t)
	collection := mongoCollection(session.Collection("SearchResult"))
	require.NotNil(t, collection)

	// Seed the collection the way the old shuffle left it: numbered 1 to N
	const count = 200
	documents := make([]any, count)

	for index := range documents {
		documents[index] = bson.M{"_id": primitive.NewObjectID(), "name": "Record", "shuffle": int64(index + 1)}
	}

	_, err := collection.InsertMany(context.Background(), documents)
	require.NoError(t, err)

	// stored returns every record's shuffle value, keyed by its ID
	stored := func() map[primitive.ObjectID]int64 {

		cursor, err := collection.Find(context.Background(), bson.M{})
		require.NoError(t, err)

		var records []struct {
			ID      primitive.ObjectID `bson:"_id"`
			Name    string             `bson:"name"`
			Shuffle int64              `bson:"shuffle"`
		}

		require.NoError(t, cursor.All(context.Background(), &records))
		require.Len(t, records, count)

		result := make(map[primitive.ObjectID]int64, count)

		for _, record := range records {
			require.Equal(t, "Record", record.Name, "no other field is written")
			result[record.ID] = record.Shuffle
		}

		return result
	}

	require.NoError(t, Shuffle(context.Background(), session.Collection("SearchResult")))
	first := stored()

	distinct := make(map[int64]struct{}, count)

	for id, value := range first {
		require.GreaterOrEqual(t, value, int64(0), "record %s", id.Hex())
		require.Less(t, value, shuffleRange, "record %s", id.Hex())
		distinct[value] = struct{}{}
	}

	// Each record draws its own value.  Two equal draws out of 200 is a 1 in 10^11 event.
	require.Len(t, distinct, count)

	// A second run moves the records again
	require.NoError(t, Shuffle(context.Background(), session.Collection("SearchResult")))
	second := stored()

	moved := 0

	for id, value := range second {
		if value != first[id] {
			moved++
		}
	}

	require.Equal(t, count, moved, "every record gets a new value")
}

// TestShuffle_EmptyCollection confirms that a collection with no records is not an error
func TestShuffle_EmptyCollection(t *testing.T) {

	session := newFollowersTestSession(t)

	require.NoError(t, Shuffle(context.Background(), session.Collection("SearchResult")))
}

// TestShuffle_CanceledContext confirms that a failed update is returned, not swallowed
func TestShuffle_CanceledContext(t *testing.T) {

	session := newFollowersTestSession(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	require.Error(t, Shuffle(ctx, session.Collection("SearchResult")))
}

// TestShuffle_NotMongo confirms that a collection from another database driver is refused
func TestShuffle_NotMongo(t *testing.T) {

	session, err := mockdb.New().Session(context.Background())
	require.NoError(t, err)

	require.Error(t, Shuffle(context.Background(), session.Collection("SearchResult")))
}

// TestShuffleRange_NewRecordsSortLast pins the gap that keeps a new record after every shuffled
// one.  The service layer starts a record at MaxInt64 minus at most a trillion.
func TestShuffleRange_NewRecordsSortLast(t *testing.T) {

	const lowestNewValue = math.MaxInt64 - 999_999_999_999

	require.Less(t, shuffleRange, int64(lowestNewValue))
}
