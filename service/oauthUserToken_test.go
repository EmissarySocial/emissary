package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/secretcheck"
	"github.com/benpate/data"
	mockdb "github.com/benpate/data-mock"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/sliceof"
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

// newReplicaSetOAuthUserTokenService wires an OAuthUserToken service against a throwaway database on
// the local replica set, because redeeming a code is a conditional update that the mock cannot run
func newReplicaSetOAuthUserTokenService(t *testing.T) (*OAuthUserToken, data.Server, data.Session) {

	t.Helper()

	server, session := newReplicaSetSession(t)

	jwtService := NewJWT()
	jwtService.Refresh(server)

	tokenService := NewOAuthUserToken()
	tokenService.jwtService = &jwtService
	tokenService.host = "https://example.com"

	return &tokenService, server, session
}

// issueTestTokens gives a saved grant its first access and refresh tokens, as a redeemed code
// would, without redeeming one
func issueTestTokens(t *testing.T, service *OAuthUserToken, session data.Session, grant *model.OAuthUserToken) {

	t.Helper()

	require.Nil(t, service.mintInitialTokens(grant, time.Now()))
	grant.CodeRedeemed = true
	require.Nil(t, service.Save(session, grant, "issue"))
}

// TestExchangeCode verifies that an authorization code is redeemable exactly once
func TestExchangeCode(t *testing.T) {

	service, _, session := newReplicaSetOAuthUserTokenService(t)
	grant := persistTestGrant(t, service, session)

	// A copy loaded before the redemption, as a second server would hold it
	stale := model.NewOAuthUserToken()
	require.Nil(t, service.LoadByID(session, grant.UserID, grant.OAuthUserTokenID, &stale))

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

	// The stored grant carries the redemption and the refresh hash
	stored := model.NewOAuthUserToken()
	require.Nil(t, service.LoadByID(session, grant.UserID, grant.OAuthUserTokenID, &stored))
	require.True(t, stored.CodeRedeemed)
	require.Equal(t, grant.RefreshHash, stored.RefreshHash)

	// RULE: the code is single-use — a second exchange must fail.
	require.NotNil(t, service.ExchangeCode(session, &grant))

	// RULE: ...and the database refuses it even to a copy that still reads the code as unused
	require.False(t, stale.CodeRedeemed)
	err = service.ExchangeCode(session, &stale)
	require.NotNil(t, err)
	require.True(t, derp.IsBadRequest(err), "a spent code is the client's mistake, not a server fault")
	require.Empty(t, stale.Token, "the refused copy mints nothing")
}

// TestExchangeCode_TwoServersRedeemOneCode races two redemptions of one code, sharing nothing but the
// database, and requires exactly one of them to win every time, with or without a transaction
func TestExchangeCode_TwoServersRedeemOneCode(t *testing.T) {

	service, server, session := newReplicaSetOAuthUserTokenService(t)

	// redeem loads its own copy of the grant, as each server's request would, and redeems it
	redeem := func(session data.Session, grant model.OAuthUserToken) error {

		copied := model.NewOAuthUserToken()
		if err := service.LoadByID(session, grant.UserID, grant.OAuthUserTokenID, &copied); err != nil {
			return err
		}

		return service.ExchangeCode(session, &copied)
	}

	// Every POST runs in a transaction, where a write conflict retries the loser
	inTransaction := func(grant model.OAuthUserToken) error {
		_, err := server.WithTransaction(context.Background(), func(txn data.Session) (any, error) {
			return nil, redeem(txn, grant)
		})
		return err
	}

	// RULE: Single use must not depend on the caller opening a transaction
	inSession := func(grant model.OAuthUserToken) error {
		plain, err := server.Session(context.Background())
		if err != nil {
			return err
		}
		defer plain.Close()
		return redeem(plain, grant)
	}

	for name, attempt := range map[string]func(model.OAuthUserToken) error{"transaction": inTransaction, "session": inSession} {

		t.Run(name, func(t *testing.T) {

			for round := range 10 {

				grant := persistTestGrant(t, service, session)
				start := make(chan struct{})
				results := make(chan error, 2)

				for range 2 {
					go func() {
						<-start
						results <- attempt(grant)
					}()
				}

				close(start)

				var redeemed, refused int
				for range 2 {
					if err := <-results; err == nil {
						redeemed++
					} else if derp.IsBadRequest(err) {
						refused++
					}
				}

				require.Equal(t, 1, redeemed, "round %d: exactly one server redeems the code", round)
				require.Equal(t, 1, refused, "round %d: the other is refused as a spent code", round)
			}
		})
	}
}

