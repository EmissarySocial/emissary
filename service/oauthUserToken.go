package service

import (
	"strings"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/rosetta/sliceof"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// OAuthUserToken manages all interactions with the OAuthUserToken collection
type OAuthUserToken struct {
	oauthClientService *OAuthClient
	jwtService         *JWT
	host               string
}

// NewOAuthUserToken returns a fully populated OAuthUserToken service.
func NewOAuthUserToken() OAuthUserToken {
	return OAuthUserToken{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *OAuthUserToken) Refresh(factory *Factory) {
	service.oauthClientService = factory.OAuthClient()
	service.jwtService = factory.JWT()
	service.host = factory.Host()
}

// Close stops any background processes controlled by this service
func (service *OAuthUserToken) Close() {

}

/******************************************
 * Common Data Methods
 ******************************************/

// collection returns the OAuthUserToken collection for the provided database session
func (service *OAuthUserToken) collection(session data.Session) data.Collection {
	return session.Collection("OAuthUserToken")
}

// Count returns the number of records that match the provided criteria
func (service *OAuthUserToken) Count(session data.Session, criteria exp.Expression) (int64, error) {
	return service.collection(session).Count(notDeleted(criteria))
}

// Query returns an slice containing all of the OAuthUserTokens that match the provided criteria
func (service *OAuthUserToken) Query(session data.Session, criteria exp.Expression, options ...option.Option) ([]model.OAuthUserToken, error) {
	result := make([]model.OAuthUserToken, 0)
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)
	return result, err
}

// Iterator returns an iterator containing all of the OAuthUserTokens that match the provided criteria
func (service *OAuthUserToken) Iterator(session data.Session, criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection(session).Iterator(notDeleted(criteria), options...)
}

// Load retrieves an OAuthUserToken from the database
func (service *OAuthUserToken) Load(session data.Session, criteria exp.Expression, application *model.OAuthUserToken) error {

	if err := service.collection(session).Load(notDeleted(criteria), application); err != nil {
		return derp.Wrap(err, "service.OAuthUserToken", "Loading OAuthUserToken", criteria)
	}

	return nil
}

// Save adds/updates an OAuthUserToken in the database
func (service *OAuthUserToken) Save(session data.Session, application *model.OAuthUserToken, note string) error {

	const location = "service.OAuthUserToken"

	// Validate the value (using the global application schema) before saving
	if _, err := service.Schema().Validate(application); err != nil {
		return derp.Wrap(err, location, "Validating OAuthUserToken using OAuthUserTokenSchema", application)
	}

	// Try to save the OAuthUserToken to the database
	if err := service.collection(session).Save(application, note); err != nil {
		return derp.Wrap(err, location, "Saving OAuthUserToken", application, note)
	}

	return nil
}

// Delete removes an OAuthUserToken from the database (virtual delete)
func (service *OAuthUserToken) Delete(session data.Session, application *model.OAuthUserToken, note string) error {

	// Delete this OAuthUserToken
	if err := service.collection(session).Delete(application, note); err != nil {
		return derp.Wrap(err, "service.OAuthUserToken.Delete", "Deleting OAuthUserToken", application, note)
	}

	// Bueno!!
	return nil
}

// DeleteMany removes all child streams from the provided stream (virtual delete)
func (service *OAuthUserToken) DeleteMany(session data.Session, criteria exp.Expression, note string) error {

	const location = "service.OAuthUserToken.DeleteMany"

	// Get an iterator of all OAuthUserTokens that match the criteria
	it, err := service.Iterator(session, criteria)

	if err != nil {
		return derp.Wrap(err, location, "Listing streams to delete", criteria)
	}

	// Loop over each OAuthUserToken and delete it
	for userToken := model.NewOAuthUserToken(); it.Next(&userToken); userToken = model.NewOAuthUserToken() {
		if err := service.Delete(session, &userToken, note); err != nil {
			return derp.Wrap(err, location, "Deleting stream", userToken)
		}
	}

	return nil
}

/******************************************
 * Model Service Methods
 ******************************************/

// ObjectType returns the type of object that this service manages
func (service *OAuthUserToken) ObjectType() string {
	return "OAuthUserToken"
}

// New returns a fully initialized model.OAuthUserToken as a data.Object.
func (service *OAuthUserToken) ObjectNew() data.Object {
	result := model.NewOAuthUserToken()
	return &result
}

// ObjectID returns the unique ID of the provided OAuthUserToken. Implements the ModelService interface.
func (service *OAuthUserToken) ObjectID(object data.Object) primitive.ObjectID {

	if folder, ok := object.(*model.OAuthUserToken); ok {
		return folder.OAuthUserTokenID
	}

	return primitive.NilObjectID
}

// ObjectQuery returns every OAuthUserToken that matches the provided criteria. Implements the ModelService interface.
func (service *OAuthUserToken) ObjectQuery(session data.Session, result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection(session).Query(result, notDeleted(criteria), options...)
}

// ObjectLoad retrieves a single OAuthUserToken as a data.Object. Implements the ModelService interface.
func (service *OAuthUserToken) ObjectLoad(session data.Session, criteria exp.Expression) (data.Object, error) {
	result := model.NewOAuthUserToken()
	err := service.Load(session, criteria, &result)
	return &result, err
}

// ObjectSave adds or updates a OAuthUserToken in the database. Implements the ModelService interface.
func (service *OAuthUserToken) ObjectSave(session data.Session, object data.Object, comment string) error {
	if folder, ok := object.(*model.OAuthUserToken); ok {
		return service.Save(session, folder, comment)
	}
	return derp.Internal("service.OAuthUserToken.ObjectSave", "Invalid object type", object)
}

// ObjectDelete marks a OAuthUserToken as deleted. Implements the ModelService interface.
func (service *OAuthUserToken) ObjectDelete(session data.Session, object data.Object, comment string) error {
	if folder, ok := object.(*model.OAuthUserToken); ok {
		return service.Delete(session, folder, comment)
	}
	return derp.Internal("service.OAuthUserToken.ObjectDelete", "Invalid object type", object)
}

// ObjectUserCan reports whether the provided Authorization may run an action on a OAuthUserToken. Implements the ModelService interface.
func (service *OAuthUserToken) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.Unauthorized("service.OAuthUserToken", "Not Authorized")
}

