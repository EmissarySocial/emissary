package ascache

import (
	"time"

	"github.com/EmissarySocial/emissary/tools/cacheheader"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/streams"
)

type Client struct {
	session        data.Session
	innerClient    streams.Client
	purgeFrequency int64
	cacheMode      string
}

// New returns a fully initialized Client object
func New(session data.Session, innerClient streams.Client, options ...OptionFunc) *Client {

	// Create a default client
	result := Client{
		session:        session,
		innerClient:    innerClient,
		purgeFrequency: 60 * 60 * 4, // Default purge frequency is 4 hours
		cacheMode:      CacheModeReadWrite,
	}

	// Apply option functions to the client
	for _, option := range options {
		option(&result)
	}

	go result.start()

	return &result
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
		time.Sleep(time.Second * time.Duration(client.purgeFrequency))

		criteria := exp.LessThan("expires", time.Now().Unix())

		// Try to remove expired actors
		if err := client.session.Collection(CollectionActor).HardDelete(criteria); err != nil {
			derp.Report(derp.Wrap(err, "ascache.Client.delete", "Error purging expired actors from cache"))
		}

		// Try to remove expired documents
		if err := client.session.Collection(CollectionDocument).HardDelete(criteria); err != nil {
			derp.Report(derp.Wrap(err, "ascache.Client.delete", "Error purging expired documents from cache"))
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
		cachedValue := NewCachedValue()
		if err := client.loadByURI(CollectionActor, uri, &cachedValue); err == nil {

			// If we're allowed to write to the cache, then do it.
			if client.IsWritable() && cachedValue.ShouldRevalidate() {
				go client.revalidate(uri, options...)
			}

			result := client.asDocument(cachedValue)
			return result, nil
		}
	}

	// Pass the request to the inner client
	result, err := client.innerClient.Load(uri, options...)

	if err != nil {
		return result, derp.Wrap(err, "ascache.Client.LoadActor", "error loading document from inner client", uri)
	}

	result.WithOptions(streams.WithClient(client))

	// Try to save the new value asynchronously
	if client.IsWritable() {
		go client.save(CollectionActor, uri, result)
	}

	return result, nil
}

func (client *Client) PurgeByURI(collection string, uri string) error {

	// If the client is not writable then don't try to purge the cache
	if client.NotWritable() {
		return nil
	}

	// Try to purge the cache
	if err := client.session.Collection(collection).HardDelete(exp.Equal("uri", uri)); err != nil {
		return derp.Wrap(err, "ascache.Client.delete", "Error deleting document from cache (by URI)", uri)
	}

	// Woot woot
	return nil
}

/******************************************
 * Other Cache Management Methods
 ******************************************/

// revalidate reloads a document from the source even if it has not yet expired.
// This potentially updates the cache timeout value, keeping the document
// fresh in the cache for longer.
func (client *Client) revalidate(uri string, options ...any) {

	// If the client is not writable, then don't try to refresh the cache
	if client.NotWritable() {
		return
	}

	// Pass the request to the inner client
	if result, err := client.innerClient.Load(uri, options...); err == nil {
		client.save(CollectionActor, uri, result)
	}
}

func (client *Client) save(collection string, uri string, document streams.Document) {

	// If the client is not writable, then don't try to save the document
	if client.NotWritable() {
		return
	}

	// Create a new cachedValue
	cachedValue := NewCachedValue()
	cachedValue.URI = uri
	cachedValue.Original = document.Map()
	cachedValue.HTTPHeader = document.HTTPHeader()

	if inReplyTo := document.InReplyTo(); inReplyTo.NotNil() {
		cachedValue.InReplyTo = inReplyTo.String()
	}

	// Calculate caching rules
	cacheControl := cacheheader.Parse(cachedValue.HTTPHeader)

	if cacheControl.NotCacheAllowed() {
		return
	}

	cachedValue.calcPublished()
	cachedValue.calcExpires(cacheControl)
	cachedValue.calcRevalidates(cacheControl)

	// Try to remove any existing documents with the same URI
	if err := client.session.Collection(collection).HardDelete(exp.Equal("uri", uri)); err != nil {
		derp.Report(derp.Wrap(err, "ascache.Client.save", "Error deleting document from cache (by URI)", uri))
	}

	// Save the document to the cache
	if err := client.session.Collection(collection).Save(&cachedValue, ""); err != nil {
		derp.Report(derp.Wrap(err, "ascache.Client.save", "Error saving document to cache", document.ID()))
	}
}

func (client *Client) asDocument(cachedValue CachedValue) streams.Document {
	result := streams.NewDocument(
		cachedValue.Original,
		streams.WithClient(client),
		streams.WithHTTPHeader(cachedValue.HTTPHeader),
	)

	/*
		for key, value := range cachedValue.ResponseCounts {
			result.MetaSetInt(key, value)
		}
	*/

	return result
}

/******************************************
 * Other Queries
 ******************************************/

func (client *Client) loadByURI(collection string, uri string, document *CachedValue) error {

	if client.session == nil {
		return derp.NewInternalError("ascache.Client.loadByURI", "Cache connection is not defined")
	}

	criteria := exp.Equal("uri", uri)

	if err := client.session.Collection(collection).Load(criteria, document); err != nil {
		return derp.Wrap(err, "ascache.Client.loadByURI", "Error loading document from cache", uri)
	}

	return nil
}

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
