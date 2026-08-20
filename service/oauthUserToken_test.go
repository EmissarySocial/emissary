package service

import (
	"context"
	"testing"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	mockdb "github.com/benpate/data-mock"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// newTestOAuthUserTokenService wires an OAuthUserToken service against an in-memory
// mock database, with a working JWT service for minting access tokens.
func newTestOAuthUserTokenService(t *testing.T) (*OAuthUserToken, data.Session) {

	t.Helper()

	server := mockdb.New()

	session, err := server.Session(context.Background())
	require.Nil(t, err)

	jwtService := NewJWT()
	jwtService.Refresh(server)

	tokenService := NewOAuthUserToken()
	tokenService.jwtService = &jwtService
	tokenService.host = "https://example.com"

	return &tokenService, session
}

// newTestGrant builds a valid, unsaved ("new") code grant.
func newTestGrant() model.OAuthUserToken {
	grant := model.NewOAuthUserToken()
	grant.UserID = primitive.NewObjectID()
	grant.ClientID = primitive.NewObjectID()
	grant.Scopes = append(grant.Scopes, "read", "write")
	return grant
}

// persistTestGrant inserts a fresh grant (as the authorize step would), so its
// journal carries a current CreateDate and later Saves update in place.
func persistTestGrant(t *testing.T, service *OAuthUserToken, session data.Session) model.OAuthUserToken {
	t.Helper()
	grant := newTestGrant()
	require.Nil(t, service.Save(session, &grant, "create"))
	return grant
}

// TestExchangeCode verifies that an authorization code is redeemable exactly once
func TestExchangeCode(t *testing.T) {

	service, session := newTestOAuthUserTokenService(t)
	grant := persistTestGrant(t, service, session)

	require.Nil(t, service.ExchangeCode(session, &grant))

	// The code is now consumed.
	require.True(t, grant.CodeRedeemed)

	// An access token was minted.
	require.NotEmpty(t, grant.Token)

	// A generation-1 refresh token was minted, embedding this grant's ID.
	require.NotEmpty(t, grant.RefreshToken)
	gotID, gotGen, _, err := model.ParseRefreshToken(grant.RefreshToken)
	require.Nil(t, err)
	require.Equal(t, grant.OAuthUserTokenID, gotID)
	require.Equal(t, 1, gotGen)
	require.Equal(t, 1, grant.Generation)

	// RULE: the code is single-use — a second exchange must fail.
	require.NotNil(t, service.ExchangeCode(session, &grant))
}

// TestExchangeCode_Expired verifies that an expired code is rejected and never consumed
func TestExchangeCode_Expired(t *testing.T) {

	service, session := newTestOAuthUserTokenService(t)
	grant := newTestGrant()

	// Age the code past its TTL.
	grant.CreateDate = time.Now().Add(-model.OAuthCodeLifetime - time.Minute).UnixMilli()

	require.NotNil(t, service.ExchangeCode(session, &grant))
	require.False(t, grant.CodeRedeemed, "an expired code is never consumed")
}

// TestRefreshGrant_Rotate verifies that refreshing a grant advances its generation and rotates its secret
func TestRefreshGrant_Rotate(t *testing.T) {

	service, session := newTestOAuthUserTokenService(t)
	grant := persistTestGrant(t, service, session)
	require.Nil(t, service.ExchangeCode(session, &grant))

	// Capture the generation-1 refresh secret.
	_, _, secret1, err := model.ParseRefreshToken(grant.RefreshToken)
	require.Nil(t, err)

	// Refresh with the current secret.
	require.Nil(t, service.RefreshGrant(session, &grant, 1, secret1, ""))

	// The grant rotated to generation 2 with a fresh pair.
	require.Equal(t, 2, grant.Generation)
	require.NotEmpty(t, grant.Token)
	_, gotGen, secret2, err := model.ParseRefreshToken(grant.RefreshToken)
	require.Nil(t, err)
	require.Equal(t, 2, gotGen)
	require.NotEqual(t, secret1, secret2, "the refresh secret rotated")
}

// TestRefreshGrant_GarbageSecret verifies that a wrong secret is rejected without revoking the grant
func TestRefreshGrant_GarbageSecret(t *testing.T) {

	service, session := newTestOAuthUserTokenService(t)
	grant := persistTestGrant(t, service, session)
	require.Nil(t, service.ExchangeCode(session, &grant))

	// A wrong secret at the current generation is invalid_grant — NOT a reuse alarm.
	require.NotNil(t, service.RefreshGrant(session, &grant, 1, "garbage-secret", ""))

	// The grant survives (was not revoked).
	require.False(t, grant.IsDeleted())
}

// TestRefreshGrant_ReuseRevokes verifies that replaying a spent refresh secret revokes the whole grant
func TestRefreshGrant_ReuseRevokes(t *testing.T) {

	service, session := newTestOAuthUserTokenService(t)
	grant := persistTestGrant(t, service, session)
	require.Nil(t, service.ExchangeCode(session, &grant))

	_, _, secret1, err := model.ParseRefreshToken(grant.RefreshToken)
	require.Nil(t, err)

	// Rotate to generation 2.
	require.Nil(t, service.RefreshGrant(session, &grant, 1, secret1, ""))

	// Force the prior secret past the grace window, then present it again.
	grant.RotatedAt = time.Now().Add(-model.OAuthRefreshGracePeriod - time.Minute).Unix()

	err = service.RefreshGrant(session, &grant, 1, secret1, "")
	require.NotNil(t, err, "reuse of a superseded secret is rejected")

	// The grant was revoked — it is gone from the datastore.
	reloaded := model.NewOAuthUserToken()
	require.NotNil(t, service.LoadByID(session, grant.UserID, grant.OAuthUserTokenID, &reloaded), "the grant is revoked on reuse")
}

// TestRefreshGrant_ScopeNarrowing verifies which scope changes a refresh request may ask for
func TestRefreshGrant_ScopeNarrowing(t *testing.T) {

	service, session := newTestOAuthUserTokenService(t)

	t.Run("narrowing to a subset is allowed", func(t *testing.T) {
		grant := persistTestGrant(t, service, session) // scopes: read write
		require.Nil(t, service.ExchangeCode(session, &grant))
		_, _, secret, _ := model.ParseRefreshToken(grant.RefreshToken)

		require.Nil(t, service.RefreshGrant(session, &grant, 1, secret, "read"))
		require.Equal(t, "read", grant.JSONResponse()["scope"], "the issued token carries only the narrowed scope")
	})

	t.Run("widening beyond the grant is rejected", func(t *testing.T) {
		grant := newTestGrant()
		grant.Scopes = grant.Scopes[:0]
		grant.Scopes = append(grant.Scopes, "read") // grant only read
		require.Nil(t, service.Save(session, &grant, "create"))
		require.Nil(t, service.ExchangeCode(session, &grant))
		_, _, secret, _ := model.ParseRefreshToken(grant.RefreshToken)

		require.NotNil(t, service.RefreshGrant(session, &grant, 1, secret, "read write"))
	})
}