// Schema returns the rosetta schema that describes a OAuthUserToken
func (service *OAuthUserToken) Schema() schema.Schema {
	return schema.New(model.OAuthUserTokenSchema())
}

/******************************************
 * Custom Queries
 ******************************************/

// LoadByID retrieves an OAuthUserToken using its ID and the ID of the User who owns it
func (service *OAuthUserToken) LoadByID(session data.Session, userID primitive.ObjectID, oauthUserTokenID primitive.ObjectID, result *model.OAuthUserToken) error {

	criteria := exp.Equal("userId", userID).
		AndEqual("_id", oauthUserTokenID)

	return service.Load(session, criteria, result)
}

// LoadByUserAndScope returns all OAUthUserTokens for the provided UserID that match the provided scope
func (service *OAuthUserToken) LoadByUserAndScope(session data.Session, userID primitive.ObjectID, scope string, result *model.OAuthUserToken) error {

	criteria := exp.Equal("userId", userID).
		AndIn("scopes", scope)

	return service.Load(session, criteria, result)
}

// LoadByClientAndID loads the grant identified by its ObjectID for the given
// client.  The ObjectID is both the authorization code (code grant) and the grant
// ID embedded in a refresh token (refresh grant).  It performs NO authentication:
// the caller MUST authenticate the redemption (a matching client_secret for a
// confidential client, or PKCE / the refresh secret for a public client) before
// issuing a token.
func (service *OAuthUserToken) LoadByClientAndID(session data.Session, userTokenID primitive.ObjectID, clientID primitive.ObjectID, result *model.OAuthUserToken) error {

	criteria := exp.Equal("_id", userTokenID).
		AndEqual("clientId", clientID)

	return service.Load(session, criteria, result)
}

