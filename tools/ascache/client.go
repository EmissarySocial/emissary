package ascache

import (
	"time"

	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/rosetta/mapof"
)

type Client struct {
	session        data.Session
	innerClient    streams.Client
	expireSeconds  int64
	purgeFrequency int64
}

// New returns a fully initialized Client object
func New(session data.Session, innerClient streams.Client, options ...OptionFunc) *Client {

	// Create a default client
	result := Client{
		session:        session,
		innerClient:    innerClient,
		expireSeconds:  60 * 60 * 24 * 30, // Default expiration is 30 days
		purgeFrequency: 60 * 60 * 4,       // Default purge frequency is 4 hours
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

	if client.session == nil {
		return
	}

	for client.purgeFrequency > 0 {
		time.Sleep(time.Second * time.Duration(client.purgeFrequency))

		if err := client.session.Collection(CollectionActors).HardDelete(exp.LessThan("exp", time.Now().Unix())); err != nil {
			// nolint: errcheck
			derp.Report(derp.Wrap(err, "cache.Client.delete", "Error purging expired actors from cache"))
		}

		if err := client.session.Collection(CollectionDocuments).HardDelete(exp.LessThan("exp", time.Now().Unix())); err != nil {
			// nolint: errcheck
			derp.Report(derp.Wrap(err, "cache.Client.delete", "Error purging expired documents from cache"))
		}
	}
}

/******************************************
 * Hannibal HTTP Client Methods
 ******************************************/

func (client *Client) LoadActor(uri string) (streams.Document, error) {

	// Search the cache for the document
	if client.session != nil {
		cachedValue := NewCachedValue()
		if err := client.loadByURI(CollectionActors, uri, &cachedValue); err == nil {

			if cachedValue.ShouldRefresh() {
				go client.refresh(CollectionActors, uri, cachedValue)
			}

			result := client.asDocument(cachedValue)
			result.MetaSet(cachedValue.Metadata)

			return result, nil
		}
	}

	// Pass the request to the inner client
	result, err := client.innerClient.LoadActor(uri)

	if err != nil {
		return result, derp.Wrap(err, "cache.Client.Load", "error loading document from inner client", uri)
	}

	// Try to save the new value asynchronously
	if client.session != nil {
		go client.save(CollectionActors, uri, result)
	}

	result.WithOptions(streams.WithClient(client))

	return result, nil
}

func (client *Client) LoadDocument(uri string, defaultValue map[string]any) (streams.Document, error) {

	// Search the cache for the document
	if client.session != nil {
		cachedValue := NewCachedValue()
		if err := client.loadByURI(CollectionDocuments, uri, &cachedValue); err == nil {

			if cachedValue.ShouldRefresh() {
				go client.refresh(CollectionDocuments, uri, cachedValue)
			}

			return client.asDocument(cachedValue), nil
		}
	}

	// Pass the request to the inner client
	result, err := client.innerClient.LoadDocument(uri, defaultValue)

	if err != nil {
		return result, derp.Wrap(err, "cache.Client.Load", "error loading document from inner client", uri)
	}

	// Try to save the new value asynchronously
	if client.session != nil {
		go client.save(CollectionDocuments, uri, result)
	}

	result.WithOptions(streams.WithClient(client))

	return result, nil
}

/******************************************
 * Other Cache Management Methods
 ******************************************/

func (client *Client) PurgeByURI(collection string, uri string) error {

	if client.session == nil {
		return derp.NewInternalError("cache.Client.delete", "Cache connection is not defined")
	}

	if err := client.session.Collection(collection).HardDelete(exp.Equal("uri", uri)); err != nil {
		return derp.Wrap(err, "cache.Client.delete", "Error deleting document from cache (by URI)", uri)
	}

	return nil
}

func (client *Client) refresh(collection string, uri string, value CachedValue) {

	if client.session == nil {
		return
	}

	// Pass the request to the inner client
	if result, err := client.innerClient.LoadDocument(uri, mapof.NewAny()); err == nil {
		client.save(collection, uri, result)
	}
}

func (client *Client) save(collection string, uri string, document streams.Document) {

	if client.session == nil {
		return
	}

	// Create a new cachedValue
	cachedValue := NewCachedValue()
	cachedValue.URI = uri
	cachedValue.Original = document.Map()
	cachedValue.PublishedDate = document.Published().Unix()

	// TODO: LOW: should see if the document has a header that defines the cache duration
	cachedValue.ExpiresDate = time.Now().Add(time.Second * time.Duration(client.expireSeconds)).Unix()
	cachedValue.RefreshesDate = CalcRefreshDate(time.Now().Unix(), cachedValue.ExpiresDate)

	if inReplyTo := document.InReplyTo(); inReplyTo.NotNil() {
		cachedValue.InReplyTo = inReplyTo.String()
	}

	// Save it to the cache
	if err := client.session.Collection(collection).Save(&cachedValue, ""); err != nil {
		// nolint: errcheck // derp.Report has us covered.
		derp.Report(derp.Wrap(err, "cache.Client.save", "Error saving document to cache", document.ID()))
	}

	// If this is a reply, then cache the parent document as well
	if cachedValue.InReplyTo != "" {
		// nolint: errcheck // This is just a pre-emptive load, so we don't care if it fails
		go client.LoadDocument(cachedValue.InReplyTo, mapof.NewAny())
	}
}

func (client *Client) asDocument(cachedValue CachedValue) streams.Document {
	result := streams.NewDocument(
		cachedValue.Original,
		streams.WithClient(client),
	)

	for key, value := range cachedValue.ResponseCounts {
		result.MetaSetInt(key, value)
	}

	return result
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
