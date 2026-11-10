package service

import (
	"crypto/subtle"
	"net/url"
	"strings"

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

	// Validate the value (using the global OAuthClient schema) before saving.  Errors name the
	// ClientID, never the client, whose ClientSecret is hidden from JSON but not from BSON.
	if _, err := service.Schema().Validate(client); err != nil {
		return derp.Wrap(err, location, "Validating OAuthClient using OAuthClientSchema", client.ClientID)
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
		return derp.Wrap(err, location, "Saving OAuthClient", client.ClientID, note)
	}

	return nil
}

// Delete removes an OAuthClient from the database (virtual delete)
func (service *OAuthClient) Delete(session data.Session, client *model.OAuthClient, note string) error {

	const location = "service.OAuthClient.Delete"

	// Delete this OAuthClient.  Errors name its ClientID, because the client carries its secret.
	if err := service.collection(session).Delete(client, note); err != nil {
		return derp.Wrap(err, location, "Deleting OAuthClient", client.ClientID, note)
	}

	// Delete related records -- this can happen in the background
	if err := service.oauthUserTokenService.DeleteByClient(session, client.ClientID, note); err != nil {
		return derp.Wrap(err, location, "Deleting attachments", client.ClientID, note)
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

// LoadByToken loads a single OAuth client.  An ObjectID token loads a registered client by its ClientID,
// and any other token is a URL client_id, which loads by its canonical form
func (service *OAuthClient) LoadByToken(session data.Session, token string, client *model.OAuthClient) error {

	const location = "service.OAuthClient.LoadByToken"

	// RULE: An empty token names no client. Without this, it matches every client whose clientUrl is empty
	if token == "" {
		return derp.BadRequest(location, "client_id is required")
	}

	// If the token is an ObjectID, then load using ClientID
	if clientID, err := primitive.ObjectIDFromHex(token); err == nil {
		return service.LoadByClientID(session, clientID, client)
	}

	// RULE: Any other token must be a valid URL client_id, and is stored in canonical form
	clientURL, err := cimd.ParseClientID(token)

	if err != nil {
		return derp.Wrap(err, location, "Invalid client_id", token)
	}

	return service.Load(session, exp.Equal("clientUrl", clientURL), client)
}

// LoadOrCreateByClientToken loads a single OAuth client by its token.  A URL client_id seen for the first
// time is created from its Client ID Metadata Document, or from its URL alone when it publishes none
func (service *OAuthClient) LoadOrCreateByClientToken(session data.Session, token string, client *model.OAuthClient) error {

	const location = "service.OAuthClient.LoadOrCreateByClientToken"

	// First, try to load the client using the token
	err := service.LoadByToken(session, token, client)

	if err == nil {
		return nil
	}

	// Anything other than NotFound is a real failure
	if !derp.IsNotFound(err) {
		return derp.Wrap(err, location, "Loading OAuthClient", token)
	}

	// RULE: Only a URL client is created on sight; a registered client that does not exist stays not found
	if _, idError := primitive.ObjectIDFromHex(token); idError == nil {
		return derp.Wrap(err, location, "Unknown client_id", token)
	}

	// Otherwise, describe the new client from its Client ID Metadata Document, if it publishes one
	clientURL, err := cimd.ParseClientID(token)

	if err != nil {
		return derp.Wrap(err, location, "Invalid client_id", token)
	}

	metadata, published, err := cimd.GetMetadata(service.host, clientURL, service.activityService.AllowPrivateIPs())

	if err != nil {
		return derp.Wrap(err, location, "Loading Client ID Metadata Document", clientURL)
	}

	*client = newURLClient(clientURL, metadata, published)

	// Save the new client.  When another server saved the same client first, the unique index refuses
	// this one, and the client that won is the one to use
	if err := service.Save(session, client, "Created from its client_id"); err != nil {

		if !derp.IsConflict(err) {
			return derp.Wrap(err, location, "Saving client", clientURL)
		}

		*client = model.NewOAuthClient()

		if err := service.LoadByToken(session, clientURL, client); err != nil {
			return derp.Wrap(err, location, "Loading the client another server created", clientURL)
		}
	}

	// Success.
	return nil
}

// newURLClient returns a new client for a URL client_id, keeping only the metadata values its schema accepts,
// so a malformed optional value is dropped rather than refusing the client
func newURLClient(clientURL string, metadata cimd.Metadata, published bool) model.OAuthClient {

	result := model.NewOAuthClient()
	result.ClientURL = clientURL
	result.Name = uri.Hostname(clientURL)

	// A client that publishes no metadata is named by its hostname, and may redirect only to its own origin.
	// RULE: Nothing describes it but its address, so it may learn who the User is and nothing more
	if !published {
		result.Scopes = append(result.Scopes, model.OAuthUserTokenScopeProfile, model.OAuthUserTokenScopeEmail)
		return result
	}

	if name := truncateRunes(strings.TrimSpace(metadata.ClientName), model.OAuthClientNameMaxLength); name != "" {
		result.Name = name
	}

	if isWebURL(metadata.LogoURI) {
		result.IconURL = metadata.LogoURI
	}

	if isWebURL(metadata.ClientURI) {
		result.Website = metadata.ClientURI
	}

	result.RedirectURIs = append(result.RedirectURIs, metadata.RedirectURIs...)
	return result
}

// isWebURL returns TRUE if a value is an absolute http or https URL with a host
func isWebURL(value string) bool {

	parsed, err := url.Parse(value)

	if err != nil {
		return false
	}

	if (parsed.Scheme != "http") && (parsed.Scheme != "https") {
		return false
	}

	return parsed.Host != ""
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
