package handler

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/secretcheck"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/steranko"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// The credentials a token request presents.  None of them may appear in a reported error.
const (
	oauthPresentedSecret   = "presented-client-s3cr3t"
	oauthStoredSecret      = "stored-client-s3cr3t"
	oauthCodeVerifier      = "code-verifier-s3cr3t"
	oauthRefreshSecret     = "refresh-token-s3cr3t"
	oauthRefreshGeneration = 1
)

// oauthTokenRequest builds a urlencoded POST to the token endpoint
func oauthTokenRequest(form url.Values) *steranko.Context {

	request := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	recorder := httptest.NewRecorder()
	return &steranko.Context{Context: echo.New().NewContext(request, recorder)}
}

// oauthSession is a data.Session that serves one fixed record per collection, whatever the
// criteria.  These tests pin error details, not queries, and mockdb cannot match deleteDate.
type oauthSession struct {
	data.Session
	client *model.OAuthClient
	grant  *model.OAuthUserToken
}

// Collection returns the fixed record for this collection, if any
func (session oauthSession) Collection(name string) data.Collection {

	switch name {

	case "OAuthClient":
		return oauthCollection{record: session.client}

	case "OAuthUserToken":
		return oauthCollection{record: session.grant}
	}

	return oauthCollection{}
}

// oauthCollection is a data.Collection whose Load returns its one record, or NotFound
type oauthCollection struct {
	data.Collection
	record any
}

// Load copies the fixed record into target, or fails with NotFound when there is none
func (collection oauthCollection) Load(_ exp.Expression, target data.Object, _ ...option.Option) error {

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
	}

	return derp.NotFound("handler.oauthCollection.Load", "Record not found")
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

// TestPostOAuthToken_OmitsCredentialsFromErrors drives every failure of the token endpoint,
// and requires that none reports a credential from the request.
func TestPostOAuthToken_OmitsCredentialsFromErrors(t *testing.T) {

	// BUG-173: these six sites attached the whole OAuthUserTokenRequest, which has no json:"-"
	// field, so the secret, code, verifier, and refresh token were printed and stored.
	factory := &service.Factory{}

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

	t.Run("UnknownClient", func(t *testing.T) {
		session := oauthSession{}
		err := PostOAuthToken(oauthTokenRequest(form(primitive.NewObjectID().Hex(), "authorization_code", "")), factory, session)
		requireNoOAuthSecrets(t, err, "handler.PostOAuthToken", "Invalid client_id")
	})

	t.Run("MalformedCode", func(t *testing.T) {
		client := newOAuthClient(oauthStoredSecret)
		session := oauthSession{client: client}
		err := PostOAuthToken(oauthTokenRequest(form(client.ClientID.Hex(), "authorization_code", "not-an-objectid")), factory, session)
		requireNoOAuthSecrets(t, err, "handler.postOAuthToken_authorizationCode", "Invalid code")
	})

	t.Run("UnknownCode", func(t *testing.T) {
		client := newOAuthClient(oauthStoredSecret)
		session := oauthSession{client: client}
		err := PostOAuthToken(oauthTokenRequest(form(client.ClientID.Hex(), "authorization_code", primitive.NewObjectID().Hex())), factory, session)
		requireNoOAuthSecrets(t, err, "handler.postOAuthToken_authorizationCode", "Loading OAuthUserToken")
	})

	t.Run("WrongSecretForCode", func(t *testing.T) {
		client := newOAuthClient(oauthStoredSecret)
		grant := newOAuthGrant(client)
		session := oauthSession{client: client, grant: grant}
		err := PostOAuthToken(oauthTokenRequest(form(client.ClientID.Hex(), "authorization_code", grant.OAuthUserTokenID.Hex())), factory, session)
		requireNoOAuthSecrets(t, err, "handler.postOAuthToken_authorizationCode", "Invalid client_secret")
	})

	t.Run("PublicClientWithoutPKCE", func(t *testing.T) {
		client := newOAuthClient("")
		grant := newOAuthGrant(client)
		session := oauthSession{client: client, grant: grant}
		err := PostOAuthToken(oauthTokenRequest(form(client.ClientID.Hex(), "authorization_code", grant.OAuthUserTokenID.Hex())), factory, session)
		requireNoOAuthSecrets(t, err, "handler.postOAuthToken_authorizationCode", "This client must use PKCE (a code_verifier is required)")
	})

	t.Run("WrongSecretForRefresh", func(t *testing.T) {
		client := newOAuthClient(oauthStoredSecret)
		session := oauthSession{client: client}
		err := PostOAuthToken(oauthTokenRequest(form(client.ClientID.Hex(), "refresh_token", "")), factory, session)
		requireNoOAuthSecrets(t, err, "handler.postOAuthToken_refresh", "Invalid client_secret")
	})
}
