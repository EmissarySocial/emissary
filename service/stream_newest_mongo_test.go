package service

import (
	"context"
	"testing"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// These tests run NewestPublicPost against a real MongoDB, because what they guard is which
// stored posts the visibility criteria let through.

// insertNewestTestPost writes one raw post for the owner, in the outbox or not
func insertNewestTestPost(t *testing.T, collection *mongo.Collection, ownerID primitive.ObjectID, isOutbox bool, publishDate int64, allow any, deleteDate int64) {

	t.Helper()

	streamID := primitive.NewObjectID()

	// Each live post needs a token of its own, or the token index refuses it
	_, err := collection.InsertOne(context.Background(), bson.M{
		"_id":          streamID,
		"token":        streamID.Hex(),
		"parentId":     ownerID,
		"attributedTo": bson.M{"userId": ownerID},
		"isOutbox":     isOutbox,
		"publishDate":  publishDate,
		"defaultAllow": bson.A{allow},
		"deleteDate":   deleteDate,
	})

	require.NoError(t, err)
}

// TestStream_NewestPublicPost finds the newest post anyone may see, ignoring drafts, private, deleted, and others' posts
func TestStream_NewestPublicPost(t *testing.T) {

	database := newStreamTokenDatabase(t)
	session := newStreamTokenSession(t, database)
	collection := database.Collection("Stream")
	service := testStreamService()

	ownerID := primitive.NewObjectID()
	now := time.Now().Unix()

	// No posts yet
	newest, err := service.NewestPublicPost(session, ownerID)
	require.NoError(t, err)
	require.Zero(t, newest)

	// One public post, plus newer ones that must not count
	insertNewestTestPost(t, collection, ownerID, true, now-100, model.MagicGroupIDAnonymous, 0)
	insertNewestTestPost(t, collection, ownerID, true, now-50, ownerID, 0)                                    // Private
	insertNewestTestPost(t, collection, ownerID, false, now-20, model.MagicGroupIDAnonymous, 0)               // Draft, outside the outbox
	insertNewestTestPost(t, collection, ownerID, true, now-10, model.MagicGroupIDAnonymous, now)              // Deleted
	insertNewestTestPost(t, collection, primitive.NewObjectID(), true, now-5, model.MagicGroupIDAnonymous, 0) // Someone else

	newest, err = service.NewestPublicPost(session, ownerID)
	require.NoError(t, err)
	require.Equal(t, now-100, newest)
}
