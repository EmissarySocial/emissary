package queries

import (
	"context"
	"testing"

	mockdb "github.com/benpate/data-mock"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestRedeemOAuthCode verifies which stored grants a code can be redeemed from, against a real
// MongoDB, because the conditional update is the whole subject
func TestRedeemOAuthCode(t *testing.T) {

	session := newFollowersTestSession(t)
	collection := mongoCollection(session.Collection("OAuthUserToken"))
	require.NotNil(t, collection)

	// insert writes one grant document with the provided fields
	insert := func(fields bson.M) primitive.ObjectID {
		grantID := primitive.NewObjectID()
		fields["_id"] = grantID
		_, err := collection.InsertOne(context.Background(), fields)
		require.NoError(t, err)
		return grantID
	}

	// codeRedeemed reads the stored flag back
	codeRedeemed := func(grantID primitive.ObjectID) bool {
		stored := bson.M{}
		require.NoError(t, collection.FindOne(context.Background(), bson.M{"_id": grantID}).Decode(&stored))
		return stored["codeRedeemed"] == true
	}

	t.Run("an unused code is redeemed once", func(t *testing.T) {
		grantID := insert(bson.M{"codeRedeemed": false, "deleteDate": 0})

		redeemed, err := RedeemOAuthCode(session, grantID)
		require.NoError(t, err)
		require.True(t, redeemed)
		require.True(t, codeRedeemed(grantID))

		redeemed, err = RedeemOAuthCode(session, grantID)
		require.NoError(t, err)
		require.False(t, redeemed, "a spent code is not redeemed again")
	})

	t.Run("a grant stored before the flag existed", func(t *testing.T) {
		grantID := insert(bson.M{"deleteDate": 0})

		redeemed, err := RedeemOAuthCode(session, grantID)
		require.NoError(t, err)
		require.True(t, redeemed, "a missing flag reads as unused")
		require.True(t, codeRedeemed(grantID))
	})

	t.Run("a revoked grant", func(t *testing.T) {
		grantID := insert(bson.M{"codeRedeemed": false, "deleteDate": 1794200000000})

		redeemed, err := RedeemOAuthCode(session, grantID)
		require.NoError(t, err)
		require.False(t, redeemed)
		require.False(t, codeRedeemed(grantID), "a revoked grant is left alone")
	})

	t.Run("an unknown grant", func(t *testing.T) {
		redeemed, err := RedeemOAuthCode(session, primitive.NewObjectID())
		require.NoError(t, err)
		require.False(t, redeemed)
	})
}

// TestRedeemOAuthCode_RequiresMongoDB verifies that a collection the update cannot run on is a
// server fault, never a refusal that would read as a spent code
func TestRedeemOAuthCode_RequiresMongoDB(t *testing.T) {

	session, err := mockdb.New().Session(context.Background())
	require.NoError(t, err)

	redeemed, err := RedeemOAuthCode(session, primitive.NewObjectID())
	require.Error(t, err)
	require.False(t, redeemed)
}