// TestExchangeCode_IdentityOnly verifies that a grant with no scope spends its code and mints nothing
func TestExchangeCode_IdentityOnly(t *testing.T) {

	service, _, session := newReplicaSetOAuthUserTokenService(t)

	grant := newTestGrant()
	grant.Scopes = sliceof.String{""} // what an authorization request with no scope stores
	require.Nil(t, service.Save(session, &grant, "create"))

	require.Nil(t, service.ExchangeCode(session, &grant))
	require.True(t, grant.CodeRedeemed)
	require.Empty(t, grant.Token, "IndieAuth §5.3.3: no access token without a scope")
	require.Empty(t, grant.RefreshToken)

	stored := model.NewOAuthUserToken()
	require.Nil(t, service.LoadByID(session, grant.UserID, grant.OAuthUserTokenID, &stored))
	require.True(t, stored.CodeRedeemed, "the code is spent")
	require.Empty(t, stored.RefreshHash, "nothing can be refreshed")

	// The code is still single-use
	stale := model.NewOAuthUserToken()
	stale.OAuthUserTokenID = grant.OAuthUserTokenID
	stale.CreateDate = grant.CreateDate
	require.NotNil(t, service.ExchangeCode(session, &stale))
}

// TestExchangeCode_RevokedGrant verifies that a code whose grant was revoked cannot be redeemed
func TestExchangeCode_RevokedGrant(t *testing.T) {

	service, _, session := newReplicaSetOAuthUserTokenService(t)
	grant := persistTestGrant(t, service, session)

	// A copy loaded before the revocation
	stale := grant
	require.Nil(t, service.Delete(session, &grant, "revoked"))

	err := service.ExchangeCode(session, &stale)
	require.NotNil(t, err)
	require.True(t, derp.IsBadRequest(err))
	require.Empty(t, stale.Token)
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
	issueTestTokens(t, service, session, &grant)

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
	issueTestTokens(t, service, session, &grant)

	// A wrong secret at the current generation is invalid_grant — NOT a reuse alarm.
	require.NotNil(t, service.RefreshGrant(session, &grant, 1, "garbage-secret", ""))

	// The grant survives (was not revoked).
	require.False(t, grant.IsDeleted())
}

// TestRefreshGrant_ReuseRevokes verifies that replaying a spent refresh secret revokes the whole grant
func TestRefreshGrant_ReuseRevokes(t *testing.T) {

	service, session := newTestOAuthUserTokenService(t)
	grant := persistTestGrant(t, service, session)
	issueTestTokens(t, service, session, &grant)

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
		issueTestTokens(t, service, session, &grant)
		_, _, secret, _ := model.ParseRefreshToken(grant.RefreshToken)

		require.Nil(t, service.RefreshGrant(session, &grant, 1, secret, "read"))
		require.Equal(t, "read", grant.JSONResponse()["scope"], "the issued token carries only the narrowed scope")
	})

	t.Run("widening beyond the grant is rejected", func(t *testing.T) {
		grant := newTestGrant()
		grant.Scopes = grant.Scopes[:0]
		grant.Scopes = append(grant.Scopes, "read") // grant only read
		require.Nil(t, service.Save(session, &grant, "create"))
		issueTestTokens(t, service, session, &grant)
		_, _, secret, _ := model.ParseRefreshToken(grant.RefreshToken)

		require.NotNil(t, service.RefreshGrant(session, &grant, 1, secret, "read write"))
	})
}

