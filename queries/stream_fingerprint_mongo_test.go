package queries

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestClaimStreamFingerprint verifies which stored fingerprints a claim can move, against a real
// MongoDB, because the conditional update is the whole subject
func TestClaimStreamFingerprint(t *testing.T) {

	session := newFollowersTestSession(t)
	collection := mongoCollection(session.Collection("Stream"))
	require.NotNil(t, collection)

	insert := func(fields bson.M) primitive.ObjectID {
		streamID := primitive.NewObjectID()
		fields["_id"] = streamID
		_, err := collection.InsertOne(context.Background(), fields)
		require.NoError(t, err)
		return streamID
	}

	stored := func(streamID primitive.ObjectID) bson.M {
		result := bson.M{}
		require.NoError(t, collection.FindOne(context.Background(), bson.M{"_id": streamID}).Decode(&result))
		return result
	}

	t.Run("the named fingerprint moves once", func(t *testing.T) {
		streamID := insert(bson.M{"fingerprint": "abc", "label": "Judgment Day"})

		claimed, err := ClaimStreamFingerprint(session, streamID, "abc", "def")
		require.NoError(t, err)
		require.True(t, claimed)

		// A second sender that read the same value has lost
		claimed, err = ClaimStreamFingerprint(session, streamID, "abc", "ghi")
		require.NoError(t, err)
		require.False(t, claimed)

		require.Equal(t, "def", stored(streamID)["fingerprint"])
		require.Equal(t, "Judgment Day", stored(streamID)["label"], "no other field is written")
	})

	t.Run("an empty fingerprint matches a missing, null, or empty property", func(t *testing.T) {
		for name, fields := range map[string]bson.M{
			"missing": {},
			"null":    {"fingerprint": nil},
			"empty":   {"fingerprint": ""},
		} {
			streamID := insert(fields)

			claimed, err := ClaimStreamFingerprint(session, streamID, "", "abc")
			require.NoError(t, err, name)
			require.True(t, claimed, name)
			require.Equal(t, "abc", stored(streamID)["fingerprint"], name)
		}
	})

	t.Run("an empty fingerprint does not match a stored one", func(t *testing.T) {
		streamID := insert(bson.M{"fingerprint": "abc"})

		claimed, err := ClaimStreamFingerprint(session, streamID, "", "def")
		require.NoError(t, err)
		require.False(t, claimed)
		require.Equal(t, "abc", stored(streamID)["fingerprint"])
	})

	t.Run("claiming an empty fingerprint removes the property", func(t *testing.T) {
		streamID := insert(bson.M{"fingerprint": "abc"})

		claimed, err := ClaimStreamFingerprint(session, streamID, "abc", "")
		require.NoError(t, err)
		require.True(t, claimed)
		require.NotContains(t, stored(streamID), "fingerprint")
	})

	t.Run("a deleted Stream can be claimed", func(t *testing.T) {
		streamID := insert(bson.M{"fingerprint": "abc", "deleteDate": int64(1)})

		claimed, err := ClaimStreamFingerprint(session, streamID, "abc", "")
		require.NoError(t, err)
		require.True(t, claimed)
	})

	t.Run("a Stream that does not exist claims nothing", func(t *testing.T) {
		claimed, err := ClaimStreamFingerprint(session, primitive.NewObjectID(), "", "abc")
		require.NoError(t, err)
		require.False(t, claimed)
	})
}
