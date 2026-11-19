package handler

import (
	"net/http"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestRequireAccessToken confirms that only an access token carrying the scope, for the User named in the
// URL, may use an OAuth route, and that each refusal keeps its status through a wrap
func TestRequireAccessToken(t *testing.T) {

	const scope = model.OAuthUserTokenScopeActivityPubPortability
	userID := primitive.NewObjectID()

	accessToken := func(scopes string) model.Authorization {
		result := model.NewAuthorization()
		result.UserID = userID
		result.APIUser = true
		result.OAuthUserTokenID = primitive.NewObjectID()
		result.Scope = scopes
		return result
	}

	require.NoError(t, requireAccessToken(accessToken("read "+scope), scope, userID.Hex()))

	session := model.NewAuthorization()
	session.UserID = userID
	session.DomainOwner = true
	err := requireAccessToken(session, scope, userID.Hex())
	require.Equal(t, http.StatusUnauthorized, derp.ErrorCode(err), "a cookie session is refused, even the owner's")

	err = requireAccessToken(model.NewAuthorization(), scope, userID.Hex())
	require.Equal(t, http.StatusUnauthorized, derp.ErrorCode(err), "anonymous is refused")

	legacy := accessToken(scope)
	legacy.OAuthUserTokenID = primitive.NilObjectID
	err = requireAccessToken(legacy, scope, userID.Hex())
	require.Equal(t, http.StatusUnauthorized, derp.ErrorCode(err), "a token with no grant cannot be revoked, so it is refused")

	err = requireAccessToken(accessToken("read write"), scope, userID.Hex())
	require.Equal(t, http.StatusForbidden, derp.ErrorCode(err), "a token without the scope is refused")

	err = requireAccessToken(accessToken(scope), scope, primitive.NewObjectID().Hex())
	require.Equal(t, http.StatusNotFound, derp.ErrorCode(err), "another User's URL is not found")

	err = requireAccessToken(accessToken(scope), scope, "")
	require.Equal(t, http.StatusNotFound, derp.ErrorCode(err), "a URL with no User is not found")

	wrapped := derp.Wrap(requireAccessToken(accessToken("read"), scope, userID.Hex()), "test", "Checking access token")
	require.Equal(t, http.StatusForbidden, derp.ErrorCode(wrapped), "wrapping keeps the status")
}
