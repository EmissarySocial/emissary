package handler

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/cimd"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/steranko"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

// The IndieAuth client in these tests: identified by URL, publishing no metadata, redirecting to its own origin
const (
	indieAuthClientID     = "https://client.example.com/"
	indieAuthRedirectURI  = "https://client.example.com/callback"
	indieAuthState        = "state-from-the-client"
	indieAuthSessionKeyID = "test"
	indieAuthSessionKey   = "indieauth-flow-test-session-key-indieauth-flow-test-session-key"
)

// indieAuthSessionKeys signs and verifies the test's session tokens with one fixed HMAC secret
type indieAuthSessionKeys struct{}

// GetCurrentKey returns the fixed signing key
func (indieAuthSessionKeys) GetCurrentKey() (string, any, error) {
	return indieAuthSessionKeyID, []byte(indieAuthSessionKey), nil
}

// FindKey returns the fixed signing key, whatever the token names
func (indieAuthSessionKeys) FindKey(*jwt.Token) (any, error) {
	return []byte(indieAuthSessionKey), nil
}

// indieAuthSessionUsers parses session tokens into an Authorization, which is all these tests need of steranko
type indieAuthSessionUsers struct{}

// New is unused by these tests
func (indieAuthSessionUsers) New() steranko.User { return nil }

// Load is unused by these tests
func (indieAuthSessionUsers) Load(string, steranko.User) error {
	return derp.NotFound("test", "unused")
}

// LoadBySubject is unused by these tests
func (indieAuthSessionUsers) LoadBySubject(string, steranko.User) error {
	return derp.NotFound("test", "unused")
}

// Save is unused by these tests
func (indieAuthSessionUsers) Save(steranko.User, string) error {
	return derp.Internal("test", "unused")
}

// Delete is unused by these tests
func (indieAuthSessionUsers) Delete(steranko.User, string) error {
	return derp.Internal("test", "unused")
}

// RequestPasswordReset is unused by these tests
func (indieAuthSessionUsers) RequestPasswordReset(steranko.User) error {
	return derp.Internal("test", "unused")
}

// Claims is unused by these tests
func (indieAuthSessionUsers) Claims(steranko.User) (jwt.Claims, error) {
	return nil, derp.Internal("test", "unused")
}

// Close holds nothing to release
func (indieAuthSessionUsers) Close() {}

// NewClaims returns an empty Authorization for a session token to parse into
func (indieAuthSessionUsers) NewClaims() jwt.Claims {
	authorization := model.NewAuthorization()
	return &authorization
}

// indieAuthFlow is one domain in a throwaway database, with the URL client already known to it
type indieAuthFlow struct {
	server   data.Server
	session  data.Session
	factory  *service.Factory
	client   model.OAuthClient
	steranko *steranko.Steranko
}

// newIndieAuthFlow stores the URL client the way the consent page leaves one that publishes no metadata
func newIndieAuthFlow(t *testing.T) indieAuthFlow {

	t.Helper()
	server, session := newOAuthTestServer(t)

	clientURL, err := cimd.ParseClientID(indieAuthClientID)
	require.NoError(t, err)

	client := model.NewOAuthClient()
	client.ClientURL = clientURL
	client.Name = "client.example.com"
	require.NoError(t, session.Collection("OAuthClient").Save(&client, "test"))

	return indieAuthFlow{
		server:   server,
		session:  session,
		factory:  &service.Factory{},
		client:   client,
		steranko: steranko.New(indieAuthSessionUsers{}, indieAuthSessionKeys{}),
	}
}

// storeUser saves a public owner, flagged primary or not
func (flow indieAuthFlow) storeUser(t *testing.T, primary bool) model.User {

	t.Helper()

	user := model.NewUser()
	user.DisplayName = "Test User"
	user.EmailAddress = "test@example.com"
	user.IsOwner = true
	user.IsPrimary = primary
	user.IsPublic = true
	user.ProfileURL = "https://example.com/@" + user.UserID.Hex()
	require.NoError(t, flow.session.Collection("User").Save(&user, "test"))

	return user
}

// approve posts the consent form as the signed-in User, and returns the query of the redirect back to the client
func (flow indieAuthFlow) approve(t *testing.T, user model.User, scope string) url.Values {

	t.Helper()
	return flow.submit(t, user, scope, nil)
}

