package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/random"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// OAuthClient manages all interactions with the OAuthClient collection
type OAuthClient struct {
	collection            data.Collection
	oauthUserTokenService *OAuthUserToken
	host                  string
}

// NewOAuthClient returns a fully populated OAuthClient service.
func NewOAuthClient() OAuthClient {
	return OAuthClient{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *OAuthClient) Refresh(collection data.Collection, oauthUserTokenService *OAuthUserToken, host string) {
	service.collection = collection
	service.oauthUserTokenService = oauthUserTokenService
	service.host = host
}

// Close stops any background processes controlled by this service
func (service *OAuthClient) Close() {

}

/******************************************
 * Common Data Methods
 ******************************************/

// Query returns an slice containing all of the OAuthClients that match the provided criteria
func (service *OAuthClient) Query(criteria exp.Expression, options ...option.Option) ([]model.OAuthClient, error) {
	result := make([]model.OAuthClient, 0)
	err := service.collection.Query(&result, notDeleted(criteria), options...)
	return result, err
}

// Iterator returns an iterator containing all of the OAuthClients that match the provided criteria
func (service *OAuthClient) Iterator(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.Iterator(notDeleted(criteria), options...)
}

// Load retrieves an OAuthClient from the database
func (service *OAuthClient) Load(criteria exp.Expression, application *model.OAuthClient) error {

	if err := service.collection.Load(notDeleted(criteria), application); err != nil {
		return derp.Wrap(err, "service.OAuthClient", "Error loading OAuthClient", criteria)
	}

	return nil
}

// Save adds/updates an OAuthClient in the database
func (service *OAuthClient) Save(app *model.OAuthClient, note string) error {

	const location = "service.OAuthClient.Save"

	// Clean the value (using the global OAuthClient schema) before saving
	if err := service.Schema().Clean(app); err != nil {
		return derp.Wrap(err, location, "Error cleaning OAuthClient using OAuthClientSchema", app)
	}

	// If this is a new record, generate client secret
	if app.IsNew() {
		secret, err := random.GenerateString(64)

		if err != nil {
			return derp.Wrap(err, location, "Error generating client secret")
		}

		app.ClientSecret = secret
	}

	// Try to save the OAuthClient to the database
	if err := service.collection.Save(app, note); err != nil {
		return derp.Wrap(err, location, "Error saving OAuthClient", app, note)
	}

	return nil
}

// Delete removes an OAuthClient from the database (virtual delete)
func (service *OAuthClient) Delete(app *model.OAuthClient, note string) error {

	// Delete this OAuthClient
	if err := service.collection.Delete(app, note); err != nil {
		return derp.Wrap(err, "service.OAuthClient.Delete", "Error deleting OAuthClient", app, note)
	}

	// Delete related records -- this can happen in the background
	go func() {

		// RULE: Delete all related Attachments
		if err := service.oauthUserTokenService.DeleteByClient(app.ClientID, note); err != nil {
			derp.Report(derp.Wrap(err, "service.OAuthClient.Delete", "Error deleting attachments", app, note))
		}
	}()

	// Bueno!!
	return nil
}

func (service *OAuthClient) Schema() schema.Schema {
	return schema.New(model.OAuthClientSchema())
}

/******************************************
 * Custom Queries
 ******************************************/

// LoadByClientID loads a single application using the "client_id" field (which is just a stringified ObjectID)
func (service *OAuthClient) LoadByClientID(clientID primitive.ObjectID, app *model.OAuthClient) error {

	criteria := exp.Equal("_id", clientID)
	return service.Load(criteria, app)
}

/******************************************
 * Custom Methods
 ******************************************/

func (service *OAuthClient) ValidateClientSecret(clientID primitive.ObjectID, clientSecret string) error {

	const location = "service.OAuthClient.ValidateClientSecret"

	// Try to load the client to confirm its secret
	client := model.NewOAuthClient()
	if err := service.LoadByClientID(clientID, &client); err != nil {
		return derp.Wrap(err, location, "Error loading client", clientID)
	}

	// Confirm the client.Secret
	if client.ClientSecret != clientSecret {
		return derp.NewBadRequestError(location, "Invalid client_secret")
	}

	// Success!
	return nil
}
