package queries

import (
	"context"
	"math"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestClaimNotificationEmail verifies which stored watermarks a claim can move, against a real
// MongoDB, because the conditional update is the whole subject
func TestClaimNotificationEmail(t *testing.T) {

	session := newFollowersTestSession(t)
	collection := mongoCollection(session.Collection("User"))
	require.NotNil(t, collection)

	insert := func(fields bson.M) primitive.ObjectID {
		userID := primitive.NewObjectID()
		fields["_id"] = userID
		_, err := collection.InsertOne(context.Background(), fields)
		require.NoError(t, err)
		return userID
	}

	watermark := func(userID primitive.ObjectID) any {
		stored := bson.M{}
		require.NoError(t, collection.FindOne(context.Background(), bson.M{"_id": userID}).Decode(&stored))
		return stored["notificationEmailDate"]
	}

	t.Run("the named watermark moves once", func(t *testing.T) {
		userID := insert(bson.M{"notificationEmailDate": int64(100), "deleteDate": 0, "displayName": "Sarah"})

		claimed, err := ClaimNotificationEmail(session, userID, 100, 200)
		require.NoError(t, err)
		require.True(t, claimed)
		require.Equal(t, int64(200), watermark(userID))

		claimed, err = ClaimNotificationEmail(session, userID, 100, 300)
		require.NoError(t, err)
		require.False(t, claimed, "a watermark that already moved is not claimed again")
		require.Equal(t, int64(200), watermark(userID))

		// Nothing else on the record changes
		stored := bson.M{}
		require.NoError(t, collection.FindOne(context.Background(), bson.M{"_id": userID}).Decode(&stored))
		require.Equal(t, "Sarah", stored["displayName"])
	})

	t.Run("a deleted User", func(t *testing.T) {
		userID := insert(bson.M{"notificationEmailDate": int64(100), "deleteDate": 1794200000000})

		claimed, err := ClaimNotificationEmail(session, userID, 100, 200)
		require.NoError(t, err)
		require.False(t, claimed)
	})

	t.Run("a User with no watermark field", func(t *testing.T) {
		userID := insert(bson.M{"deleteDate": 0})

		claimed, err := ClaimNotificationEmail(session, userID, 0, 200)
		require.NoError(t, err)
		require.False(t, claimed, "a record written by an older server is never emailed")
	})

	t.Run("a User who does not exist", func(t *testing.T) {
		claimed, err := ClaimNotificationEmail(session, primitive.NewObjectID(), 0, 200)
		require.NoError(t, err)
		require.False(t, claimed)
	})

	t.Run("many claimers, one winner", func(t *testing.T) {
		userID := insert(bson.M{"notificationEmailDate": int64(0), "deleteDate": 0})

		var winners atomic.Int32
		var group sync.WaitGroup

		for range 8 {
			group.Go(func() {
				claimed, err := ClaimNotificationEmail(session, userID, 0, 500)
				if (err == nil) && claimed {
					winners.Add(1)
				}
			})
		}

		group.Wait()
		require.Equal(t, int32(1), winners.Load())
	})
}

// TestNotificationUsersWithUnread verifies that only an unread, undeleted, old enough Notification
// names its User
func TestNotificationUsersWithUnread(t *testing.T) {

	session := newFollowersTestSession(t)
	collection := mongoCollection(session.Collection("Notification"))
	require.NotNil(t, collection)

	insert := func(userID primitive.ObjectID, createDate int64, readDate int64, deleteDate int64) {
		_, err := collection.InsertOne(context.Background(), bson.M{
			"_id":        primitive.NewObjectID(),
			"userId":     userID,
			"createDate": createDate,
			"readDate":   readDate,
			"deleteDate": deleteDate,
		})
		require.NoError(t, err)
	}

	unread := int64(math.MaxInt64)

	owed := primitive.NewObjectID()
	insert(owed, 100, unread, 0)
	insert(owed, 150, unread, 0) // A second row names the User once

	atTheCutoff := primitive.NewObjectID()
	insert(atTheCutoff, 200, unread, 0)

	tooNew := primitive.NewObjectID()
	insert(tooNew, 201, unread, 0)

	alreadyRead := primitive.NewObjectID()
	insert(alreadyRead, 100, 1234, 0)

	deleted := primitive.NewObjectID()
	insert(deleted, 100, unread, 999)

	result, err := NotificationUsersWithUnread(session, 200)
	require.NoError(t, err)
	require.ElementsMatch(t, []primitive.ObjectID{owed, atTheCutoff}, result)
}
