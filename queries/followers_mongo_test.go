package queries

import (
	"context"
	"testing"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	mongodb "github.com/benpate/data-mongo"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// These tests cover SetFollowersCount, which BUG-10 promotes from a local UI convenience to a
// number published on the wire as `totalItems` on the ActivityPub followers collection. Two
// properties matter now that remote servers read it:
//
//   - It counts ONLY this User's followers. `parentId` is shared with Stream and Search followers,
//     so the query needs the `type` discriminator to be correct rather than merely lucky (the same
//     shape as BUG-33).
//   - It counts EVERY follower method. Email subscribers are included deliberately: the published
//     number answers "how many people receive this actor's posts," not "how many ActivityPub
//     records exist."
//
// They run against a REAL MongoDB because SetFollowersCount ends in queries.RawUpdate, which
// unwraps the collection to a mongo driver handle and refuses anything else -- a fake can observe
// the criteria but never the read-modify-write that is the actual subject here. They skip when no
// database is reachable, so `go test ./...` still passes on a machine without one.

// followersTestConnection is the local MongoDB that these tests require
const followersTestConnection = "mongodb://localhost:27017/?directConnection=true"

// newFollowersTestSession connects to a local MongoDB and returns a throwaway session that is
// dropped when the test ends. The test is skipped when no database is reachable.
func newFollowersTestSession(t *testing.T) data.Session {

	t.Helper()

	if testing.Short() {
		t.Skip("Skipping MongoDB integration test in -short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(followersTestConnection))

	if err != nil {
		t.Skip("Skipping: no MongoDB at " + followersTestConnection)
	}

	if err := client.Ping(ctx, nil); err != nil {
		t.Skip("Skipping: no MongoDB at " + followersTestConnection)
	}

	// A per-test database name keeps parallel runs (and the real demo data) well clear of this.
	database := client.Database("emissary_followerstest_" + primitive.NewObjectID().Hex())

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = database.Drop(ctx)
		_ = client.Disconnect(ctx)
	})

	return mongodb.NewSession(database)
}

// insertTestUser writes one User document with a known starting followerCount.
func insertTestUser(t *testing.T, session data.Session, userID primitive.ObjectID, followerCount int) {

	t.Helper()

	_, err := mongoCollection(session.Collection("User")).InsertOne(context.Background(), bson.M{
		"_id":           userID,
		"followerCount": followerCount,
		"deleteDate":    0,
	})

	require.Nil(t, err)
}

// insertTestFollower writes one raw Follower document, bypassing the service layer.
func insertTestFollower(t *testing.T, session data.Session, parentType string, parentID primitive.ObjectID, method string, deleteDate int64) {

	t.Helper()

	_, err := mongoCollection(session.Collection("Follower")).InsertOne(context.Background(), bson.M{
		"_id":        primitive.NewObjectID(),
		"type":       parentType,
		"parentId":   parentID,
		"method":     method,
		"actor":      bson.M{"profileUrl": "https://example.test/@follower"},
		"deleteDate": deleteDate,
	})

	require.Nil(t, err)
}

// readFollowerCount returns the denormalized count now stored on the User record.
func readFollowerCount(t *testing.T, session data.Session, userID primitive.ObjectID) int {

	t.Helper()

	result := struct {
		FollowerCount int `bson:"followerCount"`
	}{}

	err := mongoCollection(session.Collection("User")).
		FindOne(context.Background(), bson.M{"_id": userID}).
		Decode(&result)

	require.Nil(t, err)
	return result.FollowerCount
}

// recalc runs the function under test against the session's User and Follower collections.
func recalc(t *testing.T, session data.Session, userID primitive.ObjectID) {

	t.Helper()

	require.Nil(t, SetFollowersCount(
		session.Collection("User"),
		session.Collection("Follower"),
		userID,
	))
}

// TestSetFollowersCount_CountsEveryMethod confirms the deliberate inclusiveness decision: an
// email subscriber counts exactly as much as an ActivityPub follower.
func TestSetFollowersCount_CountsEveryMethod(t *testing.T) {

	session := newFollowersTestSession(t)
	userID := primitive.NewObjectID()
	insertTestUser(t, session, userID, 0)

	insertTestFollower(t, session, model.FollowerTypeUser, userID, model.FollowerMethodActivityPub, 0)
	insertTestFollower(t, session, model.FollowerTypeUser, userID, model.FollowerMethodActivityPub, 0)
	insertTestFollower(t, session, model.FollowerTypeUser, userID, model.FollowerMethodEmail, 0)

	recalc(t, session, userID)
	require.Equal(t, 3, readFollowerCount(t, session, userID))
}

// TestSetFollowersCount_IgnoresOtherParentTypes is the discriminator fix. Stream and Search
// followers share the `parentId` field, so a query without the `type` clause is correct only by
// the accident that ObjectIDs do not collide across collections. Here they deliberately DO
// collide, which is the only way to prove the clause is carrying its weight.
func TestSetFollowersCount_IgnoresOtherParentTypes(t *testing.T) {

	session := newFollowersTestSession(t)
	userID := primitive.NewObjectID()
	insertTestUser(t, session, userID, 0)

	insertTestFollower(t, session, model.FollowerTypeUser, userID, model.FollowerMethodActivityPub, 0)

	// Same ObjectID, different parent types -- these must not be counted
	insertTestFollower(t, session, model.FollowerTypeStream, userID, model.FollowerMethodActivityPub, 0)
	insertTestFollower(t, session, model.FollowerTypeSearch, userID, model.FollowerMethodActivityPub, 0)

	recalc(t, session, userID)
	require.Equal(t, 1, readFollowerCount(t, session, userID))
}

// TestSetFollowersCount_IgnoresDeleted confirms soft-deleted rows are excluded.
func TestSetFollowersCount_IgnoresDeleted(t *testing.T) {

	session := newFollowersTestSession(t)
	userID := primitive.NewObjectID()
	insertTestUser(t, session, userID, 0)

	insertTestFollower(t, session, model.FollowerTypeUser, userID, model.FollowerMethodActivityPub, 0)
	insertTestFollower(t, session, model.FollowerTypeUser, userID, model.FollowerMethodActivityPub, 1234567890)

	recalc(t, session, userID)
	require.Equal(t, 1, readFollowerCount(t, session, userID))
}

// TestSetFollowersCount_Decrements is the BUG-10 regression. Follower.Delete now calls
// CalcFollowerCount, so the published `totalItems` must be able to go DOWN. Before that fix the
// count was monotonic, and an actor's follower number would only ever climb.
func TestSetFollowersCount_Decrements(t *testing.T) {

	session := newFollowersTestSession(t)
	userID := primitive.NewObjectID()
	insertTestUser(t, session, userID, 0)

	insertTestFollower(t, session, model.FollowerTypeUser, userID, model.FollowerMethodActivityPub, 0)
	insertTestFollower(t, session, model.FollowerTypeUser, userID, model.FollowerMethodActivityPub, 0)

	recalc(t, session, userID)
	require.Equal(t, 2, readFollowerCount(t, session, userID))

	// Soft-delete one follower, exactly as Follower.Delete does, then recalculate
	_, err := mongoCollection(session.Collection("Follower")).UpdateOne(
		context.Background(),
		bson.M{"type": model.FollowerTypeUser, "parentId": userID, "deleteDate": 0},
		bson.M{"$set": bson.M{"deleteDate": int64(1234567890)}},
	)
	require.Nil(t, err)

	recalc(t, session, userID)
	require.Equal(t, 1, readFollowerCount(t, session, userID))
}

// TestSetFollowersCount_ZeroIsWritten confirms the count is written back as 0 rather than left
// stale when the last follower goes away -- the case that decides whether a departing follower is
// visible to remote servers at all.
func TestSetFollowersCount_ZeroIsWritten(t *testing.T) {

	session := newFollowersTestSession(t)
	userID := primitive.NewObjectID()
	insertTestUser(t, session, userID, 7) // a stale, inflated starting value

	recalc(t, session, userID)
	require.Equal(t, 0, readFollowerCount(t, session, userID))
}
