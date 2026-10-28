package mastodon

import (
	"errors"
	"net/http"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestRequireGrant confirms a live grant passes, a missing or absent grant is a 401, and a database
// error is not mistaken for a revoke
func TestRequireGrant(t *testing.T) {

	userID := primitive.NewObjectID()
	grantID := primitive.NewObjectID()
	authorization := model.Authorization{UserID: userID, OAuthUserTokenID: grantID}

	var askedUser, askedGrant primitive.ObjectID

	found := func(user primitive.ObjectID, grant primitive.ObjectID) error {
		askedUser, askedGrant = user, grant
		return nil
	}

	require.NoError(t, requireGrant(authorization, found))
	require.Equal(t, userID, askedUser, "the grant is loaded for the token's own User")
	require.Equal(t, grantID, askedGrant)

	revoked := func(primitive.ObjectID, primitive.ObjectID) error {
		return derp.NotFound("test", "OAuthUserToken not found")
	}

	err := requireGrant(authorization, revoked)
	require.Equal(t, http.StatusUnauthorized, derp.ErrorCode(err), "a disconnected app is signed out")
	require.Equal(t, http.StatusUnauthorized, derp.ErrorCode(derp.Wrap(err, "test", "Checking OAuth grant")), "wrapping keeps the 401")

	err = requireGrant(model.Authorization{UserID: userID}, found)
	require.Equal(t, http.StatusUnauthorized, derp.ErrorCode(err), "a token with no grant is refused")

	broken := func(primitive.ObjectID, primitive.ObjectID) error {
		return derp.Internal("test", "Database is unavailable", errors.New("connection refused"))
	}

	err = requireGrant(authorization, broken)
	require.Error(t, err)
	require.NotEqual(t, http.StatusUnauthorized, derp.ErrorCode(err), "a database error does not sign the app out")
}