// submit posts the consent form as the signed-in User, with any extra values the pressed button adds,
// and returns the query of the redirect back to the client
func (flow indieAuthFlow) submit(t *testing.T, user model.User, scope string, extra url.Values) url.Values {

	t.Helper()

	// Sign the User in, as WithAuthenticatedUser requires
	authorization := model.NewAuthorization()
	authorization.UserID = user.UserID
	authorization.Revalidate = time.Now().Unix()
	token, err := flow.steranko.CreateJWT(&authorization)
	require.NoError(t, err)

	// Post every value the consent page posts back
	form := url.Values{
		"response_type":         {"code"},
		"client_id":             {flow.client.ClientID.Hex()},
		"redirect_uri":          {indieAuthRedirectURI},
		"state":                 {indieAuthState},
		"scope":                 {scope},
		"code_challenge":        {pkceS256(oauthValidVerifier)},
		"code_challenge_method": {model.PKCEMethodS256},
	}

	for name, values := range extra {
		form[name] = values
	}

	request := httptest.NewRequest(http.MethodPost, "/oauth/authorize", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	ctx := flow.steranko.Context(echo.New().NewContext(request, recorder))

	// Every POST runs in a transaction
	_, err = flow.server.WithTransaction(context.Background(), func(session data.Session) (any, error) {
		return nil, PostOAuthAuthorization(ctx, flow.factory, session, &user)
	})
	require.NoError(t, err)

	// The User is sent back to the client's own callback
	require.Equal(t, http.StatusFound, recorder.Code)
	target, rawQuery, found := strings.Cut(recorder.Header().Get("Location"), "?")
	require.True(t, found, "the redirect carries no query")
	require.Equal(t, indieAuthRedirectURI, target)

	query, err := url.ParseQuery(rawQuery)
	require.NoError(t, err)

	return query
}

// redemption returns the form a client posts to redeem a code, with any values replaced
func redemption(code string, replace url.Values) url.Values {

	result := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"client_id":     {indieAuthClientID},
		"redirect_uri":  {indieAuthRedirectURI},
		"code_verifier": {oauthValidVerifier},
	}

	for name, values := range replace {
		result[name] = values
	}

	return result
}

// requireOAuthError requires an RFC 6749 §5.2 answer naming the error
func requireOAuthError(t *testing.T, recorder *httptest.ResponseRecorder, errorCode string) {

	t.Helper()
	require.Equal(t, http.StatusBadRequest, recorder.Code, recorder.Body.String())
	require.Contains(t, recorder.Body.String(), `"error":"`+errorCode+`"`)
}

// pkceS256 returns the S256 code challenge for a verifier (RFC 7636 §4.2)
func pkceS256(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// TestIndieAuthFlow_Discovery follows the Link header to the metadata document, and requires the two to agree
func TestIndieAuthFlow_Discovery(t *testing.T) {

	flow := newIndieAuthFlow(t)
	host := flow.factory.Host()

	// A page from a domain with a primary user names the metadata document and both endpoints
	header := http.Header{}
	advertiseIndieAuth(header, host, true)
	links := parseLinkHeader(t, header.Get("Link"))

	// The metadata document is served by the handler itself
	request := httptest.NewRequest(http.MethodGet, "/.well-known/oauth-authorization-server", nil)
	recorder := httptest.NewRecorder()
	require.NoError(t, GetOAuthWellKnown(&steranko.Context{Context: echo.New().NewContext(request, recorder)}, flow.factory, flow.session))
	require.Equal(t, http.StatusOK, recorder.Code)

	metadata := map[string]any{}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &metadata))

	// RULE: The client finds the same endpoints by either route, under the issuer that redirects name
	require.Equal(t, metadata["authorization_endpoint"], links["authorization_endpoint"])
	require.Equal(t, metadata["token_endpoint"], links["token_endpoint"])
	require.Equal(t, host, metadata["issuer"])
	require.Contains(t, metadata["scopes_supported"], model.OAuthUserTokenScopeProfile)
	require.Contains(t, metadata["code_challenge_methods_supported"], model.PKCEMethodS256)
}

// TestIndieAuthFlow_SignIn approves an identity-only request and redeems the code at the authorization endpoint
func TestIndieAuthFlow_SignIn(t *testing.T) {

	flow := newIndieAuthFlow(t)
	user := flow.storeUser(t, true)

	// The redirect carries the code, the client's own state, and the issuer (RFC 9207)
	query := flow.approve(t, user, "")
	require.NotEmpty(t, query.Get("code"))
	require.Equal(t, indieAuthState, query.Get("state"))
	require.Equal(t, []string{flow.factory.Host()}, query["iss"])

	// The client redeems it for the User's profile URL, and nothing else
	recorder := postOAuthInTransaction(t, flow.server, PostOAuthAuthorizationRedemption, redemption(query.Get("code"), nil))
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.JSONEq(t, `{"me":"`+user.ProfileURL+`"}`, recorder.Body.String())

	// RULE: The code is single-use, at either endpoint
	requireOAuthError(t, postOAuthInTransaction(t, flow.server, PostOAuthAuthorizationRedemption, redemption(query.Get("code"), nil)), "invalid_grant")
	requireOAuthError(t, postOAuthTokenInTransaction(t, flow.server, redemption(query.Get("code"), nil)), "invalid_grant")
}

