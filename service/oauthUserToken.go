package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/random"
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
	collection              data.Collection
	oauthApplicationService *OAuthApplication
	jwtService              JWT
	host                    string
}

// NewOAuthUserToken returns a fully populated OAuthUserToken service.
func NewOAuthUserToken() OAuthUserToken {
	return OAuthUserToken{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *OAuthUserToken) Refresh(collection data.Collection, oauthApplicationService *OAuthApplication, jwtService JWT, host string) {
	service.collection = collection
	service.oauthApplicationService = oauthApplicationService
	service.jwtService = jwtService
	service.host = host
}

// Close stops any background processes controlled by this service
func (service *OAuthUserToken) Close() {

}

/******************************************
 * Common Data Methods
 ******************************************/

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

	// Clean the value (using the global application schema) before saving
	if err := service.Schema().Clean(application); err != nil {
		return derp.Wrap(err, "service.OAuthUserToken.Save", "Error cleaning OAuthUserToken using OAuthUserTokenSchema", application)
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
 * Custom Methods
 ******************************************/

// Create creates a new OAuthUserToken for the provided application and authorization
func (service *OAuthUserToken) Create(application model.OAuthApplication, authorization model.Authorization, transaction model.OAuthAuthorizationRequest) (model.OAuthUserToken, error) {

	const location = "service.OAuthUserToken.Create"

	// Require that the user is actualy logged in
	if !authorization.IsAuthenticated() {
		return model.OAuthUserToken{}, derp.NewUnauthorizedError(location, "User is not logged in")
	}

	// Validate the request
	if err := transaction.Validate(application); err != nil {
		return model.OAuthUserToken{}, derp.Wrap(err, location, "Invalid OAuthUserTokenRequest")
	}

	// Create a random token
	token, err := random.GenerateString(64)

	if err != nil {
		return model.OAuthUserToken{}, derp.Wrap(err, location, "Error generating random token")
	}

	// Create the result object
	result := model.NewOAuthUserToken()

	// Copy data from the authorization
	result.OAuthApplicationID = application.OAuthApplicationID
	result.ClientSecret = application.ClientSecret
	result.UserID = authorization.UserID
	result.Scopes = transaction.Scopes()
	result.Token = token

	// Save the result to the database
	if err := service.Save(&result, "Create"); err != nil {
		return model.OAuthUserToken{}, derp.Wrap(err, location, "Error saving OAuthUserToken", result)
	}

	return result, nil
}

func (service *OAuthUserToken) DeleteByApplication(applicationID primitive.ObjectID, note string) error {
	criteria := exp.Equal("applicationId", applicationID)
	return service.DeleteMany(criteria, note)
}

// JWT encodes an OAuthUserToken as a new JWT.
func (service *OAuthUserToken) JWT(oauthUserToken model.OAuthUserToken) (string, error) {

	// Collect claims
	claims := jwt.MapClaims{
		"api":     true,
		"userId":  oauthUserToken.UserID,
		"scopes:": oauthUserToken.Scopes,
	}

	// Create the token
	result := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign the token
	keyName, keyValue := service.jwtService.NewJWTKey()
	result.Header["kid"] = keyName

	token, err := result.SignedString(keyValue)

	if err != nil {
		return "", derp.Wrap(err, "service.OAuthUserToken.JWT", "Error signing JWT")
	}

	// Woot.
	return token, nil
}
