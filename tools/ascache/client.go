package ascache

import (
	"context"
	"time"

	"github.com/EmissarySocial/emissary/tools/cacheheader"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/streams"
	"github.com/rs/zerolog/log"
)

type Client struct {
	commonDatabase data.Server
	innerClient    streams.Client
	cacheMode      string
	obeyHeaders    bool
}

// New returns a fully initialized Client object
func New(innerClient streams.Client, commonDatabase data.Server, options ...ClientOptionFunc) *Client {

	// Create a default client
	result := &Client{
		commonDatabase: commonDatabase,
		innerClient:    innerClient,
		cacheMode:      CacheModeReadWrite,
		obeyHeaders:    true,
	}

	// Apply option functions to the client
	result.WithOptions(options...)
	result.innerClient.SetRootClient(result)
	return result
}

func (client *Client) WithOptions(options ...ClientOptionFunc) {
	for _, option := range options {
		option(client)
	}
}

/******************************************
 * Hannibal HTTP Client Methods
 ******************************************/

func (client *Client) SetRootClient(rootClient streams.Client) {
	client.innerClient.SetRootClient(rootClient)
}

func (client *Client) Load(url string, options ...any) (streams.Document, error) {

	const location = "tools.ascache.client.Load"

	config := NewLoadConfig(options...)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.timeoutSeconds)*time.Second)
	defer cancel()

	collection, err := client.collection(ctx)

	if err != nil {
		return streams.NilDocument(), derp.Wrap(err, location, "Unable to connect to database")
	}

	// If we're not forcing the cache to reload, then try to load from the cache first
	if client.IsReadable() && config.isCacheAllowed() {
		// Search the cache for the document
		value := NewValue()

		if err := client.loadByURLs(collection, url, &value); err == nil {

			// If we're allowed to write to the cache, then do it.
			if client.IsWritable() && value.ShouldRevalidate() {
				go derp.Report(client.revalidate(url, options...))
			}

			return client.asDocument(value), nil
		}
	}

	// Pass the request to the inner client
	result, err := client.innerClient.Load(url, options...)

	if err != nil {

		// If the original document is gone, and we're forcing a reload, then remove the value from the cache
		if derp.IsNotFound(err) && config.forceReload {
			if err := client.Delete(url); err != nil {
				return result, derp.Wrap(err, location, "Error removing document from cache", url)
			}
		}

		return result, derp.Wrap(err, location, "Error loading document from inner client", url)
	}

	// Try to save the new value asynchronously
	if client.IsWritable() {
		if err := client.save(collection, url, result); err != nil {
			return result, derp.Wrap(err, location, "Error writing document to cache")
		}
	}

	return result, nil
}

/******************************************
 * Other Cache Management Methods
 ******************************************/

func (client *Client) Put(document streams.Document) error {

	const location = "tools.ascache.client.Put"

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	collection, err := client.collection(ctx)

	if err != nil {
		return derp.Wrap(err, location, "Unable to connect to database")
	}

	if err := client.save(collection, document.ID(), document); err != nil {
		return derp.Wrap(err, location, "Unable to put document into cache")
	}

	return nil
}

// Delete removes a single document from the cache
func (client *Client) Delete(url string) error {

	const location = "ascache.Client.Delete"

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection, err := client.collection(ctx)

	if err != nil {
		return derp.InternalError(location, "Unable to connect to ActivityStream cache")
	}

	// Load the document from the database (to recalculate statistics after delete)
	value, err := client.Load(url)

	if derp.IsNotFound(err) {
		return nil
	}

	if err != nil {
		return derp.Wrap(err, location, "Unable to load cached ActivityStream document", url)
	}

	// Delete the document from the cache
	criteria := exp.Equal("urls", url)

	if err := collection.HardDelete(criteria); err != nil {
		return derp.Wrap(err, location, "Unable to delete", url)
	}

	// Recalculate statistics
	if err := client.calcStatistics(collection, value); err != nil {
		return derp.Wrap(err, location, "Error calculating statistics", url)
	}

	// Success!
	return nil
}

