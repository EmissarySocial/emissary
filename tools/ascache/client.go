package ascache

import (
	"time"

	"github.com/benpate/cachecontrol"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/rosetta/mapof"
)

type Client struct {
	session             data.Session
	innerClient         streams.Client
	defaultCacheSeconds int
	minCacheSeconds     int
	maxCacheSeconds     int
	purgeFrequency      int64
	cacheMode           string
}

// New returns a fully initialized Client object
func New(session data.Session, innerClient streams.Client, options ...OptionFunc) *Client {

	// Create a default client
	result := Client{
		session:         session,
		innerClient:     innerClient,
		minCacheSeconds: 60 * 60 * 24 * 30,  // Default minimum expiration is 30 days
		maxCacheSeconds: 60 * 60 * 24 * 365, // Default maximum expiration is 365 days
		purgeFrequency:  60 * 60 * 4,        // Default purge frequency is 4 hours
		cacheMode:       CacheModeReadWrite,
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
	if client.notWritable() {
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
			derp.Report(derp.Wrap(err, "cache.Client.delete", "Error purging expired actors from cache"))
		}

		// Try to remove expired documents
		if err := client.session.Collection(CollectionDocument).HardDelete(criteria); err != nil {
			derp.Report(derp.Wrap(err, "cache.Client.delete", "Error purging expired documents from cache"))
		}
	}
}

/******************************************
 * Hannibal HTTP Client Methods
 ******************************************/

func (client *Client) LoadActor(uri string) (streams.Document, error) {

	// Search the cache for the document
	cachedValue := NewCachedValue()
	if err := client.loadByURI(CollectionActor, uri, &cachedValue); err == nil {

		if cachedValue.ShouldRefresh() {
			go client.refreshActor(CollectionActor, uri, cachedValue)
		}

		result := client.asDocument(cachedValue)
		return result, nil
	}

	// Pass the request to the inner client
	result, err := client.innerClient.LoadActor(uri)

	if err != nil {
		return result, derp.Wrap(err, "cache.Client.Load", "error loading document from inner client", uri)
	}

	result.WithOptions(streams.WithClient(client))

	// Try to save the new value asynchronously
	if client.isWritable() {
		go client.save(CollectionActor, uri, result)
	}

	return result, nil
}

func (client *Client) LoadDocument(uri string, defaultValue map[string]any) (streams.Document, error) {

	// Search the cache for the document
	cachedValue := NewCachedValue()
	if err := client.loadByURI(CollectionDocument, uri, &cachedValue); err == nil {

		if cachedValue.ShouldRefresh() {
			go client.refreshDocument(CollectionDocument, uri, cachedValue)
		}

		return client.asDocument(cachedValue), nil
	}

	// Pass the request to the inner client
	result, err := client.innerClient.LoadDocument(uri, defaultValue)

	if err != nil {
		return result, derp.Wrap(err, "cache.Client.Load", "error loading document from inner client", uri)
	}

	result.WithOptions(streams.WithClient(client))

	// Try to save the new value asynchronously
	if client.isWritable() {
		go client.save(CollectionDocument, uri, result)
	}

	return result, nil
}

/******************************************
 * Other Cache Management Methods
 ******************************************/

func (client *Client) PurgeByURI(collection string, uri string) error {

	// If the client is not writable then don't try to purge the cache
	if client.notWritable() {
		return nil
	}

	// Try to purge the cache
	if err := client.session.Collection(collection).HardDelete(exp.Equal("uri", uri)); err != nil {
		return derp.Wrap(err, "cache.Client.delete", "Error deleting document from cache (by URI)", uri)
	}

	// Woot woot
	return nil
}

func (client *Client) refreshActor(collection string, uri string, value CachedValue) {

	// If the client is not writable, then don't try to refresh the cache
	if client.notWritable() {
		return
	}

	// Pass the request to the inner client
	if result, err := client.innerClient.LoadActor(uri); err == nil {
		client.save(collection, uri, result)
	}
}

func (client *Client) refreshDocument(collection string, uri string, value CachedValue) {

	// If the client is not writable, then don't try to refresh the cache
	if client.notWritable() {
		return
	}

	// Pass the request to the inner client
	if result, err := client.innerClient.LoadDocument(uri, mapof.NewAny()); err == nil {
		client.save(collection, uri, result)
	}
}

