package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/dataset"
	"github.com/EmissarySocial/emissary/tools/secretcheck"
	"github.com/benpate/data"
	mongodb "github.com/benpate/data-mongo"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/sliceof"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// The credentials a token request presents.  None of them may appear in a reported error.
const (
	oauthPresentedSecret   = "presented-client-s3cr3t"
	oauthStoredSecret      = "stored-client-s3cr3t"
	oauthCodeVerifier      = "code-verifier-s3cr3t"
	oauthRefreshSecret     = "refresh-token-s3cr3t"
	oauthRefreshGeneration = 1

	// oauthValidVerifier is a well-formed PKCE verifier (RFC 7636 §4.1), used as its own plain challenge
	oauthValidVerifier = "pkce-verifier-pkce-verifier-pkce-verifier-pkce"

	// oauthTestConnection is the local replica set that the transaction tests require
	oauthTestConnection = "mongodb://localhost:27017/?directConnection=true"
)

// oauthSession is a data.Session that serves one fixed record per collection, whatever the
// criteria, or fails every load with its failure
type oauthSession struct {
	data.Session
	client  *model.OAuthClient
	grant   *model.OAuthUserToken
	user    *model.User
	failure error
}

// Collection returns the fixed record for this collection, if any
func (session oauthSession) Collection(name string) data.Collection {

	switch name {

	case "OAuthClient":
		return oauthCollection{record: session.client, failure: session.failure}

	case "OAuthUserToken":
		return oauthCollection{record: session.grant, failure: session.failure}

	case "User":
		return oauthCollection{record: session.user, failure: session.failure}
	}

	return oauthCollection{failure: session.failure}
}

// oauthCollection is a data.Collection whose Load returns its one record, or NotFound
type oauthCollection struct {
	data.Collection
	record  any
	failure error
}

// Load copies the fixed record into target, or fails with NotFound when there is none
func (collection oauthCollection) Load(_ exp.Expression, target data.Object, _ ...option.Option) error {

	if collection.failure != nil {
		return collection.failure
	}

	switch record := collection.record.(type) {

	case *model.OAuthClient:
		if typed, ok := target.(*model.OAuthClient); ok && record != nil {
			*typed = *record
			return nil
		}

	case *model.OAuthUserToken:
		if typed, ok := target.(*model.OAuthUserToken); ok && record != nil {
			*typed = *record
			return nil
		}

	case *model.User:
		if typed, ok := target.(*model.User); ok && record != nil {
			*typed = *record
			return nil
		}
	}

	return derp.NotFound("handler.oauthCollection.Load", "Record not found")
}