// TestCreate_CopiesMastodonApp confirms a grant records whether its Client is a Mastodon app
func TestCreate_CopiesMastodonApp(t *testing.T) {

	for _, mastodonApp := range []bool{true, false} {

		service, session := newTestOAuthUserTokenService(t)

		client := model.NewOAuthClient()
		client.ClientSecret = "secret"
		client.RedirectURIs = append(client.RedirectURIs, "icecubesapp://oauth")
		client.Scopes = append(client.Scopes, "read", "write")
		client.MastodonApp = mastodonApp

		authorization := model.Authorization{UserID: primitive.NewObjectID()}
		request := model.OAuthAuthorizationRequest{ResponseType: "code", ClientID: client.ClientID.Hex()}

		grant, err := service.Create(session, client, authorization, request)
		require.Nil(t, err)
		require.Equal(t, mastodonApp, grant.MastodonApp)

		// The flag is saved with the grant, so token minting can read it later. The mock database
		// cannot match the inlined deleteDate, so this reads the record by _id directly.
		saved := model.NewOAuthUserToken()
		require.Nil(t, service.collection(session).Load(exp.Equal("_id", grant.OAuthUserTokenID), &saved))
		require.Equal(t, mastodonApp, saved.MastodonApp)
	}
}

// TestCreate_SavesThePKCEChallenge confirms a code grant is saved bound to the challenge its request
// carried, so only the client's verifier redeems it
func TestCreate_SavesThePKCEChallenge(t *testing.T) {

	// BUG-361 pins the consent form's half of this; RFC 7636 Appendix B's verifier and S256 challenge
	const verifier = "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	const challenge = "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"

	service, session := newTestOAuthUserTokenService(t)

	// A public client: no secret, so the challenge is the only thing protecting its code
	client := model.NewOAuthClient()
	client.RedirectURIs = append(client.RedirectURIs, "https://client.example/callback")

	authorization := model.Authorization{UserID: primitive.NewObjectID()}
	request := model.OAuthAuthorizationRequest{
		ResponseType:        "code",
		ClientID:            client.ClientID.Hex(),
		CodeChallenge:       challenge,
		CodeChallengeMethod: model.PKCEMethodS256,
	}

	grant, err := service.Create(session, client, authorization, request)
	require.Nil(t, err)

	// Read the saved record by _id, as TestCreate_CopiesMastodonApp does
	saved := model.NewOAuthUserToken()
	require.Nil(t, service.collection(session).Load(exp.Equal("_id", grant.OAuthUserTokenID), &saved))
	require.True(t, saved.HasPKCEChallenge())
	require.Nil(t, saved.VerifyPKCE(verifier))
	require.NotNil(t, saved.VerifyPKCE(strings.Repeat("x", 43)))

	// A public client's request without a challenge is refused before anything is saved
	request.CodeChallenge = ""
	request.CodeChallengeMethod = ""
	_, err = service.Create(session, client, authorization, request)
	require.NotNil(t, err)
}

// TestCreate_SavesTheRedirectURI confirms a grant records the redirect_uri its code was issued to, including
// the default a registered client's request left out
func TestCreate_SavesTheRedirectURI(t *testing.T) {

	service, session := newTestOAuthUserTokenService(t)

	client := model.NewOAuthClient()
	client.ClientSecret = "secret"
	client.RedirectURIs = append(client.RedirectURIs, "https://client.example/first", "https://client.example/second")

	authorization := model.Authorization{UserID: primitive.NewObjectID()}

	// create issues a code for the redirect_uri, and reads the saved grant back by _id
	create := func(redirectURI string) model.OAuthUserToken {
		t.Helper()

		request := model.OAuthAuthorizationRequest{ResponseType: "code", ClientID: client.ClientID.Hex(), RedirectURI: redirectURI}
		grant, err := service.Create(session, client, authorization, request)
		require.Nil(t, err)

		saved := model.NewOAuthUserToken()
		require.Nil(t, service.collection(session).Load(exp.Equal("_id", grant.OAuthUserTokenID), &saved))
		return saved
	}

	require.Equal(t, "https://client.example/second", create("https://client.example/second").RedirectURI())
	require.Equal(t, "https://client.example/first", create("").RedirectURI(), "a request with none records the default its code went to")
}

