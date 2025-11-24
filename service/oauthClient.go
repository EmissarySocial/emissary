package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/random"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/sherlock"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// OAuthClient manages all interactions with the OAuthClient collection
type OAuthClient struct {
	oauthUserTokenService *OAuthUserToken
	activityService       ActivityStream
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
func (service *OAuthClient) Refresh(oauthUserTokenService *OAuthUserToken, activityService ActivityStream, host string) {
	service.oauthUserTokenService = oauthUserTokenService
	service.activityService = activityService
	service.host = host
}

// Close stops any background processes controlled by this service
func (service *OAuthClient) Close() {

}

/******************************************
 * Common Data Methods
 ******************************************/

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
		return derp.Wrap(err, "service.OAuthClient.Load", "Unable to load OAuthClient", criteria)
	}

	return nil
}

// Save adds/updates an OAuthClient in the database
func (service *OAuthClient) Save(session data.Session, client *model.OAuthClient, note string) error {

	const location = "service.OAuthClient.Save"

	// Validate the value (using the global OAuthClient schema) before saving
	if err := service.Schema().Validate(client); err != nil {
		return derp.Wrap(err, location, "Error validating OAuthClient using OAuthClientSchema", client)
	}

	// If this is a new record and NOT an ActivityPub client
	if client.IsNew() && (client.ActorID == "") {

		// Generate a new ClientSecret
		secret, err := random.GenerateString(64)

		if err != nil {
			return derp.Wrap(err, location, "Unable to generate client secret")
		}

		client.ClientSecret = secret
	}

	// Try to save the OAuthClient to the database
	if err := service.collection(session).Save(client, note); err != nil {
		return derp.Wrap(err, location, "Unable to save OAuthClient", client, note)
	}

	return nil
}

// Delete removes an OAuthClient from the database (virtual delete)
func (service *OAuthClient) Delete(session data.Session, client *model.OAuthClient, note string) error {

	const location = "service.OAuthClient.Delete"

	// Delete this OAuthClient
	if err := service.collection(session).Delete(client, note); err != nil {
		return derp.Wrap(err, location, "Error deleting OAuthClient", client, note)
	}

	// Delete related records -- this can happen in the background
	if err := service.oauthUserTokenService.DeleteByClient(session, client.ClientID, note); err != nil {
		return derp.Wrap(err, location, "Error deleting attachments", client, note)
	}

	// Bueno!!
	return nil
}

func (service *OAuthClient) Schema() schema.Schema {
	return schema.New(model.OAuthClientSchema())
}

/******************************************
 * Custom Queries
 ******************************************/

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
	criteria := exp.Equal("actorId", token)
	return service.Load(session, criteria, client)
}

// LoadOrCreateByClientToken loads a single OAuth client.  If the token is an ObjectID, then it searches for
func (service *OAuthClient) LoadOrCreateByClientToken(session data.Session, token string, client *model.OAuthClient) error {

	const location = "service.OAuthClient.LoadOrCreateByClientToken"

	// If the token is an ObjectID, then just use that.  It's not possible
	// to look up Actors from an ObjectID, so if this fails, it just fails.
	if clientID, err := primitive.ObjectIDFromHex(token); err == nil {
		return service.LoadByClientID(session, clientID, client)
	}

	// Try to load the client using the ActorID, if success then success
	criteria := exp.Equal("actorId", token)
	if err := service.Load(session, criteria, client); err == nil {
		return nil
	}

	// Otherwise, create a new Client by looking up the ActivityPub actor
	actor, err := service.activityService.Client().Load(token, sherlock.AsActor())

	if err != nil {
		return derp.Wrap(err, location, "Unable to find Client by Token", token)
	}

	// Populate the new Client from the Actor's data
	client.ActorID = token
	client.Name = actor.Name()
	client.RedirectURIs = convert.SliceOfString(actor.Get(vocab.PropertyRedirectURI).Value())
	client.IconURL = actor.Icon().Href()
	client.Summary = actor.Summary()
	client.Scopes = []string{"read:export", "write:move"}

	// Save the new Client
	if err := service.Save(session, client, "Created via ActivityPub actor"); err != nil {
		return derp.Wrap(err, location, "Unable to save client")
	}

	// Success.
	return nil
}

/******************************************
 * Custom Methods
 ******************************************/

func (service *OAuthClient) ValidateClientSecret(session data.Session, clientID primitive.ObjectID, clientSecret string) error {

	const location = "service.OAuthClient.ValidateClientSecret"

	// Try to load the client to confirm its secret
	client := model.NewOAuthClient()
	if err := service.LoadByClientID(session, clientID, &client); err != nil {
		return derp.Wrap(err, location, "Unable to load client", clientID)
	}

	// Confirm the client.Secret
	if client.ClientSecret != clientSecret {
		return derp.BadRequestError(location, "Invalid client_secret")
	}

	// Success!
	return nil
}