// TestPostOAuthToken_AnswersClientMistakesAsJSON drives every refusal of the token endpoint, and
// requires an RFC 6749 §5.2 answer that carries no credential the request presented
func TestPostOAuthToken_AnswersClientMistakesAsJSON(t *testing.T) {

	// BUG-173: these sites attached the whole OAuthUserTokenRequest, so the secret, code, verifier,
	// and refresh token were printed and stored.  oauthSession serves fixed records, because mockdb
	// cannot match deleteDate.

	// A form that presents every credential at once, for the grant being tested
	form := func(clientID string, grantType string, code string) url.Values {
		return url.Values{
			"client_id":     {clientID},
			"grant_type":    {grantType},
			"code":          {code},
			"client_secret": {oauthPresentedSecret},
			"code_verifier": {oauthCodeVerifier},
			"refresh_token": {model.BuildRefreshToken(primitive.NewObjectID(), oauthRefreshGeneration, oauthRefreshSecret)},
		}
	}

	// A confidential client whose stored secret is the one the form presents, so a request passes client authentication
	authenticated := func() (*model.OAuthClient, *model.OAuthUserToken) {
		client := newOAuthClient(oauthPresentedSecret)
		grant := newOAuthGrant(client)
		grant.SetRedirectURI("https://client.example/callback")
		return client, grant
	}

	t.Run("UnknownClient", func(t *testing.T) {
		session := oauthSession{}
		requireOAuthRefusal(t, session, form(primitive.NewObjectID().Hex(), "authorization_code", ""), "invalid_client", "Invalid client_id")
	})

	t.Run("MissingGrantType", func(t *testing.T) {
		client := newOAuthClient(oauthStoredSecret)
		session := oauthSession{client: client}

		// An empty grant_type and an absent one are the same omission
		values := form(client.ClientID.Hex(), "", "")
		requireOAuthRefusal(t, session, values, "invalid_request", "Missing grant_type")

		values.Del("grant_type")
		requireOAuthRefusal(t, session, values, "invalid_request", "Missing grant_type")
	})

	t.Run("UnsupportedGrantType", func(t *testing.T) {
		client := newOAuthClient(oauthStoredSecret)
		session := oauthSession{client: client}
		requireOAuthRefusal(t, session, form(client.ClientID.Hex(), "password", ""), "unsupported_grant_type", "Unsupported grant_type")
	})

	t.Run("MalformedCode", func(t *testing.T) {
		client := newOAuthClient(oauthStoredSecret)
		session := oauthSession{client: client}
		requireOAuthRefusal(t, session, form(client.ClientID.Hex(), "authorization_code", "not-an-objectid"), "invalid_grant", "Invalid code")
	})

	t.Run("UnknownCode", func(t *testing.T) {
		client := newOAuthClient(oauthStoredSecret)
		session := oauthSession{client: client}
		requireOAuthRefusal(t, session, form(client.ClientID.Hex(), "authorization_code", primitive.NewObjectID().Hex()), "invalid_grant", "Loading OAuthUserToken")
	})

	t.Run("WrongSecretForCode", func(t *testing.T) {
		client := newOAuthClient(oauthStoredSecret)
		grant := newOAuthGrant(client)
		session := oauthSession{client: client, grant: grant}
		requireOAuthRefusal(t, session, form(client.ClientID.Hex(), "authorization_code", grant.Code()), "invalid_client", "Invalid client_secret")
	})

	t.Run("PublicClientWithoutPKCE", func(t *testing.T) {
		client := newOAuthClient("")
		grant := newOAuthGrant(client)
		session := oauthSession{client: client, grant: grant}
		requireOAuthRefusal(t, session, form(client.ClientID.Hex(), "authorization_code", grant.Code()), "invalid_grant", "This client must use PKCE (a code_verifier is required)")
	})

	t.Run("WrongVerifier", func(t *testing.T) {
		client, grant := authenticated()
		grant.SetPKCEChallenge(oauthValidVerifier, model.PKCEMethodPlain)
		session := oauthSession{client: client, grant: grant}
		requireOAuthRefusal(t, session, form(client.ClientID.Hex(), "authorization_code", grant.Code()), "invalid_grant", "Invalid PKCE code_verifier")
	})

	t.Run("WrongRedirectURI", func(t *testing.T) {
		client, grant := authenticated()
		session := oauthSession{client: client, grant: grant}
		values := form(client.ClientID.Hex(), "authorization_code", grant.Code())
		values.Set("redirect_uri", "https://evil.example/callback")
		requireOAuthRefusal(t, session, values, "invalid_grant", "Invalid redirect_uri")
	})

	t.Run("URLClientWithoutRedirectURI", func(t *testing.T) {
		client := newOAuthClient("")
		client.ClientURL = "https://client.example/"
		grant := newOAuthGrant(client)
		grant.SetRedirectURI("https://client.example/callback")
		grant.SetPKCEChallenge(oauthValidVerifier, model.PKCEMethodPlain)
		session := oauthSession{client: client, grant: grant}
		values := form(client.ClientID.Hex(), "authorization_code", grant.Code())
		values.Set("code_verifier", oauthValidVerifier)
		requireOAuthRefusal(t, session, values, "invalid_grant", "Invalid redirect_uri")
	})

	t.Run("UnknownUser", func(t *testing.T) {
		client, grant := authenticated()
		session := oauthSession{client: client, grant: grant}
		requireOAuthRefusal(t, session, form(client.ClientID.Hex(), "authorization_code", grant.Code()), "invalid_grant", "Loading User")
	})

	t.Run("NoScopeForAUserTheDomainDoesNotVouchFor", func(t *testing.T) {
		client, grant := authenticated()
		grant.Scopes = sliceof.String{""}
		user := model.NewUser()
		user.IsOwner = true
		user.IsPublic = true
		session := oauthSession{client: client, grant: grant, user: &user}

		// Refused before the code is spent: redeeming it would reach the database, which this session lacks
		requireOAuthRefusal(t, session, form(client.ClientID.Hex(), "authorization_code", grant.Code()), "invalid_grant", "This code carries no scope, and this server vouches for no identity to return")
	})

	t.Run("WrongSecretForRefresh", func(t *testing.T) {
		client := newOAuthClient(oauthStoredSecret)
		session := oauthSession{client: client}
		requireOAuthRefusal(t, session, form(client.ClientID.Hex(), "refresh_token", ""), "invalid_client", "Invalid client_secret")
	})

	t.Run("MalformedRefreshToken", func(t *testing.T) {
		client := newOAuthClient(oauthPresentedSecret)
		session := oauthSession{client: client}
		values := form(client.ClientID.Hex(), "refresh_token", "")
		values.Set("refresh_token", "not-a-refresh-token")
		requireOAuthRefusal(t, session, values, "invalid_grant", "Invalid refresh_token")
	})
}

// TestPostOAuthToken_ReturnsServerFaults confirms that a failure of the server's own is returned for
// errorHandler to report, not answered as the client's mistake, and still carries no credential
func TestPostOAuthToken_ReturnsServerFaults(t *testing.T) {

	session := oauthSession{failure: derp.Internal("handler.test", "Database is down")}
	values := url.Values{
		"client_id":     {primitive.NewObjectID().Hex()},
		"client_secret": {oauthPresentedSecret},
		"code_verifier": {oauthCodeVerifier},
	}

	ctx, recorder := oauthTokenRequest(values)
	err := PostOAuthToken(ctx, &service.Factory{}, session)

	requireNoOAuthSecrets(t, err, "handler.PostOAuthToken", "Invalid client_id")
	require.True(t, derp.IsServerError(err))
	require.Empty(t, recorder.Body.String(), "nothing is written, so errorHandler can answer")
}

