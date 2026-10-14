package queries

import (
	"context"
	"testing"

	"github.com/benpate/data"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// insertTestStream writes one raw Stream document beneath a parent, bypassing the service layer
func insertTestStream(t *testing.T, session data.Session, parentID primitive.ObjectID, rank int, deleteDate int64) {

	t.Helper()

	_, err := mongoCollection(session.Collection("Stream")).InsertOne(context.Background(), bson.M{
		"_id":        primitive.NewObjectID(),
		"parentId":   parentID,
		"rank":       rank,
		"deleteDate": deleteDate,
	})

	require.NoError(t, err)
}

// TestMaxRank requires the next rank beneath a parent to start at 1, and to follow the
// highest rank of its living children
func TestMaxRank(t *testing.T) {

	session := newFollowersTestSession(t)
	collection := session.Collection("Stream")
	ctx := context.Background()
	parentID := primitive.NewObjectID()

	// FUNKWHALE task 1.3: rank 0 means "not set", and the sort step numbers children from 1,
	// so the first child of an empty parent must not be given 0
	rank, err := MaxRank(ctx, collection, parentID)
	require.NoError(t, err)
	require.Equal(t, 1, rank, "an empty parent")

	insertTestStream(t, session, parentID, 3, 0)
	insertTestStream(t, session, parentID, 9, 1234567890)
	insertTestStream(t, session, primitive.NewObjectID(), 20, 0)

	rank, err = MaxRank(ctx, collection, parentID)
	require.NoError(t, err)
	require.Equal(t, 4, rank, "one after the highest living child of this parent")
}