/******************************************
 * Custom Methods
 ******************************************/

// CreateFromUser issues a new OAuth token that grants the provided client access to a User's account
func (service *OAuthUserToken) CreateFromUser(session data.Session, user *model.User, clientID primitive.ObjectID, scope string) (model.OAuthUserToken, error) {

	const location = "service.OAuthUserToken.CreateFromUser"

	// Load the client from the database
	client := model.NewOAuthClient()
	if err := service.oauthClientService.LoadByClientID(session, clientID, &client); err != nil {
		return model.OAuthUserToken{}, derp.Wrap(err, location, "Loading client", clientID)
	}

	// Build a fresh grant directly. Unlike the browser implicit grant, this
	// server-initiated path (Mastodon account creation) DOES issue a refresh token,
	// so the created account's token can be renewed rather than dying at expiry.
	grant := model.NewOAuthUserToken()
	grant.ClientID = client.ClientID
	grant.UserID = user.UserID
	grant.Scopes = scopesFromString(scope, client.Scopes)
	grant.APIUser = true
	grant.CodeRedeemed = true // there is no authorization code in this path

	// Mint the access + initial refresh token.
	if err := service.mintInitialTokens(&grant, time.Now()); err != nil {
		return model.OAuthUserToken{}, derp.Wrap(err, location, "Minting tokens")
	}

	if err := service.Save(session, &grant, "Create from user"); err != nil {
		return model.OAuthUserToken{}, derp.Wrap(err, location, "Saving grant")
	}

	return grant, nil
}

// scopesFromString parses a space/comma-delimited scope string, falling back to the
// client's full scope set when the request specifies none.
func scopesFromString(scope string, fallback sliceof.String) sliceof.String {

	fields := strings.Fields(strings.ReplaceAll(scope, ",", " "))

	if len(fields) == 0 {
		return fallback
	}

	return fields
}

// Create creates a new grant (OAuthUserToken) for the provided application and
// authorization. Every authorization mints a FRESH grant — there is no reuse of a
// prior grant (that shortcut handed back stale/expired tokens and ignored scope
// changes).
//
// For the authorization-code grant ("code"), Create only records the pending
// grant; the access and refresh tokens are minted later, at code exchange (see
// ExchangeCode). For the implicit grant ("token") — which has no exchange step —
// Create mints the access token now. The implicit grant does NOT issue a refresh
// token (RFC 6749 §4.2).
func (service *OAuthUserToken) Create(session data.Session, client model.OAuthClient, authorization model.Authorization, transaction model.OAuthAuthorizationRequest) (model.OAuthUserToken, error) {

	const location = "service.OAuthUserToken.Create"

	// Require that the user is actualy logged in
	if !authorization.IsAuthenticated() {
		return model.OAuthUserToken{}, derp.Unauthorized(location, "User is not logged in")
	}

	// Validate the request
	if err := transaction.Validate(client); err != nil {
		return model.OAuthUserToken{}, derp.Wrap(err, location, "Invalid OAuthUserTokenRequest")
	}

	// Build a fresh grant from the authorization
	result := model.NewOAuthUserToken()
	result.ClientID = client.ClientID
	result.UserID = authorization.UserID
	result.Scopes = transaction.Scopes()
	result.APIUser = true

	// Bind the PKCE challenge (if any) so the code can only be redeemed with the
	// matching verifier. No-op when the client did not use PKCE.
	result.SetPKCEChallenge(transaction.CodeChallenge, transaction.CodeChallengeMethod)

	// RULE: The implicit ("token") grant returns its access token directly from the
	// authorization endpoint, so it is minted now. The code grant defers minting to
	// ExchangeCode.
	if transaction.ResponseType == "token" {
		token, err := service.JWT(result.OAuthUserTokenID, result.UserID, strings.Join(result.Scopes, " "))

		if err != nil {
			return model.OAuthUserToken{}, derp.Wrap(err, location, "Generating access token")
		}

		result.Token = token
	}

	// Save the grant to the database
	if err := service.Save(session, &result, "Create"); err != nil {
		return model.OAuthUserToken{}, derp.Wrap(err, location, "Saving OAuthUserToken", result)
	}

	return result, nil
}

