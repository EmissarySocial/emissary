package ascache

import (
	"context"
	"time"

	"github.com/EmissarySocial/emissary/tools/cacheheader"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/streams"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Client struct {
	collection     *mongo.Collection
	innerClient    streams.Client
	purgeFrequency int64
	cacheMode      string
	obeyHeaders    bool
}

// New returns a fully initialized Client object
func New(innerClient streams.Client, collection *mongo.Collection, options ...ClientOptionFunc) *Client {

	// Create a default client
	result := Client{
		collection:     collection,
		innerClient:    innerClient,
		purgeFrequency: 60 * 60 * 4, // Default purge frequency is 4 hours
		cacheMode:      CacheModeReadWrite,
		obeyHeaders:    true,
	}

	// Apply option functions to the client
	result.WithOptions(options...)

	go result.start()

	return &result
}

func (client *Client) WithOptions(options ...ClientOptionFunc) {
	for _, option := range options {
		option(client)
	}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// start is a background process that purges expired documents from the cache
func (client *Client) start() {

	// If the client is not writable then don't purge expired documents
	if client.NotWritable() {
		return
	}

	// If the purge frequency is 0 then don't purge expired documents
	if client.purgeFrequency == 0 {
		return
	}

	for {
		// wait for the purge frequency duration
		time.Sleep(time.Duration(client.purgeFrequency) * time.Second)

		// Try to remove expired actors
		criteria := exp.LessThan("expires", time.Now().Unix())

		if _, err := client.collection.DeleteMany(context.Background(), criteria); err != nil {
			derp.Report(derp.Wrap(err, "ascache.Client.start", "Error purging expired documents from cache"))
		}
	}
}

/******************************************
 * Hannibal HTTP Client Methods
 ******************************************/

func (client *Client) Load(uri string, options ...any) (streams.Document, error) {

	config := NewLoadConfig(options...)

	// If we're not forcing the cache to reload, then try to load from the cache first
	if client.IsReadable() && config.isCacheAllowed() {

		// Search the cache for the document
		value := NewValue()

		if err := client.loadByURLs(uri, &value); err == nil {

			// If we're allowed to write to the cache, then do it.
			if client.IsWritable() && value.ShouldRevalidate() {
				go client.revalidate(uri, options...)
			}

			return client.asDocument(value), nil
		}
	}

	// Pass the request to the inner client
	result, err := client.innerClient.Load(uri, options...)

	if err != nil {
		return result, derp.Wrap(err, "ascache.Client.Load", "Error loading document from inner client", uri)
	}

	// Try to save the new value asynchronously
	if client.IsWritable() {
		client.save(uri, result)
	}

	return result, nil
}

/******************************************
 * Other Cache Management Methods
 ******************************************/

// Delete removes a single document from the cache
func (client *Client) Delete(url string) error {

	const location = "ascache.Client.Delete"

	// NPE Check
	if client.collection == nil {
		return derp.NewInternalError(location, "Document Collection not initialized")
	}

	// Look for the document in the cache
	value, err := client.Load(url)

	// If there's nothing in the cache, then there's nothing to delete
	if derp.NotFound(err) {
		return nil
	}

	// Return actual errors to the caller
	if err != nil {
		return derp.Wrap(err, location, "Error loading document", url)
	}

	// Delete the document from the cache
	criteria := bson.M{"urls": url}

	if _, err := client.collection.DeleteOne(context.Background(), criteria); err != nil {
		derp.Report(derp.Wrap(err, location, "Error purging expired actors from cache"))
	}

	// Recalculate statistics
	if err := client.calcStatistics(value); err != nil {
		return derp.Wrap(err, location, "Error calculating statistics", url)
	}

	// Success!
	return nil
}

// revalidate reloads a document from the source even if it has not yet expired.
// This potentially updates the cache timeout value, keeping the document
// fresh in the cache for longer.
func (client *Client) revalidate(url string, options ...any) {

	// If the client is not writable, then don't try to refresh the cache
	if client.NotWritable() {
		return
	}

	// Pass the request to the inner client
	log.Trace().Str("url", url).Msg("ascache.Client.revalidate")
	if result, err := client.innerClient.Load(url, options...); err == nil {
		client.save(url, result)
	}
}

// save adds/updates a document in the cache
func (client *Client) save(url string, document streams.Document) {

	const location = "ascache.Client.save"

	// If the client is not writable, then don't try to save the document
	if client.NotWritable() {
		return
	}

	// Write to trace log
	log.Trace().Str("url", url).Msg(location)

	// Calculate caching rules and exit if cache is not allowed.
	cacheControl := cacheheader.Parse(document.HTTPHeader())
	if client.obeyHeaders && cacheControl.NotCacheAllowed() {
		log.Trace().Str("url", url).Msg("Cache not allowed by HTTP headers. Skipping save method.")
		return
	}

	// Try to load an existing/duplicate values using the object.id field.
	// There may be multiple URLs that point to the same document, so we're
	// doing this check HERE using the object.id field.
	value := NewValue()

	if err := client.loadByID(document.ID(), &value); !derp.NilOrNotFound(err) {
		derp.Report(derp.Wrap(err, location, "Error searching for duplicate document in cache", document))
		return
	}

	// Add the document.id to the list of URLs (avoiding duplicates)
	if !value.URLs.Contains(document.ID()) {
		value.URLs.Append(document.ID())
	}

	// Create a new value
	value.URLs = []string{url}
	value.Object = document.Map()
	value.HTTPHeader = document.HTTPHeader()
	value.HTTPHeader.Set(HeaderHannibalCache, "true")
	value.HTTPHeader.Set(HeaderHannibalCacheDate, time.Now().Format(time.RFC3339))

	// Additional metadata
	value.Received = time.Now().Unix()
	value.calcPublished()
	value.calcExpires(cacheControl)
	value.calcRevalidates(cacheControl)
	value.calcRelationType(document)
	value.calcDocumentType(document)

	// Try to upsert the document into the cache
	filter := bson.M{"object.id": value.Object.GetString("id")}
	update := bson.M{"$set": value}
	queryOptions := options.Update().SetUpsert(true)

	if _, err := client.collection.UpdateOne(context.Background(), filter, update, queryOptions); err != nil {
		derp.Report(derp.Wrap(err, location, "Error saving document to cache", document.ID()))
		return
	}

	// Finally, try to recalculate statistics of linked documents
	if err := client.calcStatistics(document); err != nil {
		derp.Report(derp.Wrap(err, location, "Error calculating statistics", document.ID()))
	}
}

// asDocument converts a Document into a fully-populated streams.Document
func (client *Client) asDocument(value Value) streams.Document {

	return streams.NewDocument(
		value.Object,
		streams.WithClient(client),
		streams.WithStats(value.Statistics),
		streams.WithHTTPHeader(value.HTTPHeader),
	)
}

/******************************************
 * Other Queries
 ******************************************/

// load loads a Value from the cache using any criteria expression.
func (client *Client) load(criteria bson.M, value *Value) error {

	const location = "ascache.Client.load"

	// Prevent NPE
	if client.collection == nil {
		return derp.NewInternalError(location, "Cache connection is not defined")
	}

	// Query the cache database
	if err := client.collection.FindOne(context.Background(), criteria).Decode(value); err != nil {
		if err == mongo.ErrNoDocuments {
			return derp.NewNotFoundError(location, "Document not found", criteria)
		}
		return derp.Wrap(err, location, "Error loading document", criteria)
	}

	// Success.
	return nil
}

// loadByURLs loads a Value from the cache using its URL.
// This value can match any of the URLs in the "urls" array.
func (client *Client) loadByURLs(url string, value *Value) error {
	if err := client.load(bson.M{"urls": url}, value); err != nil {
		return derp.Wrap(err, "ascache.Client.loadByURLs", "Error loading document", url)
	}
	return nil
}

// loadByID loads a Value from the cache using its document.ID.
func (client *Client) loadByID(id string, value *Value) error {
	if err := client.load(bson.M{"object.id": id}, value); err != nil {
		return derp.Wrap(err, "ascache.Client.loadByID", "Error loading document", id)
	}
	return nil
}

/******************************************
 * Configuration Accessors
 ******************************************/

// IsReadWritable returns TRUE if the cache can be read and written
func (client *Client) IsReadWritable() bool {
	return client.cacheMode == CacheModeReadWrite
}

// NotReadWritable returns TRUE if the cache cannot be read or written
func (client *Client) NotReadWritable() bool {
	return client.cacheMode != CacheModeReadWrite
}

// IsReadable returns TRUE if the client is configured to read from the cache
func (client *Client) IsReadable() bool {
	return client.cacheMode != CacheModeWriteOnly
}

// NotReadable returns TRUE if the client is not configured to read from the cache
func (client *Client) NotReadable() bool {
	return client.cacheMode == CacheModeWriteOnly
}

// isWritable returns TRUE if the client is configured to write to the cache
func (client *Client) IsWritable() bool {
	return client.cacheMode != CacheModeReadOnly
}

// NotWritable returns TRUE if the client is not configured to write to the cache
func (client *Client) NotWritable() bool {
	return client.cacheMode == CacheModeReadOnly
}
