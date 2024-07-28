package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/schema"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// OAuthUserToken manages all interactions with the OAuthUserToken collection
type OAuthUserToken struct {
	collection         data.Collection
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
func (service *OAuthUserToken) Refresh(collection data.Collection, oauthClientService *OAuthClient, jwtService *JWT, host string) {
	service.collection = collection
	service.oauthClientService = oauthClientService
	service.jwtService = jwtService
	service.host = host
}

// Close stops any background processes controlled by this service
func (service *OAuthUserToken) Close() {

}

/******************************************
 * Common Data Methods
 ******************************************/

// Count returns the number of records that match the provided criteria
func (service *OAuthUserToken) Count(criteria exp.Expression) (int64, error) {
	return service.collection.Count(criteria)
}

// Query returns an slice containing all of the OAuthUserTokens that match the provided criteria
func (service *OAuthUserToken) Query(criteria exp.Expression, options ...option.Option) ([]model.OAuthUserToken, error) {
	result := make([]model.OAuthUserToken, 0)
	err := service.collection.Query(&result, notDeleted(criteria), options...)
	return result, err
}

// Iterator returns an iterator containing all of the OAuthUserTokens that match the provided criteria
func (service *OAuthUserToken) Iterator(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.Iterator(notDeleted(criteria), options...)
}

// Load retrieves an OAuthUserToken from the database
func (service *OAuthUserToken) Load(criteria exp.Expression, application *model.OAuthUserToken) error {

	if err := service.collection.Load(notDeleted(criteria), application); err != nil {
		return derp.Wrap(err, "service.OAuthUserToken", "Error loading OAuthUserToken", criteria)
	}

	return nil
}

// Save adds/updates an OAuthUserToken in the database
func (service *OAuthUserToken) Save(application *model.OAuthUserToken, note string) error {

	const location = "service.OAuthUserToken"

	// Validate the value (using the global application schema) before saving
	if err := service.Schema().Validate(application); err != nil {
		return derp.Wrap(err, "service.OAuthUserToken.Save", "Error validating OAuthUserToken using OAuthUserTokenSchema", application)
	}

	// Try to save the OAuthUserToken to the database
	if err := service.collection.Save(application, note); err != nil {
		return derp.Wrap(err, location, "Error saving OAuthUserToken", application, note)
	}

	return nil
}

// Delete removes an OAuthUserToken from the database (virtual delete)
func (service *OAuthUserToken) Delete(application *model.OAuthUserToken, note string) error {

	// Delete this OAuthUserToken
	if err := service.collection.Delete(application, note); err != nil {
		return derp.Wrap(err, "service.OAuthUserToken.Delete", "Error deleting OAuthUserToken", application, note)
	}

	// Bueno!!
	return nil
}

// DeleteMany removes all child streams from the provided stream (virtual delete)
func (service *OAuthUserToken) DeleteMany(criteria exp.Expression, note string) error {

	it, err := service.Iterator(criteria)

	if err != nil {
		return derp.Wrap(err, "service.Stream.Delete", "Error listing streams to delete", criteria)
	}

	userToken := model.NewOAuthUserToken()

	for it.Next(&userToken) {
		if err := service.Delete(&userToken, note); err != nil {
			return derp.Wrap(err, "service.Stream.Delete", "Error deleting stream", userToken)
		}
		userToken = model.NewOAuthUserToken()
	}

	return nil
}

func (service *OAuthUserToken) Schema() schema.Schema {
	return schema.New(model.OAuthUserTokenSchema())
}

/******************************************
 * Custom Queries
 ******************************************/

func (service *OAuthUserToken) LoadByUserAndClient(userID primitive.ObjectID, clientID primitive.ObjectID, result *model.OAuthUserToken) error {

	criteria := exp.Equal("userId", userID).
		AndEqual("clientId", clientID)

	return service.Load(criteria, result)
}

func (service *OAuthUserToken) LoadByClientAndCode(userTokenID primitive.ObjectID, clientID primitive.ObjectID, clientSecret string, result *model.OAuthUserToken) error {

	// RULE: must have a valid clientSecret to load this record
	if err := service.oauthClientService.ValidateClientSecret(clientID, clientSecret); err != nil {
		return derp.Wrap(err, "service.OAuthUserToken.LoadByClientAndToken", "Invalid client secret")
	}

	criteria := exp.Equal("_id", userTokenID).
		AndEqual("clientId", clientID)

	return service.Load(criteria, result)
}

func (service *OAuthUserToken) LoadByClientAndToken(clientID primitive.ObjectID, clientSecret string, token string, result *model.OAuthUserToken) error {

	// RULE: must have a valid clientSecret to load this record
	if err := service.oauthClientService.ValidateClientSecret(clientID, clientSecret); err != nil {
		return derp.Wrap(err, "service.OAuthUserToken.LoadByClientAndToken", "Invalid client secret")
	}

	criteria := exp.Equal("clientId", clientID).
		AndEqual("token", token)

	return service.Load(criteria, result)
}

/******************************************
 * Custom Methods
 ******************************************/

func (service *OAuthUserToken) CreateFromUser(user *model.User, clientID primitive.ObjectID, scope string) (model.OAuthUserToken, error) {

	// Load the client from the database
	client := model.NewOAuthClient()
	if err := service.oauthClientService.LoadByClientID(clientID, &client); err != nil {
		return model.OAuthUserToken{}, derp.Wrap(err, "service.OAuthUserToken.CreateFromUser", "Error loading client", clientID)
	}

	// Create the JWT authorization
	authorization := model.NewAuthorization()
	authorization.UserID = user.UserID
	authorization.GroupIDs = user.GroupIDs
	authorization.ClientID = client.ClientID
	authorization.Scope = scope
	authorization.APIUser = true

	// Mock a transaction
	txn := model.NewOAuthAuthorizationRequest()
	txn.ClientID = client.ClientID.Hex()
	txn.Scope = scope
	txn.ResponseType = "token"

	// Create and return the Token
	return service.Create(client, authorization, txn)
}

// Create creates a new OAuthUserToken for the provided application and authorization
func (service *OAuthUserToken) Create(client model.OAuthClient, authorization model.Authorization, transaction model.OAuthAuthorizationRequest) (model.OAuthUserToken, error) {

	const location = "service.OAuthUserToken.Create"

	// Require that the user is actualy logged in
	if !authorization.IsAuthenticated() {
		return model.OAuthUserToken{}, derp.NewUnauthorizedError(location, "User is not logged in")
	}

	// Validate the request
	if err := transaction.Validate(client); err != nil {
		return model.OAuthUserToken{}, derp.Wrap(err, location, "Invalid OAuthUserTokenRequest")
	}

	// If we already have a token for this user/client, then just return that.
	result := model.NewOAuthUserToken()
	if err := service.LoadByUserAndClient(authorization.UserID, client.ClientID, &result); err == nil {
		return result, nil
	}

	// Fall through means we're going to create a new token
	token, err := service.JWT(authorization.UserID, transaction.Scope)

	if err != nil {
		return model.OAuthUserToken{}, derp.Wrap(err, location, "Error generating random token")
	}

	// Copy data from the authorization
	result.ClientID = client.ClientID
	result.UserID = authorization.UserID
	result.Scopes = transaction.Scopes()
	result.Token = token
	result.APIUser = true

	// Save the result to the database
	if err := service.Save(&result, "Create"); err != nil {
		return model.OAuthUserToken{}, derp.Wrap(err, location, "Error saving OAuthUserToken", result)
	}

	return result, nil
}

func (service *OAuthUserToken) DeleteByClient(clientID primitive.ObjectID, note string) error {
	criteria := exp.Equal("clientId", clientID)
	return service.DeleteMany(criteria, note)
}

// JWT encodes an OAuthUserToken as a new JWT.
func (service *OAuthUserToken) JWT(userID primitive.ObjectID, scopes string) (string, error) {

	// Collect claims
	claims := jwt.MapClaims{
		"A": true,   // apiUser
		"U": userID, // UserID
		"S": scopes, // Scopes
	}

	// Create the token
	result := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign the token
	keyName, keyValue, err := service.jwtService.GetCurrentKey()

	if err != nil {
		return "", derp.Wrap(err, "service.OAuthUserToken.JWT", "Error creating new JWT key")
	}

	result.Header["kid"] = keyName

	token, err := result.SignedString(keyValue)

	if err != nil {
		return "", derp.Wrap(err, "service.OAuthUserToken.JWT", "Error signing JWT")
	}

	// Woot.
	return token, nil
}