// ExchangeCode completes an authorization-code grant (RFC 6749 §4.1.3): it
// verifies the code is still redeemable (single-use and short-lived), then mints
// the access token and the INITIAL refresh token, marks the code consumed, and
// saves. The caller (the token handler) has already authenticated the client and
// verified PKCE.
func (service *OAuthUserToken) ExchangeCode(session data.Session, grant *model.OAuthUserToken) error {

	const location = "service.OAuthUserToken.ExchangeCode"

	now := time.Now()

	// RULE: An authorization code is single-use (RFC 6749 §4.1.2).
	if grant.CodeRedeemed {
		return derp.BadRequest(location, "Authorization code has already been used")
	}

	// RULE: An authorization code is short-lived.
	if grant.IsCodeExpired(now) {
		return derp.BadRequest(location, "Authorization code has expired")
	}

	// Mint the access token + initial refresh token for this grant.
	if err := service.mintInitialTokens(grant, now); err != nil {
		return derp.Wrap(err, location, "Minting tokens")
	}

	grant.CodeRedeemed = true

	// RULE: Persist the consumed code + refresh hash so the code cannot be replayed
	// and the refresh token can be validated later.
	if err := service.Save(session, grant, "Exchange authorization code"); err != nil {
		return derp.Wrap(err, location, "Saving grant")
	}

	return nil
}

// mintInitialTokens mints an access token and the INITIAL (generation-1) refresh
// token for a grant, setting the transient Token/RefreshToken and the stored refresh
// state (RefreshHash/Generation/RotatedAt). The caller is responsible for saving.
func (service *OAuthUserToken) mintInitialTokens(grant *model.OAuthUserToken, now time.Time) error {

	const location = "service.OAuthUserToken.mintInitialTokens"

	token, err := service.JWT(grant.OAuthUserTokenID, grant.UserID, strings.Join(grant.Scopes, " "))

	if err != nil {
		return derp.Wrap(err, location, "Generating access token")
	}

	secret, err := model.NewRefreshSecret()

	if err != nil {
		return derp.Wrap(err, location, "Generating refresh token")
	}

	grant.Token = token
	grant.InitRefresh(secret, now)
	grant.RefreshToken = model.BuildRefreshToken(grant.OAuthUserTokenID, grant.Generation, secret)

	return nil
}

