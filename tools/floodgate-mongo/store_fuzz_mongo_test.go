package floodgatemongo

import (
	"context"
	"testing"

	"github.com/benpate/floodgate"
	"github.com/benpate/floodgate/storetest"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
)

// FuzzStore_MatchesMemoryStore runs one script of Store operations against MongoDB and against
// Floodgate's in-memory reference, and verifies that both end up holding the same records
func FuzzStore_MatchesMemoryStore(f *testing.F) {

	// Every operation, on a host with no record. Only some of them create one.
	f.Add([]byte{0, 0, 0})
	f.Add([]byte{1, 0, 0})
	f.Add([]byte{2, 0, 5})
	f.Add([]byte{3, 0, 5})
	f.Add([]byte{4, 0, 5})
	f.Add([]byte{5, 0, 200})
	f.Add([]byte{6, 0, 1})
	f.Add([]byte{7, 0, 3})
	f.Add([]byte{8, 0, 3})

	// A failure, a cooldown, and a recovery
	f.Add([]byte{0, 0, 0, 2, 0, 5, 1, 0, 0})

	// Failures on separate days, a dormant mark, a second mark that must not count, and failed checks
	f.Add([]byte{0, 1, 0, 9, 0, 24, 0, 1, 0, 3, 1, 3, 9, 0, 1, 3, 1, 9, 4, 1, 200, 4, 1, 2, 9, 0, 255})

	// A long rate limit followed by a shorter one, which must not shorten it
	f.Add([]byte{5, 2, 250, 5, 2, 130, 5, 2, 3})

	// A paused host that is resumed, beside one that stays paused
	f.Add([]byte{6, 0, 1, 6, 1, 1, 6, 0, 0, 7, 0, 7, 8, 1, 2})

	// A recovery after everything has gone wrong, then trouble again
	f.Add([]byte{0, 0, 0, 2, 0, 9, 3, 0, 1, 4, 0, 1, 5, 0, 255, 6, 0, 1, 1, 0, 0, 0, 0, 0})

	store, database := newTestStore(f)

	f.Fuzz(func(t *testing.T, script []byte) {

		ctx := context.Background()

		// Every script starts from an empty collection
		_, err := database.Collection(CollectionName).DeleteMany(ctx, bson.M{})
		require.NoError(t, err)

		expected, err := storetest.Run(ctx, floodgate.NewMemoryStore(), script)
		require.NoError(t, err)

		actual, err := storetest.Run(ctx, store, script)
		require.NoError(t, err)

		require.Equal(t, expected, actual)
	})
}
