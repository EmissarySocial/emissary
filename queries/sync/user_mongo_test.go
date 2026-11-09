package sync

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// These tests drive the primary-user index against a REAL MongoDB, because what they guard is
// Mongo's own behavior: whether the unique index refuses a second primary user, and what its partial filter excludes.

// TestUserPrimaryIndex_OneLivePrimary confirms that a domain holds one live primary user, that any number
// of other Users are unaffected, and that a deleted primary user releases the flag.
func TestUserPrimaryIndex_OneLivePrimary(t *testing.T) {

	database := newResponseTestDatabase(t)
	collection := database.Collection("User")

	require.NoError(t, User(context.Background(), database))

	// One primary user, and any number of Users who are not
	_, err := insertUser(t, collection, true, 0)
	require.NoError(t, err)

	for range 3 {
		_, err := insertUser(t, collection, false, 0)
		require.NoError(t, err)
	}

	// RULE: A second live primary user is refused
	_, err = insertUser(t, collection, true, 0)
	require.True(t, mongo.IsDuplicateKeyError(err), "a second live primary user must be refused, not %v", err)

	// A deleted User keeps its flag without blocking anyone
	_, err = insertUser(t, collection, true, 999)
	require.NoError(t, err)
}

// TestUserPrimaryIndex_ClearBeforeSet confirms the order User.Save relies on: the old primary user is
// cleared before the new one is set, and the reverse order is refused.
func TestUserPrimaryIndex_ClearBeforeSet(t *testing.T) {

	database := newResponseTestDatabase(t)
	collection := database.Collection("User")

	require.NoError(t, User(context.Background(), database))

	oldPrimary, err := insertUser(t, collection, true, 0)
	require.NoError(t, err)

	newPrimary, err := insertUser(t, collection, false, 0)
	require.NoError(t, err)

	// Setting the new one first is refused while the old one holds the flag
	err = setPrimary(collection, newPrimary, true)
	require.True(t, mongo.IsDuplicateKeyError(err), "setting a second primary user must be refused, not %v", err)

	// Clearing the old one first lets the new one take it
	require.NoError(t, setPrimary(collection, oldPrimary, false))
	require.NoError(t, setPrimary(collection, newPrimary, true))

	count, err := collection.CountDocuments(context.Background(), bson.M{"isPrimary": true, "deleteDate": 0})
	require.NoError(t, err)
	require.Equal(t, int64(1), count)
}

// TestUserPrimaryIndex_TwoWritersRace races two writers that share nothing but the database, each
// making a different User primary, and confirms that exactly one of them wins every round.
func TestUserPrimaryIndex_TwoWritersRace(t *testing.T) {

	database := newResponseTestDatabase(t)
	ctx := context.Background()

	require.NoError(t, User(ctx, database))

	// The second writer gets its own client, as a second server would
	connectCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	otherClient, err := mongo.Connect(connectCtx, options.Client().ApplyURI(responseTestConnection))
	require.NoError(t, err)
	t.Cleanup(func() { _ = otherClient.Disconnect(context.Background()) })

	writers := []*mongo.Collection{
		database.Collection("User"),
		otherClient.Database(database.Name()).Collection("User"),
	}

	for range 20 {

		// Two Users, neither of them primary
		_, err := writers[0].UpdateMany(ctx, bson.M{}, bson.M{"$set": bson.M{"isPrimary": false}})
		require.NoError(t, err)

		candidates := make([]primitive.ObjectID, len(writers))
		for index := range writers {
			candidates[index], err = insertUser(t, writers[0], false, 0)
			require.NoError(t, err)
		}

		// Each writer makes its own candidate primary at the same moment
		results := make(chan error, len(writers))

		for index, writer := range writers {
			go func() {
				results <- setPrimary(writer, candidates[index], true)
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

		require.Equal(t, 1, successes, "exactly one writer must make its User primary")
	}
}

// insertUser writes one raw User document, with the unique fields every User index needs
func insertUser(t *testing.T, collection *mongo.Collection, isPrimary bool, deleteDate int64) (primitive.ObjectID, error) {

	t.Helper()

	userID := primitive.NewObjectID()

	_, err := collection.InsertOne(context.Background(), bson.M{
		"_id":          userID,
		"username":     userID.Hex(),
		"profileUrl":   "https://example.com/@" + userID.Hex(),
		"emailAddress": userID.Hex() + "@example.com",
		"isPrimary":    isPrimary,
		"deleteDate":   deleteDate,
	})

	return userID, err
}

// setPrimary writes one User's primary flag, as a save of that User would
func setPrimary(collection *mongo.Collection, userID primitive.ObjectID, isPrimary bool) error {
	_, err := collection.UpdateOne(context.Background(), bson.M{"_id": userID}, bson.M{"$set": bson.M{"isPrimary": isPrimary}})
	return err
}
