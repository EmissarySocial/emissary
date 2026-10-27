package mastodon

import (
	"testing"

	"github.com/benpate/exp"
	"github.com/benpate/toot/txn"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestCursorValue covers numeric cursors, ObjectID cursors, and cursors that cannot be used
func TestCursorValue(t *testing.T) {

	known := primitive.NewObjectID()

	resolve := func(id primitive.ObjectID) (int64, bool) {
		if id == known {
			return 42, true
		}
		return 0, false
	}

	value, ok := cursorValue("1792993144292", resolve)
	require.True(t, ok)
	require.Equal(t, int64(1792993144292), value, "a number is used as-is")

	value, ok = cursorValue(known.Hex(), resolve)
	require.True(t, ok)
	require.Equal(t, int64(42), value, "an ObjectID is replaced by its record's value")

	_, ok = cursorValue(primitive.NewObjectID().Hex(), resolve)
	require.False(t, ok, "an ObjectID with no record is not used")

	_, ok = cursorValue(known.Hex(), nil)
	require.False(t, ok, "an ObjectID is not used without a resolver")

	_, ok = cursorValue("", resolve)
	require.False(t, ok)

	_, ok = cursorValue("not-a-cursor", resolve)
	require.False(t, ok)
}

// TestQueryExpressionWithCursors confirms an ObjectID cursor pages on the record's own field
func TestQueryExpressionWithCursors(t *testing.T) {

	newest := primitive.NewObjectID()

	resolve := func(id primitive.ObjectID) (int64, bool) {
		return 1000, id == newest
	}

	result := queryExpressionWithCursors(txn.GetTimeline_Home{MinID: newest.Hex()}, "rank", resolve)
	require.Equal(t, exp.All().AndGreaterThan("rank", int64(1000)), result)

	result = queryExpressionWithCursors(txn.GetTimeline_Home{MaxID: newest.Hex(), SinceID: "5"}, "rank", resolve)
	require.Equal(t, exp.All().AndLessThan("rank", int64(1000)).AndGreaterThan("rank", int64(5)), result)

	// Without a resolver, the old numeric-only behavior is unchanged
	result = queryExpressionByField(txn.GetTimeline_Home{MinID: newest.Hex()}, "rank")
	require.Equal(t, exp.All(), result)
}