func (client *Client) save(collection string, uri string, document streams.Document) {

	// If the client is not writable, then don't try to save the document
	if client.notWritable() {
		return
	}

	// Use response headers to if we can cache this document
	expireSeconds := client.calcExpireSeconds(document.MetaString("cache-control"))

	if expireSeconds == 0 {
		return
	}

	// Create a new cachedValue
	cachedValue := NewCachedValue()
	cachedValue.URI = uri
	cachedValue.Original = document.Map()
	cachedValue.Metadata = *document.Meta()

	if publishedDate := document.Published(); !publishedDate.IsZero() {
		cachedValue.PublishedDate = publishedDate.Unix()
	} else {
		cachedValue.PublishedDate = time.Now().Unix()
	}

	if inReplyTo := document.InReplyTo(); inReplyTo.NotNil() {
		cachedValue.InReplyTo = inReplyTo.String()
	}

	// Calculate caching rules
	cachedValue.ExpiresDate = time.Now().Add(time.Second * time.Duration(expireSeconds)).Unix()
	cachedValue.RefreshesDate = client.calcRefreshDate(expireSeconds)

	// Save the document to the cache
	if err := client.session.Collection(collection).Save(&cachedValue, ""); err != nil {
		derp.Report(derp.Wrap(err, "cache.Client.save", "Error saving document to cache", document.ID()))
	}
}

func (client *Client) asDocument(cachedValue CachedValue) streams.Document {
	result := streams.NewDocument(
		cachedValue.Original,
		streams.WithClient(client),
	)

	result.MetaSet(cachedValue.Metadata)

	for key, value := range cachedValue.ResponseCounts {
		result.MetaSetInt(key, value)
	}

	return result
}

// calcExpireSeconds calculates the number of seconds to cache this document
func (client *Client) calcExpireSeconds(cacheString string) int {

	parsed := cachecontrol.Parse(cacheString)

	// If we're told not to cache, then don't cache
	// though we technically could, but there's no "revalidation" yet, so just screw it.
	// TODO: LOW: possibly implement revalidation per:
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Cache-Control#no-cache
	if ok, _ := parsed.NoCache(); ok {
		return 0
	}

	// If we're told not to store, then don't cache
	if parsed.NoStore() {
		return 0
	}

	// TODO: handle the "age" header for more precise caching:
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Age
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Cache-Control#max-age

	// By default, try to cache for the maximum duration
	result := client.defaultCacheSeconds

	// If a max-age is defined, then use that value
	if maxAge := parsed.MaxAge(); maxAge > 0 {
		result = int(maxAge.Seconds())
	}

	// Don't cache for any duration less than the minimum
	if result > client.maxCacheSeconds {
		result = client.maxCacheSeconds
	}

	// Don't cache for any duration less than the maximum
	if result < client.minCacheSeconds {
		result = client.minCacheSeconds
	}

	// This should be the "acceptable" amount of time to cache the document.
	return result
}

// calcRefreshDuration calculates the number of seconds to wait before "refreshing" a document.
// "Refreshing" means to continue using the cached document, but start a background process to
// update the cached value anyway.  This is currently set at 1/3 of the original cache duration, so
// it should mean two (at most) extra HTTP calls compared to caching the document for the full duration.
func (client *Client) calcRefreshDate(cacheSeconds int) int64 {
	refreshSeconds := cacheSeconds / 3
	return time.Now().Add(time.Duration(refreshSeconds) * time.Second).Unix()
}

/******************************************
 * Other Queries
 ******************************************/

func (client *Client) loadByURI(collection string, uri string, document *CachedValue) error {

	if client.session == nil {
		return derp.NewInternalError("cache.Client.loadByURI", "Cache connection is not defined")
	}

	criteria := exp.Equal("uri", uri)

	if err := client.session.Collection(collection).Load(criteria, document); err != nil {
		return derp.Wrap(err, "cache.Client.loadByURI", "Error loading document from cache", uri)
	}

	return nil
}

// isReadable returns TRUE if the client is configured to read from the cache
func (client *Client) isReadable() bool {
	return client.cacheMode != CacheModeWriteOnly
}

// notReadable returns TRUE if the client is not configured to read from the cache
func (client *Client) notReadable() bool {
	return client.cacheMode == CacheModeWriteOnly
}

// isWritable returns TRUE if the client is configured to write to the cache
func (client *Client) isWritable() bool {
	return client.cacheMode != CacheModeReadOnly
}

// notWritable returns TRUE if the client is not configured to write to the cache
func (client *Client) notWritable() bool {
	return client.cacheMode == CacheModeReadOnly
}