// RefreshGrant rotates a grant in response to a refresh-token grant (RFC 6749 §6,
// with RFC 6819 rotation + reuse detection). It classifies the presented
// (generation, secret) and either rotates the grant forward — issuing a new
// access+refresh pair — or, on a confirmed reuse, revokes the grant. `scope`, when
// non-empty, may narrow (never widen) the granted scopes for the issued token.
func (service *OAuthUserToken) RefreshGrant(session data.Session, grant *model.OAuthUserToken, generation int, secret string, scope string) error {

	const location = "service.OAuthUserToken.RefreshGrant"

	now := time.Now()

	switch grant.ClassifyRefresh(generation, secret, now, model.OAuthRefreshGracePeriod) {

	case model.RefreshMatchReuse:
		// RULE: A confirmed superseded secret signals theft — revoke the grant.
		if err := service.Delete(session, grant, "Refresh token reuse detected"); err != nil {
			return derp.Wrap(err, location, "Revoking grant after reuse")
		}

		return derp.Forbidden(location, "Refresh token reuse detected; grant revoked")

	case model.RefreshMatchNone:
		return derp.BadRequest(location, "Invalid refresh_token")
	}

	// RULE: The requested scope may narrow, never widen, the granted scope.
	issuedScopes, err := narrowScopes(grant.Scopes, scope)

	if err != nil {
		return derp.Wrap(err, location, "Invalid scope")
	}

	// Rotate the refresh token forward (current secret becomes prior, new secret
	// becomes current, generation increments).
	newSecret, err := model.NewRefreshSecret()

	if err != nil {
		return derp.Wrap(err, location, "Generating refresh token")
	}

	grant.RotateRefresh(newSecret, now)

	// RULE: Persist the rotation with the ORIGINAL granted scopes intact, so a later
	// refresh may still request any subset.
	if err := service.Save(session, grant, "Refresh token"); err != nil {
		return derp.Wrap(err, location, "Saving grant")
	}

	// Mint the response pair. issuedScopes applies to THIS access token only; the
	// grant.Scopes assignment below is an in-memory-only mutation to drive the
	// response (the record was already saved with the full granted scopes).
	token, err := service.JWT(grant.OAuthUserTokenID, grant.UserID, strings.Join(issuedScopes, " "))

	if err != nil {
		return derp.Wrap(err, location, "Generating access token")
	}

	grant.Token = token
	grant.RefreshToken = model.BuildRefreshToken(grant.OAuthUserTokenID, grant.Generation, newSecret)
	grant.Scopes = issuedScopes

	return nil
}

// narrowScopes returns the scopes to issue for a refresh. An empty request keeps
// the full granted set; otherwise the requested scopes MUST each be a subset of
// the granted set (RFC 6749 §6 — a refresh may narrow scope but never widen it).
func narrowScopes(granted sliceof.String, requested string) (sliceof.String, error) {

	if requested == "" {
		return granted, nil
	}

	result := sliceof.NewString()

	for _, scope := range strings.Fields(strings.ReplaceAll(requested, ",", " ")) {

		if !slice.Contains(granted, scope) {
			return nil, derp.BadRequest("service.OAuthUserToken.narrowScopes", "Requested scope exceeds granted scope", scope)
		}

		result = append(result, scope)
	}

	return result, nil
}

// DeleteByClient marks every OAuth token issued to the provided client as deleted
func (service *OAuthUserToken) DeleteByClient(session data.Session, clientID primitive.ObjectID, note string) error {
	criteria := exp.Equal("clientId", clientID)
	return service.DeleteMany(session, criteria, note)
}

// JWT encodes an access token for the provided grant as a new, short-lived JWT.
// The token carries a 1-hour expiry (RFC 6749) and the grant ID, so OAuth API
// routes can load the grant record and revocation is effective.
func (service *OAuthUserToken) JWT(grantID primitive.ObjectID, userID primitive.ObjectID, scopes string) (string, error) {

	now := time.Now()

	// Collect claims
	claims := jwt.MapClaims{
		"A":   true,                                                        // apiUser
		"U":   userID,                                                      // UserID
		"S":   scopes,                                                      // Scopes
		"K":   grantID,                                                     // OAuthUserTokenID (the grant this token was issued from)
		"sub": userID.Hex(),                                                // Standard subject (the user), for revalidation
		"iat": jwt.NewNumericDate(now),                                     // Issued-at
		"exp": jwt.NewNumericDate(now.Add(model.OAuthAccessTokenLifetime)), // Expiry (RFC 6749 — access tokens age out)
	}

	// Create the token
	result := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign the token
	keyName, keyValue, err := service.jwtService.GetCurrentKey()

	if err != nil {
		return "", derp.Wrap(err, "service.OAuthUserToken.JWT", "Creating new JWT key")
	}

	result.Header["kid"] = keyName

	token, err := result.SignedString(keyValue)

	if err != nil {
		return "", derp.Wrap(err, "service.OAuthUserToken.JWT", "Signing JWT")
	}

	// Woot.
	return token, nil
}