// TestPostOAuthToken_IdentityOnlyCode redeems a code with no scope, inside a transaction as every POST
// is, and requires `me` alone, once (IndieAuth §5.3.3)
func TestPostOAuthToken_IdentityOnlyCode(t *testing.T) {

	server, session := newOAuthTestServer(t)

	client := newOAuthClient("")
	client.ClientURL = "https://client.example/"
	require.NoError(t, session.Collection("OAuthClient").Save(client, "test"))

	user := model.NewUser()
	user.IsOwner = true
	user.IsPrimary = true
	user.IsPublic = true
	user.ProfileURL = "https://example.com/@" + user.UserID.Hex()
	require.NoError(t, session.Collection("User").Save(&user, "test"))

	grant := newOAuthGrant(client)
	grant.UserID = user.UserID
	grant.Scopes = sliceof.String{""}
	grant.SetRedirectURI("https://client.example/callback")
	grant.SetPKCEChallenge(oauthValidVerifier, model.PKCEMethodPlain)
	require.NoError(t, session.Collection("OAuthUserToken").Save(grant, "test"))

	values := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {client.ClientID.Hex()},
		"code":          {grant.Code()},
		"redirect_uri":  {"https://client.example/callback"},
		"code_verifier": {oauthValidVerifier},
	}

	// The first redemption answers with `me` alone
	recorder := postOAuthTokenInTransaction(t, server, values)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.JSONEq(t, `{"me":"`+user.ProfileURL+`"}`, recorder.Body.String())

	// RULE: The code is single-use
	recorder = postOAuthTokenInTransaction(t, server, values)
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.JSONEq(t, `{"error":"invalid_grant","error_description":"Exchanging authorization code"}`, recorder.Body.String())
}

