package ascache

import (
	"context"
	"time"

	"github.com/EmissarySocial/emissary/tools/cacheheader"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Client struct {
	commonDatabase data.Server
	enqueue        chan<- queue.Task
	innerClient    streams.Client
	hostname       string
	actorType      string
	actorID        primitive.ObjectID
	obeyHeaders    bool
}

// New returns a fully initialized Client object
func New(innerClient streams.Client, enqueue chan<- queue.Task, commonDatabase data.Server, actorType string, actorID primitive.ObjectID, hostname string, options ...ClientOptionFunc) *Client {

	// Create a default client
	result := &Client{
		commonDatabase: commonDatabase,
		enqueue:        enqueue,
		innerClient:    innerClient,
		hostname:       hostname,
		actorType:      actorType,
		actorID:        actorID,
		obeyHeaders:    true,
	}

	// Default our child's "RootClient" to our current value.
	// This may be overridden by a parent
	result.innerClient.SetRootClient(result)

	// Apply option functions to the client
	result.WithOptions(options...)

	// Woot woot.
	return result
}

// WithOptions applies one or more options to the Client
func (client *Client) WithOptions(options ...ClientOptionFunc) {
	for _, option := range options {
		option(client)
	}
}

/******************************************
 * Hannibal HTTP Client Methods
 ******************************************/

// SetRootClient applies a "top level" client (which is needed by some hannibal client implementations)
func (client *Client) SetRootClient(rootClient streams.Client) {
	client.innerClient.SetRootClient(rootClient)
}

// Load retrieves a URL from the cache/interweb, returning it as a streams.Document
func (client *Client) Load(url string, options ...any) (streams.Document, error) {

	const location = "ascache.Client.Load"

	// Get load config
	config := NewLoadConfig(options...)

	// Create a new database session and connect to the document cach collection
	session, cancel, err := client.timeoutSession(config.timeoutSeconds)

	if err != nil {
		return streams.NilDocument(), derp.Wrap(err, location, "Unable to connect to database")
	}

	defer cancel()

	// If we're not forcing the cache to reload, then try to load from the cache first
	if config.isCacheAllowed() {

		// Search the cache for the document
		value := NewValue()

		if err := client.loadByURL(session, url, &value); err == nil {

			if value.ShouldRevalidate() {
				client.revalidate(&value)
			}

			// Mark this values as "cached"
			value.HTTPHeader.Set(HeaderHannibalCache, "true")
			value.HTTPHeader.Set(HeaderHannibalCacheDate, time.Now().Format(time.RFC3339))

			// Return cached document to the caller (no HTTP call required)
			return client.asDocument(value), nil
		}
	}

	// Pass the request to the inner client
	result, err := client.innerClient.Load(url, options...)

	if err != nil {

		// If the original document is gone, and we're forcing a reload, then remove the value from the cache
		if derp.IsNotFound(err) && config.forceReload {
			if err := client.Delete(url); err != nil {
				return result, derp.Wrap(err, location, "Unable to remove document from cache", url)
			}
		}

		return result, derp.Wrap(err, location, "Unable to load document from inner client", url)
	}

	// Try to save the new value asynchronously
	value := asValue(result)

	if err := client.save(session.Context(), url, &value); err != nil {
		derp.Report(derp.Wrap(err, location, "Unable to write document to cache.. continuing process.."))
	}

	client.countRelatedDocuments(result)

	// Return the result (streams.Document) to the caller
	return result, nil
}

func (client *Client) Save(document streams.Document) error {

	const location = "ascache.Client.Save"

	// Get a new database session
	ctx, cancel := timeoutContext(60)
	defer cancel()

	// Save the document/value to the database
	value := asValue(document)

	if err := client.save(ctx, document.ID(), &value); err != nil {
		return derp.Wrap(err, location, "Unable to put document into cache")
	}

	return nil
}

// Delete removes a single document from the cache
func (client *Client) Delete(url string) error {

	const location = "ascache.Client.Delete"

	// Connect to the database; get a session and collection
	ctx, cancel := timeoutContext(10)
	defer cancel()

	session, err := client.commonDatabase.Session(ctx)

	if err != nil {
		return derp.InternalError(location, "Unable to connect to ActivityStream cache")
	}

	collection := client.collection(session)

	// Load the document from the database (to recalculate statistics after delete)
	value := NewValue()
	if err := client.loadByURL(session, url, &value); err != nil {

		if derp.IsNotFound(err) {
			return nil
		}

		return derp.Wrap(err, location, "Unable to load cached ActivityStream document", url)
	}

	// Delete the document from the cache
	criteria := exp.Equal("urls", url)

	if err := collection.HardDelete(criteria); err != nil {
		return derp.Wrap(err, location, "Unable to delete", url)
	}

	// Maybe recalculate statistics
	if value.Metadata.HasRelationship() {
		if err := client.CalcParentRelationships(session, value.Metadata.RelationType, value.Metadata.RelationHref); err != nil {
			return derp.Wrap(err, location, "Error calculating statistics", url)
		}
	}

	// Success!
	return nil
}

/******************************************
 * Database Methods
 ******************************************/

func (client *Client) session(ctx context.Context) (data.Session, error) {

	const location = "ascache.Client.session"

	if client.commonDatabase == nil {
		return nil, derp.InternalError(location, "Common Database is not initialized")
	}

	session, err := client.commonDatabase.Session(ctx)

	if err != nil {
		return nil, derp.Wrap(err, location, "Unable to connect to common database")
	}

	return session, nil
}

func (client *Client) timeoutSession(seconds int) (data.Session, context.CancelFunc, error) {

	const location = "ascache.Client.timeoutSession"

	ctx, cancel := timeoutContext(seconds)

	session, err := client.session(ctx)

	if err != nil {
		return nil, nil, derp.Wrap(err, location, "Unable to connect to common database")
	}

	return session, cancel, nil
}

func (client *Client) collection(session data.Session) data.Collection {
	return session.Collection("Document")
}

// save adds/updates a document in the cache
func (client *Client) save(ctx context.Context, url string, value *Value) error {

	const location = "ascache.Client.save"

	_, err := client.commonDatabase.WithTransaction(ctx, func(session data.Session) (any, error) {

		// Write to trace log
		log.Trace().Str("url", url).Msg(location)

		// Calculate caching rules and exit if cache is not allowed.
		cacheControl := cacheheader.Parse(value.HTTPHeader)
		if client.obeyHeaders && cacheControl.NotCacheAllowed() {
			log.Trace().Str("url", url).Msg("Cache not allowed by HTTP headers. Skipping save method.")
			return nil, nil
		}

		// Make sure all relevant URLs are included in this value
		value.AppendURL(value.Object.GetString("id"))
		value.AppendURL(url)

		// Try to load an existing/duplicate values using the object.id field.
		// There may be multiple URLs that point to the same document, so we're
		// doing this check HERE using the object.id field.

		if err := client.removeDuplicates(session, value.URLs...); err != nil {
			return nil, derp.Wrap(err, location, "Unable to search for duplicate document in cache")
		}

		// Some calculations before we save the value
		value.Received = time.Now().Unix()
		value.calcPublished()
		value.calcExpires(cacheControl)
		value.calcRevalidates(cacheControl)

		// Try to upsert the document into the cache
		collection := client.collection(session)
		if err := collection.Save(value, "updated"); err != nil {
			return nil, derp.Wrap(err, location, "Unable to save cached value", url)
		}

		// Maybe recalculate statistics
		if value.Metadata.HasRelationship() {
			if err := client.CalcParentRelationships(session, value.Metadata.RelationType, value.Metadata.RelationHref); err != nil {
				return nil, derp.Wrap(err, location, "Unable to calculate relationships", url)
			}
		}

		// Success.
		return nil, nil
	})

	return err

}

// removeDuplicates removes all valus that have duplicate URLs
func (client *Client) removeDuplicates(session data.Session, urls ...string) error {

	const location = "ascache.Client.removeDuplicates"

	collection := client.collection(session)

	criteria := exp.In("urls", urls)

	if err := collection.HardDelete(criteria); err != nil {
		return derp.Wrap(err, location, "Unable to remove duplicate documents from cache", urls)
	}

	return nil
}

// countRelatedDocuments triggers a follow-up task to count related documents.
func (client *Client) countRelatedDocuments(document streams.Document) {

	// Do not count related documents for Actors
	if document.IsActor() {
		return
	}

	// Do not count related documents for Collections
	if document.IsCollection() {
		return
	}

	url := document.ID()

	// Schedule a follow-up to count related documents (in 30 seconds)
	client.enqueue <- queue.NewTask(
		"CountRelatedDocuments",
		mapof.Any{
			"url":       url,
			"host":      client.hostname,
			"actorType": client.actorType,
			"actorID":   client.actorID,
		},
		queue.WithSignature(url+"#CountRelatedDocuments"),
		queue.WithDelaySeconds(30),
		queue.WithPriority(8),
	)
}

// asDocument converts a Document into a fully-populated streams.Document
func (client *Client) asDocument(value Value) streams.Document {

	return streams.NewDocument(
		value.Object,
		streams.WithClient(client),
		streams.WithMetadata(value.Metadata),
		streams.WithHTTPHeader(value.HTTPHeader),
	)
}

/******************************************
 * Other Queries
 ******************************************/

// load loads a Value from the cache using any criteria expression.
func (client *Client) load(session data.Session, criteria exp.Expression, value *Value) error {

	const location = "ascache.Client.load"

	// Get the database connection
	collection := client.collection(session)

	// Query the cache database
	if err := collection.Load(criteria, value); err != nil {
		return derp.Wrap(err, location, "Unable to load cached value", criteria)
	}

	// Success.
	return nil
}

// loadByURL loads a Value from the cache using its URL.
// This value can match any of the URLs in the "urls" array.
func (client *Client) loadByURL(session data.Session, url string, value *Value) error {
	return client.load(session, exp.Equal("urls", url), value)
}

func (client *Client) revalidate(value *Value) {

	// If we're allowed to write to the cache, then do it.
	if value.ShouldRevalidate() {

		documentID := value.DocumentID()

		client.enqueue <- queue.NewTask(
			"LoadActivityStream",
			mapof.Any{
				"host":         client.hostname,
				"actorType":    client.actorType,
				"actorID":      client.actorID,
				"url":          documentID,
				"revalidating": true,
			},
			queue.WithSignature(documentID),
			queue.WithPriority(512),
		)
	}
}