// TestIndieAuthFlow_SignInAtTheTokenEndpoint redeems an identity-only code where a client that asked for tokens would
func TestIndieAuthFlow_SignInAtTheTokenEndpoint(t *testing.T) {

	flow := newIndieAuthFlow(t)
	user := flow.storeUser(t, true)
	query := flow.approve(t, user, "")

	// IndieAuth §5.3.3: an identity-only code gets `me` and no access token
	recorder := postOAuthTokenInTransaction(t, flow.server, redemption(query.Get("code"), nil))
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.JSONEq(t, `{"me":"`+user.ProfileURL+`"}`, recorder.Body.String())

	requireOAuthError(t, postOAuthInTransaction(t, flow.server, PostOAuthAuthorizationRedemption, redemption(query.Get("code"), nil)), "invalid_grant")
}

// TestIndieAuthFlow_Profile approves the profile and email scopes, and requires them in the identity response
func TestIndieAuthFlow_Profile(t *testing.T) {

	flow := newIndieAuthFlow(t)
	user := flow.storeUser(t, true)
	query := flow.approve(t, user, "profile email")

	recorder := postOAuthInTransaction(t, flow.server, PostOAuthAuthorizationRedemption, redemption(query.Get("code"), nil))
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())

	response := map[string]any{}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, user.ProfileURL, response["me"])
	require.Equal(t, map[string]any{
		"url":   user.ProfileURL,
		"name":  user.DisplayName,
		"email": user.EmailAddress,
	}, response["profile"])

	// No token is minted at the authorization endpoint, whatever the scope
	require.NotContains(t, response, "access_token")
}

// TestIndieAuthFlow_RefusalsLeaveTheCodeUnspent sends each mistake a client can make, then redeems the same code
func TestIndieAuthFlow_RefusalsLeaveTheCodeUnspent(t *testing.T) {

	flow := newIndieAuthFlow(t)
	user := flow.storeUser(t, true)
	code := flow.approve(t, user, "").Get("code")

	for _, endpoint := range []WithFunc0{PostOAuthAuthorizationRedemption, PostOAuthToken} {

		requireOAuthError(t, postOAuthInTransaction(t, flow.server, endpoint, redemption(code, url.Values{"redirect_uri": {"https://client.example.com/elsewhere"}})), "invalid_grant")
		requireOAuthError(t, postOAuthInTransaction(t, flow.server, endpoint, redemption(code, url.Values{"code_verifier": {oauthValidVerifier + "-wrong"}})), "invalid_grant")
		requireOAuthError(t, postOAuthInTransaction(t, flow.server, endpoint, redemption(code, url.Values{"code_verifier": {""}})), "invalid_grant")
		requireOAuthError(t, postOAuthInTransaction(t, flow.server, endpoint, redemption(code, url.Values{"grant_type": {""}})), "invalid_request")
		requireOAuthError(t, postOAuthInTransaction(t, flow.server, endpoint, redemption(code, url.Values{"client_id": {"https://other.example.com/"}})), "invalid_client")
	}

	// RULE: None of them spent the code
	recorder := postOAuthInTransaction(t, flow.server, PostOAuthAuthorizationRedemption, redemption(code, nil))
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
}

// TestIndieAuthFlow_NotThePrimaryUser requires that a User the domain does not vouch for gets no `me` anywhere
func TestIndieAuthFlow_NotThePrimaryUser(t *testing.T) {

	flow := newIndieAuthFlow(t)
	flow.storeUser(t, true)
	member := flow.storeUser(t, false)

	// The consent form still works, because the grant belongs to whoever is signed in
	identityOnly := flow.approve(t, member, "").Get("code")
	withProfile := flow.approve(t, member, "profile").Get("code")

	requireOAuthError(t, postOAuthInTransaction(t, flow.server, PostOAuthAuthorizationRedemption, redemption(identityOnly, nil)), "invalid_grant")
	requireOAuthError(t, postOAuthTokenInTransaction(t, flow.server, redemption(identityOnly, nil)), "invalid_grant")
	requireOAuthError(t, postOAuthInTransaction(t, flow.server, PostOAuthAuthorizationRedemption, redemption(withProfile, nil)), "invalid_grant")
}