// TestOAuthUserToken_ErrorsCarryNoGrant confirms a failed write names the grant by its ID, and never carries
// the grant, whose PKCE challenge is the verifier itself under the "plain" method
func TestOAuthUserToken_ErrorsCarryNoGrant(t *testing.T) {

	challenge := strings.Repeat("v", 43)

	// newGrant returns an unsaved grant bound to the challenge
	newGrant := func() model.OAuthUserToken {
		grant := newTestGrant()
		grant.SetPKCEChallenge(challenge, model.PKCEMethodPlain)
		grant.SetRedirectURI("https://client.example/callback")
		return grant
	}

	t.Run("Save refused by the database", func(t *testing.T) {
		service, session := newTestOAuthUserTokenService(t)
		grant := newGrant()

		err := service.Save(session, &grant, "ERROR: synthetic")
		require.NotNil(t, err)
		require.Equal(t, "service.OAuthUserToken.Save", derp.Location(err))
		secretcheck.RequireAbsent(t, err, challenge)
	})

	t.Run("Save refused by the schema", func(t *testing.T) {
		service, session := newTestOAuthUserTokenService(t)
		grant := newGrant()
		grant.Scopes = append(grant.Scopes, strings.Repeat("s", 200))

		err := service.Save(session, &grant, "create")
		require.NotNil(t, err)
		require.Equal(t, "service.OAuthUserToken.Save", derp.Location(err))
		secretcheck.RequireAbsent(t, err, challenge)
	})

	t.Run("Delete", func(t *testing.T) {
		service, session := newTestOAuthUserTokenService(t)
		grant := newGrant()
		require.Nil(t, service.Save(session, &grant, "create"))

		err := service.Delete(session, &grant, "ERROR: synthetic")
		require.NotNil(t, err)
		require.Equal(t, "service.OAuthUserToken.Delete", derp.Location(err))
		secretcheck.RequireAbsent(t, err, challenge)
	})

	t.Run("Create", func(t *testing.T) {
		service, session := newTestOAuthUserTokenService(t)

		client := model.NewOAuthClient()
		client.RedirectURIs = append(client.RedirectURIs, "https://client.example/callback")

		// A scope longer than the schema allows fails the save after the grant is built
		request := model.OAuthAuthorizationRequest{
			ResponseType:        "code",
			ClientID:            client.ClientID.Hex(),
			Scope:               strings.Repeat("s", 200),
			CodeChallenge:       challenge,
			CodeChallengeMethod: model.PKCEMethodPlain,
		}

		_, err := service.Create(session, client, model.Authorization{UserID: primitive.NewObjectID()}, request)
		require.NotNil(t, err)
		require.Equal(t, "service.OAuthUserToken.Create", derp.Location(err))
		secretcheck.RequireAbsent(t, err, challenge)
	})
}

// TestExchangeCode_MastodonAppTokenHasNoExpiry confirms a Mastodon app's access token carries no
// "exp" claim and still parses as valid, while other grants keep theirs
func TestExchangeCode_MastodonAppTokenHasNoExpiry(t *testing.T) {

	service, _, session := newReplicaSetOAuthUserTokenService(t)

	for _, mastodonApp := range []bool{true, false} {

		grant := newTestGrant()
		grant.MastodonApp = mastodonApp
		require.Nil(t, service.Save(session, &grant, "create"))
		require.Nil(t, service.ExchangeCode(session, &grant))

		parsed, err := service.jwtService.ParseString(grant.Token)
		require.Nil(t, err)
		require.True(t, parsed.Valid)

		expiry, err := parsed.Claims.GetExpirationTime()
		require.Nil(t, err)

		if mastodonApp {
			require.Nil(t, expiry, "a Mastodon app's token does not expire")
			continue
		}

		require.NotNil(t, expiry)
		require.WithinDuration(t, time.Now().Add(model.OAuthAccessTokenLifetime), expiry.Time, time.Minute)
	}
}
