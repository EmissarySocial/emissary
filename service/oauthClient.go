package service

import (
	"crypto/subtle"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/cimd"
	"github.com/EmissarySocial/emissary/tools/random"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/sliceof"
	"github.com/benpate/uri"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// OAuthClient manages all interactions with the OAuthClient collection
type OAuthClient struct {
	oauthUserTokenService *OAuthUserToken
	activityService       *ActivityStream
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
func (service *OAuthClient) Refresh(factory *Factory) {
	service.oauthUserTokenService = factory.OAuthUserToken()
	service.activityService = factory.ActivityStream()
	service.host = factory.Hostname()
}

// Close stops any background processes controlled by this service
func (service *OAuthClient) Close() {

}

/******************************************
 * Common Data Methods
 ******************************************/

// collection returns the OAuthClient collection for the provided database session
func (service *OAuthClient) collection(session data.Session) data.Collection {
	return session.Collection("OAuthClient")
}

// Count returns the number of records that match the provided criteria
func (service *OAuthClient) Count(session data.Session, criteria exp.Expression) (int64, error) {
	return service.collection(session).Count(notDeleted(criteria))
}

// Query returns an slice containing all of the OAuthClients that match the provided criteria
func (service *OAuthClient) Query(session data.Session, criteria exp.Expression, options ...option.Option) ([]model.OAuthClient, error) {
	result := make([]model.OAuthClient, 0)
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)
	return result, err
}

// Iterator returns an iterator containing all of the OAuthClients that match the provided criteria
func (service *OAuthClient) Iterator(session data.Session, criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection(session).Iterator(notDeleted(criteria), options...)
}

// Load retrieves an OAuthClient from the database
func (service *OAuthClient) Load(session data.Session, criteria exp.Expression, client *model.OAuthClient) error {

	if err := service.collection(session).Load(notDeleted(criteria), client); err != nil {
		return derp.Wrap(err, "service.OAuthClient.Load", "Loading OAuthClient", criteria)
	}

	return nil
}

// Save adds/updates an OAuthClient in the database
func (service *OAuthClient) Save(session data.Session, client *model.OAuthClient, note string) error {

	const location = "service.OAuthClient.Save"

	// Validate the value (using the global OAuthClient schema) before saving
	if _, err := service.Schema().Validate(client); err != nil {
		return derp.Wrap(err, location, "Validating OAuthClient using OAuthClientSchema", client)
	}

	// Generate secrets for new clients that weren't created via "Client ID Metadata Documents"
	if client.IsNew() && (client.ClientURL == "") {

		// Generate a new ClientSecret
		secret, err := random.GenerateString(64)

		if err != nil {
			return derp.Wrap(err, location, "Generating client secret")
		}

		client.ClientSecret = secret
	}

	// Try to save the OAuthClient to the database
	if err := service.collection(session).Save(client, note); err != nil {
		return derp.Wrap(err, location, "Saving OAuthClient", client, note)
	}

	return nil
}

// Delete removes an OAuthClient from the database (virtual delete)
func (service *OAuthClient) Delete(session data.Session, client *model.OAuthClient, note string) error {

	const location = "service.OAuthClient.Delete"

	// Delete this OAuthClient
	if err := service.collection(session).Delete(client, note); err != nil {
		return derp.Wrap(err, location, "Deleting OAuthClient", client, note)
	}

	// Delete related records -- this can happen in the background
	if err := service.oauthUserTokenService.DeleteByClient(session, client.ClientID, note); err != nil {
		return derp.Wrap(err, location, "Deleting attachments", client, note)
	}

	// Bueno!!
	return nil
}

// Schema returns the rosetta schema that describes a OAuthClient
func (service *OAuthClient) Schema() schema.Schema {
	return schema.New(model.OAuthClientSchema())
}

/******************************************
 * Custom Queries
 ******************************************/

// QueryByUserID returns a slice of OAuthClients
func (service *OAuthClient) QueryByUserID(session data.Session, userID primitive.ObjectID) (sliceof.Object[model.OAuthClient], error) {
	criteria := exp.Equal("userId", userID)
	return service.Query(session, criteria, option.Fields("_id", "name", "summary", "iconUrl", "website"))
}

// LoadByClientID loads a single OAuth client using the "client_id" field (which is just a stringified ObjectID)
func (service *OAuthClient) LoadByClientID(session data.Session, clientID primitive.ObjectID, client *model.OAuthClient) error {

	criteria := exp.Equal("_id", clientID)
	return service.Load(session, criteria, client)
}

// LoadByToken loads a single OAuth client.  If the token is an ObjectID, then it searches on the ClientID
// field.  Otherwise, it searches on the ActorID field
func (service *OAuthClient) LoadByToken(session data.Session, token string, client *model.OAuthClient) error {

	// If the token is an ObjectID, then load using ClientID
	if clientID, err := primitive.ObjectIDFromHex(token); err == nil {
		return service.LoadByClientID(session, clientID, client)
	}

	// Load the client using the ActorID instead
	criteria := exp.Equal("clientUrl", token)
	return service.Load(session, criteria, client)
}

// LoadOrCreateByClientToken loads a single OAuth client.  If the token is an ObjectID, then it searches for
func (service *OAuthClient) LoadOrCreateByClientToken(session data.Session, token string, client *model.OAuthClient) error {

	const location = "service.OAuthClient.LoadOrCreateByClientToken"

	// First, try to load the client using the token
	if err := service.LoadByToken(session, token, client); err != nil {

		// Anything other than NotFound is a real failure; a NotFound falls through to create one
		if !derp.IsNotFound(err) {
			return derp.Wrap(err, location, "Loading OAuthClient", token)
		}
	} else {
		return nil
	}

	// Otherwise, create a new Client by looking up the "Client ID Metadata Document"
	metadata, err := cimd.GetMetadata(service.host, token)

	if err != nil {
		return derp.Wrap(err, location, "Loading Client ID Metadata Document", token)
	}

	// Populate the new Client from the Actor's data
	client.ClientURL = token
	client.Name = metadata.ClientName
	client.RedirectURIs = metadata.RedirectURIs
	client.IconURL = metadata.LogoURI
	client.RedirectURIs = metadata.RedirectURIs
	client.Website = uri.Hostname(metadata.ClientURI)

	// Save the new Client
	if err := service.Save(session, client, "Created via Client ID Metadata Document"); err != nil {
		return derp.Wrap(err, location, "Saving client")
	}

	// Success.
	return nil
}

/******************************************
 * Custom Methods
 ******************************************/

// ValidateClientSecret confirms that the provided secret matches the named OAuthClient
func (service *OAuthClient) ValidateClientSecret(session data.Session, clientID primitive.ObjectID, clientSecret string) error {

	const location = "service.OAuthClient.ValidateClientSecret"

	// Try to load the client to confirm its secret
	client := model.NewOAuthClient()
	if err := service.LoadByClientID(session, clientID, &client); err != nil {
		return derp.Wrap(err, location, "Loading client", clientID)
	}

	// Confirm the client.Secret using a constant-time comparison to avoid
	// leaking the secret through timing.
	if subtle.ConstantTimeCompare([]byte(client.ClientSecret), []byte(clientSecret)) != 1 {
		return derp.BadRequest(location, "Invalid client_secret")
	}

	// Success!
	return nil
}