// TestIndieAuthFlow_TwoServersRedeemOneCode redeems one code from two servers at once, sharing only the database
func TestIndieAuthFlow_TwoServersRedeemOneCode(t *testing.T) {

	flow := newIndieAuthFlow(t)
	user := flow.storeUser(t, true)

	for round := range 5 {

		code := flow.approve(t, user, "").Get("code")
		start := make(chan struct{})
		statuses := make(chan int, 2)

		// Each server builds its own request and factory, and runs its own transaction
		for _, endpoint := range []WithFunc0{PostOAuthAuthorizationRedemption, PostOAuthToken} {
			go func() {
				ctx, recorder := oauthTokenRequest(redemption(code, nil))
				<-start
				if _, err := flow.server.WithTransaction(context.Background(), func(session data.Session) (any, error) {
					return nil, endpoint(ctx, &service.Factory{}, session)
				}); err != nil {
					statuses <- http.StatusInternalServerError
					return
				}

				statuses <- recorder.Code
			}()
		}

		close(start)
		results := []int{<-statuses, <-statuses}

		// RULE: Exactly one server redeems the code
		require.ElementsMatch(t, []int{http.StatusOK, http.StatusBadRequest}, results, "round %d", round)
	}
}

// TestIndieAuthFlow_Cancel verifies that a User who refuses is sent back with access_denied, and that nothing is granted
func TestIndieAuthFlow_Cancel(t *testing.T) {

	flow := newIndieAuthFlow(t)
	user := flow.storeUser(t, true)

	query := flow.submit(t, user, "profile", url.Values{"decision": {"deny"}})

	require.Equal(t, "access_denied", query.Get("error"))
	require.Equal(t, indieAuthState, query.Get("state"))
	require.Equal(t, []string{flow.factory.Host()}, query["iss"])
	require.Empty(t, query.Get("code"))

	// RULE: A refusal writes no grant
	count, err := flow.session.Collection("OAuthUserToken").Count(exp.All())
	require.NoError(t, err)
	require.Zero(t, count)
}

// TestOAuthDenialURL verifies where a refusal is written for each kind of request
func TestOAuthDenialURL(t *testing.T) {

	const issuer = "https://example.com"

	t.Run("a code request is answered in the query, keeping the redirect_uri's own", func(t *testing.T) {
		request := model.OAuthAuthorizationRequest{ResponseType: "code", RedirectURI: "https://app.example.com/callback?tenant=7", State: "abc"}
		result, err := url.Parse(oauthDenialURL(request, issuer))
		require.NoError(t, err)
		require.Equal(t, "app.example.com", result.Host)
		require.Equal(t, url.Values{"tenant": {"7"}, "error": {"access_denied"}, "state": {"abc"}, "iss": {issuer}}, result.Query())
		require.Empty(t, result.Fragment)
	})

	t.Run("a token request is answered in the fragment", func(t *testing.T) {
		request := model.OAuthAuthorizationRequest{ResponseType: "token", RedirectURI: "https://app.example.com/callback", State: "abc"}
		target, fragment, found := strings.Cut(oauthDenialURL(request, issuer), "#")
		require.True(t, found)
		require.Equal(t, "https://app.example.com/callback", target)

		values, err := url.ParseQuery(fragment)
		require.NoError(t, err)
		require.Equal(t, url.Values{"error": {"access_denied"}, "state": {"abc"}, "iss": {issuer}}, values)
	})

	t.Run("a state cannot add parameters of its own", func(t *testing.T) {
		for _, responseType := range []string{"code", "token"} {
			request := model.OAuthAuthorizationRequest{ResponseType: responseType, RedirectURI: "https://app.example.com/callback", State: "x&error=none&iss=https://evil.example.com#y"}
			result := oauthDenialURL(request, issuer)
			_, answer, _ := strings.Cut(result, map[string]string{"code": "?", "token": "#"}[responseType])

			values, err := url.ParseQuery(answer)
			require.NoError(t, err, result)
			require.Equal(t, []string{"access_denied"}, values["error"])
			require.Equal(t, []string{issuer}, values["iss"])
			require.Equal(t, request.State, values.Get("state"))
		}
	})

	t.Run("a registered app's own scheme is answered too", func(t *testing.T) {
		request := model.OAuthAuthorizationRequest{ResponseType: "code", RedirectURI: "icecubesapp://oauth", State: "abc"}
		require.Equal(t, "icecubesapp://oauth?error=access_denied&iss=https%3A%2F%2Fexample.com&state=abc", oauthDenialURL(request, issuer))
	})
}