// revalidate reloads a document from the source even if it has not yet expired.
// This potentially updates the cache timeout value, keeping the document
// fresh in the cache for longer.
func (client *Client) revalidate(url string, options ...any) error {

	const location = "tools.ascache.client.revalidate"

	// If the client is not writable, then don't try to refresh the cache
	if client.NotWritable() {
		return nil
	}

	// Pass the request to the inner client
	log.Trace().Str("loc", location).Str("url", url).Msg("Reloading URL")

	result, err := client.innerClient.Load(url, options...)

	if err != nil {
		return derp.Wrap(err, location, "Error loading document from inner client", url)
	}

	// Connect to the database
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	collection, err := client.collection(ctx)

	if err != nil {
		return derp.Wrap(err, location, "Error connecting to database", url)
	}

	// Save the updated document
	if err := client.save(collection, url, result); err != nil {
		return derp.Wrap(err, location, "Unable to save revalidated document", url)
	}

	return nil
}

/******************************************
 * Database Methods
 ******************************************/

func (client *Client) collection(ctx context.Context) (data.Collection, error) {

	const location = "tools.ascache.Client.collection"

	if client.commonDatabase == nil {
		return nil, derp.InternalError(location, "Common Database is not initialized")
	}

	session, err := client.commonDatabase.Session(ctx)

	if err != nil {
		return nil, derp.Wrap(err, location, "Unable to connect to common database")
	}

	return session.Collection("Document"), nil
}

// save adds/updates a document in the cache
func (client *Client) save(collection data.Collection, url string, document streams.Document) error {

	const location = "ascache.Client.save"

	// RULE: If the client is not writable, then don't try to save the document
	if client.NotWritable() {
		return nil
	}

	// Write to trace log
	log.Trace().Str("url", url).Msg(location)

	// Calculate caching rules and exit if cache is not allowed.
	cacheControl := cacheheader.Parse(document.HTTPHeader())
	if client.obeyHeaders && cacheControl.NotCacheAllowed() {
		log.Trace().Str("url", url).Msg("Cache not allowed by HTTP headers. Skipping save method.")
		return nil
	}

	// Try to load an existing/duplicate values using the object.id field.
	// There may be multiple URLs that point to the same document, so we're
	// doing this check HERE using the object.id field.

	value := NewValue()
	value.URLs.Append(url)

	if err := client.loadByURLs(collection, url, &value); !derp.IsNilOrNotFound(err) {
		return derp.Wrap(err, location, "Error searching for duplicate document in cache", document)
	}

	// Add the document.id and url to the list of URLs (avoiding duplicates)
	for _, item := range []string{document.ID(), url} {
		if item == "" {
			continue
		}

		if value.URLs.Contains(item) {
			continue
		}

		value.URLs.Append(item)
	}

	// Create a new value
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
	if err := collection.Save(&value, "updated"); err != nil {
		return derp.Wrap(err, location, "Unable to save cached value", url)
	}

	// Finally, try to recalculate statistics of linked documents
	if err := client.calcStatistics(collection, document); err != nil {
		return derp.Wrap(err, location, "Unable to calculate statistics", url)
	}

	// Success.
	return nil
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
func (client *Client) load(ctx context.Context, criteria exp.Expression, value *Value) error {

	const location = "ascache.Client.load"

	// Get the database connection
	collection, err := client.collection(ctx)

	if err != nil {
		return derp.Wrap(err, location, "Unable to connect to database")
	}

	// Query the cache database
	if err := collection.Load(criteria, value); err != nil {
		return derp.Wrap(err, location, "Unable to load cached value", criteria)
	}

	// Success.
	return nil
}

// loadByURLs loads a Value from the cache using its URL.
// This value can match any of the URLs in the "urls" array.
func (client *Client) loadByURLs(collection data.Collection, url string, value *Value) error {
	criteria := exp.Equal("urls", url)
	return collection.Load(criteria, value)
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