// TestPostOAuthToken_RefreshReuseRevocationCommits replays a superseded refresh token inside a
// transaction, as every POST is, and requires the revocation to outlive the refusal
func TestPostOAuthToken_RefreshReuseRevocationCommits(t *testing.T) {

	server, session := newOAuthTestServer(t)

	client := newOAuthClient("")
	require.NoError(t, session.Collection("OAuthClient").Save(client, "test"))

	// A grant already rotated to generation 2, past the grace window of generation 1
	const firstSecret = "first-refresh-secret"
	grant := newOAuthGrant(client)
	grant.InitRefresh(firstSecret, time.Now())
	grant.RotateRefresh("second-refresh-secret", time.Now().Add(-model.OAuthRefreshGracePeriod-time.Minute))
	require.NoError(t, session.Collection("OAuthUserToken").Save(grant, "test"))

	recorder := postOAuthTokenInTransaction(t, server, url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {client.ClientID.Hex()},
		"refresh_token": {model.BuildRefreshToken(grant.OAuthUserTokenID, 1, firstSecret)},
	})

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"error":"invalid_grant"`)

	// RULE: The refusal commits, so the grant stays revoked.  A returned error would roll it back.
	stored := model.NewOAuthUserToken()
	require.NoError(t, session.Collection("OAuthUserToken").Load(exp.Equal("_id", grant.OAuthUserTokenID), &stored))
	require.True(t, stored.IsDeleted(), "a reused refresh token revokes the grant")
}

// TestPostOAuthRevoke_RefreshTokenNeedsItsSecret confirms a refresh token revokes its grant only with a secret the grant issued
func TestPostOAuthRevoke_RefreshTokenNeedsItsSecret(t *testing.T) {

	const firstSecret = "first-refresh-secret"
	const secondSecret = "second-refresh-secret"

	// revoke posts the token for a new public client's grant, and reports whether the grant was revoked
	revoke := func(t *testing.T, generation int, secret string) bool {

		t.Helper()
		server, session := newOAuthTestServer(t)

		client := newOAuthClient("")
		require.NoError(t, session.Collection("OAuthClient").Save(client, "test"))

		grant := newOAuthGrant(client)
		grant.InitRefresh(firstSecret, time.Now())
		grant.RotateRefresh(secondSecret, time.Now())
		require.NoError(t, session.Collection("OAuthUserToken").Save(grant, "test"))

		recorder := postOAuthInTransaction(t, server, PostOAuthRevoke, url.Values{
			"client_id": {client.ClientID.Hex()},
			"token":     {model.BuildRefreshToken(grant.OAuthUserTokenID, generation, secret)},
		})

		// RULE: Every answer is 200, so a caller cannot tell a wrong secret from an unknown grant (RFC 7009)
		require.Equal(t, http.StatusOK, recorder.Code)

		stored := model.NewOAuthUserToken()
		require.NoError(t, session.Collection("OAuthUserToken").Load(exp.Equal("_id", grant.OAuthUserTokenID), &stored))
		return stored.IsDeleted()
	}

	t.Run("the current secret revokes", func(t *testing.T) {
		require.True(t, revoke(t, 2, secondSecret))
	})

	t.Run("the secret before it revokes", func(t *testing.T) {
		require.True(t, revoke(t, 1, firstSecret))
	})

	t.Run("a guessed secret revokes nothing", func(t *testing.T) {
		require.False(t, revoke(t, 2, "x"))
	})

	t.Run("the right secret under the wrong generation revokes nothing", func(t *testing.T) {
		require.False(t, revoke(t, 1, secondSecret))
	})
}

// TestPostOAuthAuthorization_ReturnsTheIssuer confirms both authorization redirects name this server as their
// issuer (RFC 9207, IndieAuth §5.2.1), and that a state cannot add parameters to the token fragment
func TestPostOAuthAuthorization_ReturnsTheIssuer(t *testing.T) {

	const issuer = "https://example.com"

	grant := model.NewOAuthUserToken()
	grant.Token = "header.payload.signature"

	transaction := model.OAuthAuthorizationRequest{
		RedirectURI: "https://client.example/callback?keep=1",
		State:       "a&iss=https://evil.example b",
	}

	t.Run("code", func(t *testing.T) {
		ctx, recorder := oauthAuthorizationResponse()
		require.NoError(t, postOAuthAuthorization_code(ctx, issuer, grant, transaction))
		require.Equal(t, http.StatusFound, recorder.Code)

		location, err := url.Parse(recorder.Header().Get("Location"))
		require.NoError(t, err)

		query := location.Query()
		require.Equal(t, grant.Code(), query.Get("code"))
		require.Equal(t, transaction.State, query.Get("state"))
		require.Equal(t, []string{issuer}, query["iss"])
		require.Equal(t, "1", query.Get("keep"))
	})

	t.Run("token", func(t *testing.T) {
		ctx, recorder := oauthAuthorizationResponse()
		require.NoError(t, postOAuthAuthorization_token(ctx, issuer, grant, transaction))
		require.Equal(t, http.StatusFound, recorder.Code)

		target, rawFragment, found := strings.Cut(recorder.Header().Get("Location"), "#")
		require.True(t, found)
		require.Equal(t, transaction.RedirectURI, target)

		fragment, err := url.ParseQuery(rawFragment)
		require.NoError(t, err)
		require.Equal(t, url.Values{
			"access_token": {grant.Token},
			"state":        {transaction.State},
			"token_type":   {"Bearer"},
			"iss":          {issuer},
		}, fragment)
	})

	t.Run("out of band", func(t *testing.T) {
		outOfBand := transaction
		outOfBand.RedirectURI = "urn:ietf:wg:oauth:2.0:oob"

		ctx, recorder := oauthAuthorizationResponse()
		require.NoError(t, postOAuthAuthorization_code(ctx, issuer, grant, outOfBand))
		require.Equal(t, http.StatusOK, recorder.Code)
		require.Contains(t, recorder.Body.String(), grant.Code())
		require.NotContains(t, recorder.Body.String(), issuer)
	})
}

// TestRouteOAuthAuthorization confirms that a code redemption and the consent form's submit each reach their own
// handler, and that the chosen handler can still read the body the router parsed
func TestRouteOAuthAuthorization(t *testing.T) {

	// route posts the body to the router, and returns the branch it took and the client_id that branch bound
	route := func(target string, body url.Values) (string, string) {

		request := httptest.NewRequest(http.MethodPost, target, strings.NewReader(body.Encode()))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		ctx := echo.New().NewContext(request, httptest.NewRecorder())

		branch := ""
		transaction := model.NewOAuthUserTokenRequest()

		handlerFor := func(name string) echo.HandlerFunc {
			return func(ctx echo.Context) error {
				branch = name
				return ctx.Bind(&transaction)
			}
		}

		require.NoError(t, RouteOAuthAuthorization(handlerFor("redemption"), handlerFor("consent"))(ctx))
		return branch, transaction.ClientID
	}

	// Every field the consent form posts (theme-global/oauth.html)
	consentForm := url.Values{
		"client_id":             {"consent-client"},
		"redirect_uri":          {"https://client.example/callback"},
		"scope":                 {"profile"},
		"state":                 {"state-value"},
		"response_type":         {"code"},
		"code_challenge":        {oauthValidVerifier},
		"code_challenge_method": {model.PKCEMethodPlain},
	}

	t.Run("ConsentForm", func(t *testing.T) {
		branch, clientID := route("/oauth/authorize", consentForm)
		require.Equal(t, "consent", branch)
		require.Equal(t, "consent-client", clientID, "the consent handler still reads the body")
	})

	t.Run("IndieAuthRedemption", func(t *testing.T) {
		branch, clientID := route("/oauth/authorize", url.Values{
			"grant_type":    {"authorization_code"},
			"code":          {primitive.NewObjectID().Hex()},
			"client_id":     {"redeeming-client"},
			"redirect_uri":  {"https://client.example/callback"},
			"code_verifier": {oauthValidVerifier},
		})
		require.Equal(t, "redemption", branch)
		require.Equal(t, "redeeming-client", clientID, "the redemption handler still reads the body")
	})

	t.Run("CodeWithoutGrantType", func(t *testing.T) {
		branch, _ := route("/oauth/authorize", url.Values{"code": {primitive.NewObjectID().Hex()}, "client_id": {"redeeming-client"}})
		require.Equal(t, "redemption", branch)
	})

	t.Run("GrantTypeWithoutCode", func(t *testing.T) {
		branch, _ := route("/oauth/authorize", url.Values{"grant_type": {"authorization_code"}, "client_id": {"redeeming-client"}})
		require.Equal(t, "redemption", branch, "a malformed redemption is refused as JSON, not sent to sign in")
	})

	t.Run("CodeInTheQueryString", func(t *testing.T) {
		branch, _ := route("/oauth/authorize?grant_type=authorization_code&code=abc", consentForm)
		require.Equal(t, "consent", branch, "only the body selects the branch")
	})

	t.Run("EmptyBody", func(t *testing.T) {
		branch, clientID := route("/oauth/authorize", url.Values{})
		require.Equal(t, "consent", branch)
		require.Empty(t, clientID)
	})

	t.Run("MalformedBody", func(t *testing.T) {

		request := httptest.NewRequest(http.MethodPost, "/oauth/authorize", strings.NewReader("client_id=consent-client&code=%zz"))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		// The parse fails, so it is no redemption, and the consent handler binds the fields that survived
		require.False(t, isCodeRedemption(request))
		require.Equal(t, "consent-client", request.PostForm.Get("client_id"))
	})
}

// FuzzIsCodeRedemption requires that any body which fails to parse is consent's, and that any other body is a
// redemption exactly when it carries a grant_type or a code
func FuzzIsCodeRedemption(f *testing.F) {

	f.Add("client_id=consent-client&redirect_uri=https%3A%2F%2Fclient.example%2F&response_type=code")
	f.Add("grant_type=authorization_code&code=abc&client_id=x&code_verifier=y")
	f.Add("code=abc")
	f.Add("grant_type=")
	f.Add("code=%zz&grant_type=authorization_code")
	f.Add("code;grant_type=x")
	f.Add("")

	f.Fuzz(func(t *testing.T, body string) {

		request := httptest.NewRequest(http.MethodPost, "/oauth/authorize?code=from-the-query", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		// The expected answer comes from parsing the body alone, which is what ParseForm does with it
		values, err := url.ParseQuery(body)
		expected := (err == nil) && ((values.Get("grant_type") != "") || (values.Get("code") != ""))

		require.Equal(t, expected, isCodeRedemption(request), "body %q", body)
	})
}

// TestPostOAuthAuthorizationRedemption_AnswersClientMistakesAsJSON drives the refusals of a redemption at the
// authorization endpoint, which answers in the token endpoint's RFC 6749 §5.2 form (IndieAuth §5.3.2)
func TestPostOAuthAuthorizationRedemption_AnswersClientMistakesAsJSON(t *testing.T) {

	// A redemption as an IndieAuth client posts it
	form := func(client *model.OAuthClient, grant *model.OAuthUserToken) url.Values {
		return url.Values{
			"grant_type":    {"authorization_code"},
			"client_id":     {client.ClientID.Hex()},
			"code":          {grant.Code()},
			"redirect_uri":  {"https://client.example/callback"},
			"code_verifier": {oauthValidVerifier},
		}
	}

	// A URL client's grant, bound to PKCE and a redirect_uri, which passes every check the two endpoints share
	issued := func() (*model.OAuthClient, *model.OAuthUserToken) {
		client := newOAuthClient("")
		client.ClientURL = "https://client.example/"
		grant := newOAuthGrant(client)
		grant.SetRedirectURI("https://client.example/callback")
		grant.SetPKCEChallenge(oauthValidVerifier, model.PKCEMethodPlain)
		return client, grant
	}

	t.Run("MalformedBody", func(t *testing.T) {

		request := httptest.NewRequest(http.MethodPost, "/oauth/authorize", strings.NewReader("grant_type=authorization_code&code=%zz"))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		recorder := httptest.NewRecorder()
		ctx := &steranko.Context{Context: echo.New().NewContext(request, recorder)}

		require.NoError(t, PostOAuthAuthorizationRedemption(ctx, &service.Factory{}, oauthSession{}))
		require.Equal(t, http.StatusBadRequest, recorder.Code)
		require.JSONEq(t, `{"error":"invalid_request","error_description":"Invalid form parameters"}`, recorder.Body.String())
	})

	t.Run("UnknownClient", func(t *testing.T) {
		client, grant := issued()
		requireOAuthRefusalFrom(t, PostOAuthAuthorizationRedemption, oauthSession{}, form(client, grant), "invalid_client", "Invalid client_id")
	})

	t.Run("MissingGrantType", func(t *testing.T) {
		client, grant := issued()
		values := form(client, grant)
		values.Del("grant_type")
		requireOAuthRefusalFrom(t, PostOAuthAuthorizationRedemption, oauthSession{client: client, grant: grant}, values, "invalid_request", "Missing grant_type")
	})

	t.Run("UnsupportedGrantType", func(t *testing.T) {
		client, grant := issued()
		values := form(client, grant)
		values.Set("grant_type", "refresh_token")
		requireOAuthRefusalFrom(t, PostOAuthAuthorizationRedemption, oauthSession{client: client}, values, "unsupported_grant_type", "Unsupported grant_type")
	})

	// The shared checks are driven one by one through the token endpoint; these confirm this endpoint runs them
	t.Run("WrongVerifier", func(t *testing.T) {
		client, grant := issued()
		values := form(client, grant)
		values.Set("code_verifier", oauthCodeVerifier)
		requireOAuthRefusalFrom(t, PostOAuthAuthorizationRedemption, oauthSession{client: client, grant: grant}, values, "invalid_grant", "Invalid PKCE code_verifier")
	})

	t.Run("WrongRedirectURI", func(t *testing.T) {
		client, grant := issued()
		values := form(client, grant)
		values.Set("redirect_uri", "https://evil.example/callback")
		requireOAuthRefusalFrom(t, PostOAuthAuthorizationRedemption, oauthSession{client: client, grant: grant}, values, "invalid_grant", "Invalid redirect_uri")
	})

	t.Run("UnknownUser", func(t *testing.T) {
		client, grant := issued()
		requireOAuthRefusalFrom(t, PostOAuthAuthorizationRedemption, oauthSession{client: client, grant: grant}, form(client, grant), "invalid_grant", "Loading User")
	})

	t.Run("NoIdentity", func(t *testing.T) {
		client, grant := issued()
		user := model.NewUser()
		user.IsOwner = true
		user.IsPublic = true // public, but not the domain's primary user, so it has no `me`
		session := oauthSession{client: client, grant: grant, user: &user}

		// Refused before the code is spent: redeeming it would reach the database, which this session lacks
		requireOAuthRefusalFrom(t, PostOAuthAuthorizationRedemption, session, form(client, grant), "invalid_grant", "This server vouches for no identity to return")
	})
}

// TestPostOAuthAuthorizationRedemption_ReturnsServerFaults confirms that a failure of the server's own is
// returned for errorHandler to report, and carries no credential
func TestPostOAuthAuthorizationRedemption_ReturnsServerFaults(t *testing.T) {

	session := oauthSession{failure: derp.Internal("handler.test", "Database is down")}
	values := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {primitive.NewObjectID().Hex()},
		"client_secret": {oauthPresentedSecret},
		"code_verifier": {oauthCodeVerifier},
	}

	ctx, recorder := oauthTokenRequest(values)
	err := PostOAuthAuthorizationRedemption(ctx, &service.Factory{}, session)

	requireNoOAuthSecrets(t, err, "handler.PostOAuthAuthorizationRedemption", "Invalid client_id")
	require.True(t, derp.IsServerError(err))
	require.Empty(t, recorder.Body.String(), "nothing is written, so errorHandler can answer")
}

// TestPostOAuthAuthorizationRedemption redeems codes at the authorization endpoint inside a transaction, as every
// POST is, and requires `me` with the released profile, no token, and a spent code (IndieAuth §5.3.2)
func TestPostOAuthAuthorizationRedemption(t *testing.T) {

	server, session := newOAuthTestServer(t)

	client := newOAuthClient("")
	client.ClientURL = "https://client.example/"
	require.NoError(t, session.Collection("OAuthClient").Save(client, "test"))

	user := model.NewUser()
	user.IsOwner = true
	user.IsPrimary = true
	user.IsPublic = true
	user.DisplayName = "Primary User"
	user.EmailAddress = "primary@example.com"
	user.ProfileURL = "https://example.com/@" + user.UserID.Hex()
	require.NoError(t, session.Collection("User").Save(&user, "test"))

	// issue saves a PKCE-bound grant for the User with the given scopes, and returns the form that redeems it
	issue := func(scopes ...string) (*model.OAuthUserToken, url.Values) {

		grant := newOAuthGrant(client)
		grant.UserID = user.UserID
		grant.Scopes = scopes
		grant.SetRedirectURI("https://client.example/callback")
		grant.SetPKCEChallenge(oauthValidVerifier, model.PKCEMethodPlain)
		require.NoError(t, session.Collection("OAuthUserToken").Save(grant, "test"))

		return grant, url.Values{
			"grant_type":    {"authorization_code"},
			"client_id":     {client.ClientID.Hex()},
			"code":          {grant.Code()},
			"redirect_uri":  {"https://client.example/callback"},
			"code_verifier": {oauthValidVerifier},
		}
	}

	t.Run("ScopedCode", func(t *testing.T) {

		grant, values := issue("create", "profile", "email")

		// RULE: The answer is the identity and the profile its scopes release, and never a token
		recorder := postOAuthInTransaction(t, server, PostOAuthAuthorizationRedemption, values)
		require.Equal(t, http.StatusOK, recorder.Code)
		require.JSONEq(t, `{"me":"`+user.ProfileURL+`","profile":{"url":"`+user.ProfileURL+`","name":"Primary User","email":"primary@example.com"}}`, recorder.Body.String())

		// The stored code is spent, and the grant keeps nothing to refresh
		stored := model.NewOAuthUserToken()
		require.NoError(t, session.Collection("OAuthUserToken").Load(exp.Equal("_id", grant.OAuthUserTokenID), &stored))
		require.True(t, stored.CodeRedeemed)
		require.Empty(t, stored.RefreshHash)

		// RULE: The code is single-use, here and at the token endpoint
		recorder = postOAuthInTransaction(t, server, PostOAuthAuthorizationRedemption, values)
		require.Equal(t, http.StatusBadRequest, recorder.Code)
		require.JSONEq(t, `{"error":"invalid_grant","error_description":"Redeeming authorization code"}`, recorder.Body.String())

		recorder = postOAuthTokenInTransaction(t, server, values)
		require.Equal(t, http.StatusBadRequest, recorder.Code)
		require.JSONEq(t, `{"error":"invalid_grant","error_description":"Exchanging authorization code"}`, recorder.Body.String())
	})

	t.Run("MissingGrantType", func(t *testing.T) {

		grant, values := issue("")
		grantType := values.Get("grant_type")
		values.Del("grant_type")

		// RULE: grant_type is required, and a request without it leaves the code unspent
		recorder := postOAuthInTransaction(t, server, PostOAuthAuthorizationRedemption, values)
		require.Equal(t, http.StatusBadRequest, recorder.Code)
		require.JSONEq(t, `{"error":"invalid_request","error_description":"Missing grant_type"}`, recorder.Body.String())

		stored := model.NewOAuthUserToken()
		require.NoError(t, session.Collection("OAuthUserToken").Load(exp.Equal("_id", grant.OAuthUserTokenID), &stored))
		require.False(t, stored.CodeRedeemed)

		// The same code redeems once the request names its grant
		values.Set("grant_type", grantType)
		recorder = postOAuthInTransaction(t, server, PostOAuthAuthorizationRedemption, values)
		require.Equal(t, http.StatusOK, recorder.Code)
		require.JSONEq(t, `{"me":"`+user.ProfileURL+`"}`, recorder.Body.String())
	})
}

// oauthTokenRequest builds a urlencoded POST to the token endpoint, and the recorder that holds its response
func oauthTokenRequest(form url.Values) (*steranko.Context, *httptest.ResponseRecorder) {

	request := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	recorder := httptest.NewRecorder()
	return &steranko.Context{Context: echo.New().NewContext(request, recorder)}, recorder
}

// requireOAuthRefusal posts the form to the token endpoint and requires an RFC 6749 §5.2 answer with the given
// error and description, carrying no credential that the request presented
func requireOAuthRefusal(t *testing.T, session data.Session, form url.Values, errorCode string, description string) {
	t.Helper()
	requireOAuthRefusalFrom(t, PostOAuthToken, session, form, errorCode, description)
}

// requireOAuthRefusalFrom posts the form to the handler and requires an RFC 6749 §5.2 answer with the given
// error and description, carrying no credential that the request presented
func requireOAuthRefusalFrom(t *testing.T, handler WithFunc0, session data.Session, form url.Values, errorCode string, description string) {

	t.Helper()

	ctx, recorder := oauthTokenRequest(form)

	// A client's mistake is answered, never returned, so errorHandler does not report it
	require.NoError(t, handler(ctx, &service.Factory{}, session))
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Contains(t, recorder.Header().Get("Content-Type"), "application/json")

	answer := map[string]any{}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &answer))
	require.Equal(t, map[string]any{"error": errorCode, "error_description": description}, answer)

	for _, secret := range []string{oauthPresentedSecret, oauthStoredSecret, oauthCodeVerifier, oauthRefreshSecret, oauthValidVerifier} {
		require.NotContains(t, recorder.Body.String(), secret)
	}
}

// requireNoOAuthSecrets fails unless the error came from the expected site, carrying no
// credential that the request presented
func requireNoOAuthSecrets(t *testing.T, err error, location string, message string) {

	t.Helper()

	// Pinning the site proves the test reached the line it is named for
	require.Error(t, err)
	require.Equal(t, location, derp.Location(err))
	require.Equal(t, message, derp.Message(err))

	secretcheck.RequireAbsent(t, err, oauthPresentedSecret)
	secretcheck.RequireAbsent(t, err, oauthStoredSecret)
	secretcheck.RequireAbsent(t, err, oauthCodeVerifier)
	secretcheck.RequireAbsent(t, err, oauthRefreshSecret)
}

// newOAuthClient returns a client with the given stored secret; an empty one makes it public
func newOAuthClient(clientSecret string) *model.OAuthClient {
	client := model.NewOAuthClient()
	client.ClientSecret = clientSecret
	return &client
}

// newOAuthGrant returns a grant issued to the client, with no PKCE challenge
func newOAuthGrant(client *model.OAuthClient) *model.OAuthUserToken {
	grant := model.NewOAuthUserToken()
	grant.ClientID = client.ClientID
	return &grant
}

// oauthAuthorizationResponse returns a context for the consent form's POST, and the recorder that holds its response
func oauthAuthorizationResponse() (echo.Context, *httptest.ResponseRecorder) {
	request := httptest.NewRequest(http.MethodPost, "/oauth/authorize", nil)
	recorder := httptest.NewRecorder()
	return echo.New().NewContext(request, recorder), recorder
}

// postOAuthTokenInTransaction posts the form to the token endpoint inside a transaction, as
// WithFactory runs every POST, and returns the recorded response
func postOAuthTokenInTransaction(t *testing.T, server data.Server, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	return postOAuthInTransaction(t, server, PostOAuthToken, form)
}

// postOAuthInTransaction posts the form to the handler inside a transaction, as WithFactory runs every
// POST, and returns the recorded response
func postOAuthInTransaction(t *testing.T, server data.Server, handler WithFunc0, form url.Values) *httptest.ResponseRecorder {

	t.Helper()

	ctx, recorder := oauthTokenRequest(form)

	_, err := server.WithTransaction(context.Background(), func(session data.Session) (any, error) {
		return nil, handler(ctx, &service.Factory{}, session)
	})

	require.NoError(t, err)
	return recorder
}

// newOAuthTestServer returns a throwaway database on the local replica set, dropped when the test
// ends, and a session on it.  The test is skipped in -short mode, or when no replica set is reachable.
func newOAuthTestServer(t *testing.T) (data.Server, data.Session) {

	t.Helper()

	if testing.Short() {
		t.Skip("Skipping MongoDB integration test in -short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(oauthTestConnection))

	if err != nil {
		t.Skip("Skipping: no MongoDB at " + oauthTestConnection)
	}

	// RULE: Transactions need a replica set, and a standalone server names none
	hello := bson.M{}
	if err := client.Database("admin").RunCommand(ctx, bson.D{{Key: "hello", Value: 1}}).Decode(&hello); err != nil || hello["setName"] == nil {
		_ = client.Disconnect(ctx) // Nothing to report: the test is skipped either way
		t.Skip("Skipping: no MongoDB replica set at " + oauthTestConnection)
	}

	database := client.Database("emissary_oauthtest_" + primitive.NewObjectID().Hex())

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = database.Drop(ctx)     // A leftover test database is harmless
		_ = client.Disconnect(ctx) // ...and so is a connection closed by process exit
	})

	server := mongodb.NewServer(database)
	session, err := server.Session(context.Background())
	require.NoError(t, err)

	return server, session
}

// TestOAuthMetadata verifies the metadata document's endpoints, scopes, and client authentication methods
func TestOAuthMetadata(t *testing.T) {

	const host = "https://example.com"
	metadata := oauthMetadata(host)

	// Every endpoint lives under the issuer, which is the host itself
	require.Equal(t, host, metadata["issuer"])

	for _, name := range []string{"authorization_endpoint", "token_endpoint", "revocation_endpoint"} {
		require.True(t, strings.HasPrefix(metadata[name].(string), host+"/"), name)
	}

	// RULE: Advertise only the client authentication both endpoints actually read
	for _, name := range []string{"token_endpoint_auth_methods_supported", "revocation_endpoint_auth_methods_supported"} {
		require.Equal(t, []string{"client_secret_post", "none"}, metadata[name], name)
	}

	// Authorization redirects carry `iss` (RFC 9207)
	require.Equal(t, true, metadata["authorization_response_iss_parameter_supported"])

	// RULE: Nothing is advertised that does not exist yet (INDIEAUTH §3)
	require.NotContains(t, metadata, "introspection_endpoint")
	require.NotContains(t, metadata, "userinfo_endpoint")
}

// TestOAuthScopesSupported verifies that IndieAuth's scopes are advertised, and that the consent page describes each one
func TestOAuthScopesSupported(t *testing.T) {

	advertised := oauthScopesSupported()

	require.Contains(t, advertised, model.OAuthUserTokenScopeProfile)
	require.Contains(t, advertised, model.OAuthUserTokenScopeEmail)

	// Every advertised scope must have words on the consent page, rather than landing under "Other"
	described := map[string]bool{}

	for _, lookupCode := range dataset.OAuthScopes() {
		described[lookupCode.Value] = true
	}

	for _, value := range advertised {
		require.True(t, described[value], "the consent page does not describe %q", value)
	}

	// Each call returns its own list
	advertised[0] = "changed"
	require.Equal(t, model.OAuthUserTokenScopeProfile, oauthScopesSupported()[0])
}
