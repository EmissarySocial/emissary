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

// OAuthApplication manages all interactions with the OAuthApplication collection
type OAuthApplication struct {
	collection            data.Collection
	oauthUserTokenService *OAuthUserToken
	host                  string
}

// NewOAuthApplication returns a fully populated OAuthApplication service.
func NewOAuthApplication() OAuthApplication {
	return OAuthApplication{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *OAuthApplication) Refresh(collection data.Collection, oauthUserTokenService *OAuthUserToken, host string) {
	service.collection = collection
	service.oauthUserTokenService = oauthUserTokenService
	service.host = host
}

// Close stops any background processes controlled by this service
func (service *OAuthApplication) Close() {

}

/******************************************
 * Common Data Methods
 ******************************************/

// Query returns an slice containing all of the OAuthApplications that match the provided criteria
func (service *OAuthApplication) Query(criteria exp.Expression, options ...option.Option) ([]model.OAuthApplication, error) {
	result := make([]model.OAuthApplication, 0)
	err := service.collection.Query(&result, notDeleted(criteria), options...)
	return result, err
}

// Iterator returns an iterator containing all of the OAuthApplications that match the provided criteria
func (service *OAuthApplication) Iterator(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.Iterator(notDeleted(criteria), options...)
}

// Load retrieves an OAuthApplication from the database
func (service *OAuthApplication) Load(criteria exp.Expression, application *model.OAuthApplication) error {

	if err := service.collection.Load(notDeleted(criteria), application); err != nil {
		return derp.Wrap(err, "service.OAuthApplication", "Error loading OAuthApplication", criteria)
	}

	return nil
}

// Save adds/updates an OAuthApplication in the database
func (service *OAuthApplication) Save(app *model.OAuthApplication, note string) error {

	const location = "service.OAuthApplication.Save"

	// Clean the value (using the global OAuthApplication schema) before saving
	if err := service.Schema().Clean(app); err != nil {
		return derp.Wrap(err, location, "Error cleaning OAuthApplication using OAuthApplicationSchema", app)
	}

	// If this is a new record, generate client secret
	if app.IsNew() {
		secret, err := random.GenerateString(64)

		if err != nil {
			return derp.Wrap(err, location, "Error generating client secret")
		}

		app.ClientSecret = secret
	}

	// Try to save the OAuthApplication to the database
	if err := service.collection.Save(app, note); err != nil {
		return derp.Wrap(err, location, "Error saving OAuthApplication", app, note)
	}

	return nil
}

// Delete removes an OAuthApplication from the database (virtual delete)
func (service *OAuthApplication) Delete(app *model.OAuthApplication, note string) error {

	// Delete this OAuthApplication
	if err := service.collection.Delete(app, note); err != nil {
		return derp.Wrap(err, "service.OAuthApplication.Delete", "Error deleting OAuthApplication", app, note)
	}

	// Delete related records -- this can happen in the background
	go func() {

		// RULE: Delete all related Attachments
		if err := service.oauthUserTokenService.DeleteByApplication(app.OAuthApplicationID, note); err != nil {
			derp.Report(derp.Wrap(err, "service.OAuthApplication.Delete", "Error deleting attachments", app, note))
		}
	}()

	// Bueno!!
	return nil
}

func (service *OAuthApplication) Schema() schema.Schema {
	return schema.New(model.OAuthApplicationSchema())
}

/******************************************
 * Custom Data Methods
 ******************************************/

// LoadByClientID loads a single application using the "client_id" field (which is just a stringified ObjectID)
func (service *OAuthApplication) LoadByClientID(clientID string, app *model.OAuthApplication) error {

	const location = "service.OAuthApplication.LoadByClientID"

	// Parse the clientID
	oauthApplicationID, err := primitive.ObjectIDFromHex(clientID)

	if err != nil {
		return derp.Wrap(err, location, "Invalid client ID", clientID)
	}

	// Query and return
	criteria := exp.Equal("_id", oauthApplicationID)
	return service.Load(criteria, app)
}
